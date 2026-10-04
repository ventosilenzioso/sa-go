// Command gosamp-bench benchmarks a running gosamp-server over real UDP.
//
// Each bot opens its own UDP socket (distinct local port, which is how the
// server tells clients apart), performs the full handshake
// (cookie/auth/join/class/spawn) as BotNNN, then sends --chat-per-client chat
// lines and measures the round-trip time of its own echoes.
//
// Latency histograms hold milliseconds (see internal/bench.Histogram, whose
// unit is defined at the use site).
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gosamp/internal/bench"
	"gosamp/internal/raknet"
	"gosamp/internal/samp/protocol"
	"gosamp/pkg/netsim"
)

var (
	serverFlag = flag.String("server", "127.0.0.1:7777", "server address host:port")
	clients    = flag.Int("clients", 20, "number of concurrent bots")
	chatPer    = flag.Int("chat-per-client", 5, "chat lines each bot sends")
	timeout    = flag.Duration("timeout", 30*time.Second, "overall run timeout")
	csvPath    = flag.String("csv", "", "optional CSV output path (empty = no file)")
	// ramp runs escalating client counts (comma-separated) to find the
	// server's breaking point; e.g. -ramp 50,100,200,400.
	ramp        = flag.String("ramp", "", "escalating bot counts, comma-separated (replaces -clients)")
	netProfile  = flag.String("net", "clean", "network profile: clean|mobile3g|badmobile|flaky")
	soak        = flag.Duration("soak", 0, "run soak mode for this long (keeps bots connected, logs RSS/goroutines)")
	soakMaxRTTs = flag.Int("soak-max-rtts", 4096, "cap on retained per-bot RTT samples during soak")
	metricsURL  = flag.String("server-metrics", "", "URL of the server's expvar endpoint (e.g. http://127.0.0.1:6060/debug/vars) for soak sampling")
	memGuard    = flag.Bool("memguard", true, "halt ramp automatically if system memory is critically low or CPU saturated")
	// stagger spaces bot launches by this delay so a large stage does not
	// present a single thundering-herd of simultaneous cold handshakes from
	// one host. Real player joins are naturally spread out over time.
	stagger = flag.Duration("stagger", 0, "delay between bot launches (e.g. 20ms) to avoid a synthetic handshake thundering herd")
	// soakChatEvery controls how often each bot sends a chat line during soak.
	// Every chat is broadcast to all N bots, so the aggregate load is O(N^2):
	// the 2s default is fine for 50 bots but swamps a 1000-bot hold, where a
	// longer interval keeps the test measuring the server rather than the
	// harness's own echo-processing capacity.
	soakChatEvery = flag.Duration("soak-chat-every", 2*time.Second, "chat interval per bot during soak (increase for large bot counts)")
)

type botClient struct {
	pc   net.PacketConn
	srv  *net.UDPAddr
	port uint16
	conn *raknet.Conn
	now  time.Time
	ctx  context.Context
	sim  *netsim.Sim // nil = clean network (no simulation)
	// simOut buffers packets whose netsim delivery time has arrived.
	simOut   [][]byte
	simTimes []time.Time
}

func (c *botClient) recvTimeout(d time.Duration) ([]byte, error) {
	_ = c.pc.SetReadDeadline(time.Now().Add(d))
	buf := make([]byte, 2048)
	for {
		select {
		case <-c.ctx.Done():
			return nil, c.ctx.Err()
		default:
		}
		n, addr, err := c.pc.ReadFrom(buf)
		if err != nil {
			return nil, err
		}
		if addr.String() != c.srv.String() {
			continue
		}
		return buf[:n], nil
	}
}

// exchange flushes the reliability queue to the server and feeds every
// server datagram back into the connection. It drains with a fixed 300 ms
// deadline; prefer exchangeUntil when waiting for a specific reply.
func (c *botClient) exchange() ([][]byte, error) {
	return c.exchangeDeadline(300 * time.Millisecond)
}

// exchangeUntil flushes the reliability queue once, then reads datagrams with
// a single absolute deadline until match reports the awaited reply has arrived
// (or the deadline elapses). It stops the instant the reply is seen, so
// measured RTT reflects real latency instead of a forced read-drain window or
// a polling-timer floor.
func (c *botClient) exchangeUntil(match func([][]byte) bool, deadline time.Time) ([][]byte, error) {
	var all [][]byte
	buf := make([]byte, 2048)
	lastSend := time.Time{}
	c.sendPending()
	lastSend = time.Now()
	for {
		select {
		case <-c.ctx.Done():
			return all, c.ctx.Err()
		default:
		}
		// Re-flush reliability only occasionally (not every read tick) so we
		// do not exceed the server's per-peer datagram budget with pure ACK
		// chatter while waiting for a reply.
		if time.Since(lastSend) >= 30*time.Millisecond {
			c.sendPending()
			lastSend = time.Now()
		}
		next := time.Now().Add(5 * time.Millisecond)
		if next.After(deadline) {
			next = deadline
		}
		_ = c.pc.SetReadDeadline(next)
		n, addr, err := c.pc.ReadFrom(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if !time.Now().Before(deadline) {
					return all, nil
				}
				continue
			}
			return all, err
		}
		if addr.String() != c.srv.String() {
			continue
		}
		all = append(all, c.decode(buf[:n])...)
		if match != nil && match(all) {
			return all, nil
		}
	}
}

// sendPending flushes the reliability queue through the optional netsim.
func (c *botClient) sendPending() {
	select {
	case <-c.ctx.Done():
		return
	default:
	}
	c.now = time.Now()
	for _, f := range c.conn.Tick(c.now) {
		wire := raknet.EncryptDatagram(f, uint16(c.srv.Port))
		if c.sim != nil {
			if pkts, ats := c.sim.Send(c.now, wire); pkts != nil {
				for i, p := range pkts {
					deliverAt := ats[i]
					if deliverAt.After(c.now) {
						c.simOut = append(c.simOut, p)
						c.simTimes = append(c.simTimes, deliverAt)
					} else {
						_, _ = c.pc.WriteTo(p, c.srv)
					}
				}
			}
			var keep [][]byte
			var keepT []time.Time
			for i := range c.simOut {
				if c.simTimes[i].After(c.now) {
					keep = append(keep, c.simOut[i])
					keepT = append(keepT, c.simTimes[i])
				} else {
					_, _ = c.pc.WriteTo(c.simOut[i], c.srv)
				}
			}
			c.simOut, c.simTimes = keep, keepT
		} else {
			_, _ = c.pc.WriteTo(wire, c.srv)
		}
	}
}

// decode turns one server datagram into decoded payloads (nil if none).
func (c *botClient) decode(raw []byte) [][]byte {
	if len(raw) <= 3 {
		if off := raknet.ParseOffline(raw); off.Kind != raknet.OfflineOther {
			return [][]byte{append([]byte(nil), raw...)}
		}
	}
	got, err := c.conn.HandleDatagram(raw, time.Now())
	if err != nil {
		return nil
	}
	var out [][]byte
	for _, p := range got {
		nbytes := (p.DataBitLength + 7) / 8
		out = append(out, append([]byte(nil), p.Data[:nbytes]...))
	}
	return out
}

// exchangeDeadline flushes the reliability queue and reads server datagrams
// for at most d, returning decoded payloads.
func (c *botClient) exchangeDeadline(d time.Duration) ([][]byte, error) {
	c.sendPending()
	_ = c.pc.SetReadDeadline(time.Now().Add(d))
	var out [][]byte
	buf := make([]byte, 2048)
	for {
		n, addr, err := c.pc.ReadFrom(buf)
		if err != nil {
			break // timeout: assume the server is done for now
		}
		if addr.String() != c.srv.String() {
			continue
		}
		out = append(out, c.decode(buf[:n])...)
	}
	return out, nil
}

func enqueue(c *botClient, payload []byte, rel byte) error {
	return c.conn.Enqueue(payload, rel, 0)
}

func waitRaw(c *botClient, id byte, d time.Duration) ([]byte, error) {
	var found []byte
	_, err := c.exchangeUntil(func(all [][]byte) bool {
		for _, p := range all {
			if len(p) > 0 && p[0] == id {
				found = p
				return true
			}
		}
		return false
	}, time.Now().Add(d))
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, fmt.Errorf("timeout waiting for packet %d", id)
	}
	return found, nil
}

func waitRPC(c *botClient, id byte, d time.Duration) ([]byte, error) {
	var found []byte
	_, err := c.exchangeUntil(func(all [][]byte) bool {
		for _, p := range all {
			if rid, body, err := protocol.ParseRPCFrame(p); err == nil && rid == id {
				found = body
				return true
			}
		}
		return false
	}, time.Now().Add(d))
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, fmt.Errorf("timeout waiting for RPC %d", id)
	}
	return found, nil
}

func buildJoin(name string, token uint32) []byte {
	bs := raknet.New()
	bs.WriteUint32(protocol.ClientVersion037)
	bs.WriteUint8(1)
	bs.WriteUint8(byte(len(name)))
	bs.WriteAlignedBytes([]byte(name))
	bs.WriteUint32(token ^ protocol.ClientVersion037)
	bs.WriteUint8(3)
	bs.WriteAlignedBytes([]byte("3E9"))
	bs.WriteUint8(5)
	bs.WriteAlignedBytes([]byte("0.3.7"))
	return bs.Bytes()[:bs.Len()]
}

// botResult is the per-bot outcome. Times are in milliseconds.
type botResult struct {
	index   int
	name    string
	spawnMs float64
	rtts    []float64 // per-echo RTTs in ms
	echoes  int
	err     error
}

func (r *botResult) rttP50() float64 {
	var h bench.Histogram
	for _, v := range r.rtts {
		h.Add(v)
	}
	return h.Quantile(0.5)
}

func (r *botResult) rttP95() float64 {
	var h bench.Histogram
	for _, v := range r.rtts {
		h.Add(v)
	}
	return h.Quantile(0.95)
}

func runBot(ctx context.Context, srv *net.UDPAddr, idx, chatN int, sim *netsim.Sim, spawnHist, rttHist *bench.Histogram, meter *bench.Meter, soakMode bool, alive *int64) botResult {
	name := fmt.Sprintf("Bot%03d", idx)
	res := botResult{index: idx, name: name}
	t0 := time.Now()

	pc, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		res.err = err
		return res
	}
	defer pc.Close()
	laddr := pc.LocalAddr().(*net.UDPAddr)
	c := &botClient{
		pc:   pc,
		srv:  srv,
		port: uint16(laddr.Port),
		conn: raknet.NewConn(raknet.Config{
			MessagesLimit: 100000,
			AcksLimit:     100000,
		}, time.Now()),
		now: time.Now(),
		ctx: ctx,
		sim: sim,
	}

	if err := handshake(c, name); err != nil {
		res.err = err
		return res
	}
	spawnMs := float64(time.Since(t0).Microseconds()) / 1000.0
	res.spawnMs = spawnMs
	spawnHist.Add(spawnMs)
	if alive != nil {
		atomic.AddInt64(alive, 1)
		defer atomic.AddInt64(alive, -1)
	}

	for i := 0; i < chatN; i++ {
		text := fmt.Sprintf("bench %s %d", name, i)
		rtt, err := chatRoundTrip(c, text)
		if err != nil {
			res.err = err
			return res
		}
		res.addRTT(rtt)
		rttHist.Add(rtt)
		meter.Mark(1)
		res.echoes++
	}

	if soakMode {
		// Keep the bot connected and chatting periodically so the server sees
		// a steady load for the whole soak duration.
		t := time.NewTicker(*soakChatEvery)
		defer t.Stop()
		i := chatN
		for {
			select {
			case <-ctx.Done():
				goto disconnect
			case <-t.C:
				text := fmt.Sprintf("soak %s %d", name, i)
				i++
				if rtt, err := chatRoundTrip(c, text); err == nil {
					res.addRTT(rtt)
					rttHist.Add(rtt)
					meter.Mark(1)
					res.echoes++
				}
			}
		}
	}
disconnect:
	// Best-effort clean disconnect so server slots free up.
	_ = enqueue(c, []byte{raknet.IDDisconnectionNotification}, raknet.Reliable)
	_, _ = c.exchangeDeadline(50 * time.Millisecond)
	return res
}

// addRTT records a sample, capped at soakMaxRTTs to bound harness memory.
func (r *botResult) addRTT(v float64) {
	if len(r.rtts) >= *soakMaxRTTs {
		return
	}
	r.rtts = append(r.rtts, v)
}

func handshake(c *botClient, name string) error {
	// 1. Cookie handshake.
	var cookie uint16
	for attempt := 0; attempt < 3; attempt++ {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}
		if _, err := c.pc.WriteTo(raknet.EncryptDatagram(raknet.BuildOpenRequest(cookie), uint16(c.srv.Port)), c.srv); err != nil {
			return err
		}
		raw, err := c.recvTimeout(2 * time.Second)
		if err != nil {
			return fmt.Errorf("cookie reply: %w", err)
		}
		off := raknet.ParseOffline(raw)
		switch off.Kind {
		case raknet.OfflineCookieReply:
			cookie = off.Cookie
		case raknet.OfflineOpenReply:
			goto opened
		default:
			return fmt.Errorf("unexpected offline reply %x", raw)
		}
		cookie ^= raknet.SAMPPetarded
	}
	return fmt.Errorf("cookie handshake failed")
opened:

	// 2. Connection request + auth.
	if err := enqueue(c, append([]byte{raknet.IDConnectionRequest}, []byte{}...), raknet.Reliable); err != nil {
		return err
	}
	challenge, err := waitRaw(c, raknet.IDAuthKey, 5*time.Second)
	if err != nil {
		return err
	}
	chLen := int(challenge[1])
	entry, _, ok := protocol.FindAuthBySend(string(challenge[2 : 2+chLen-1]))
	if !ok {
		return fmt.Errorf("unknown auth challenge")
	}
	if err := enqueue(c, append([]byte{raknet.IDAuthKey, 40}, []byte(entry.Recv)...), raknet.Reliable); err != nil {
		return err
	}
	cra, err := waitRaw(c, raknet.IDConnectionRequestAccepted, 5*time.Second)
	if err != nil {
		return err
	}
	if len(cra) != 13 {
		return fmt.Errorf("short connection-accepted (%d bytes)", len(cra))
	}
	token := uint32(cra[9]) | uint32(cra[10])<<8 | uint32(cra[11])<<16 | uint32(cra[12])<<24

	// 3. New incoming connection.
	if err := enqueue(c, []byte{raknet.IDNewIncomingConnection, 127, 0, 0, 1, byte(c.port), byte(c.port >> 8)}, raknet.Reliable); err != nil {
		return err
	}
	if _, err := c.exchangeDeadline(2 * time.Millisecond); err != nil {
		return err
	}

	// 4. ClientJoin.
	join := buildJoin(name, token)
	if err := enqueue(c, protocol.BuildRPCFrame(protocol.RPCClientJoin, join), raknet.Reliable); err != nil {
		return err
	}
	if _, err := waitRPC(c, protocol.RPCInitGame, 5*time.Second); err != nil {
		return err
	}

	// 5. Class -> spawn.
	if err := enqueue(c, protocol.BuildRPCFrame(protocol.RPCRequestClass, []byte{0, 0, 0, 0}), raknet.Reliable); err != nil {
		return err
	}
	classResp, err := waitRPC(c, protocol.RPCRequestClass, 5*time.Second)
	if err != nil {
		return err
	}
	if _, err := protocol.ParseSpawnInfo(classResp[1:]); err != nil {
		return err
	}
	if err := enqueue(c, protocol.BuildRPCFrame(protocol.RPCRequestSpawn, nil), raknet.Reliable); err != nil {
		return err
	}
	allow, err := waitRPC(c, protocol.RPCRequestSpawn, 5*time.Second)
	if err != nil {
		return err
	}
	if len(allow) != 4 || allow[0] != 1 {
		return fmt.Errorf("spawn not allowed: %x", allow)
	}
	if err := enqueue(c, protocol.BuildRPCFrame(protocol.RPCSpawn, nil), raknet.Reliable); err != nil {
		return err
	}
	if _, err := c.exchangeDeadline(2 * time.Millisecond); err != nil {
		return err
	}
	return nil
}

// chatRoundTrip sends one chat line and waits for the server broadcast echo
// carrying the same text. It returns the RTT in milliseconds.
func chatRoundTrip(c *botClient, text string) (float64, error) {
	payload := append([]byte{byte(len(text))}, []byte(text)...)
	if err := enqueue(c, protocol.BuildRPCFrame(protocol.RPCChat, payload), raknet.Reliable); err != nil {
		return 0, err
	}
	t0 := time.Now()
	matched := false
	_, err := c.exchangeUntil(func(all [][]byte) bool {
		for _, p := range all {
			id, body, err := protocol.ParseRPCFrame(p)
			if err != nil || id != protocol.RPCChat {
				continue
			}
			if len(body) >= 3 && string(body[3:]) == text {
				matched = true
				return true
			}
		}
		return false
	}, t0.Add(5*time.Second))
	if err != nil {
		return 0, err
	}
	if !matched {
		return 0, fmt.Errorf("timeout waiting for chat echo %q", text)
	}
	return float64(time.Since(t0).Microseconds()) / 1000.0, nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gosamp-bench:", err)
		os.Exit(1)
	}
}

func run() error {
	// Ramp mode overrides a single -clients value.
	stageCounts := []int{}
	if *ramp != "" {
		for _, s := range strings.Split(*ramp, ",") {
			n, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || n <= 0 {
				return fmt.Errorf("ramp: bad stage %q", s)
			}
			stageCounts = append(stageCounts, n)
		}
	} else if *clients > 0 {
		stageCounts = []int{*clients}
	} else {
		return fmt.Errorf("clients must be > 0 (or provide -ramp)")
	}
	if *chatPer < 0 {
		return fmt.Errorf("chat-per-client must be >= 0")
	}
	srv, err := net.ResolveUDPAddr("udp4", *serverFlag)
	if err != nil {
		return fmt.Errorf("resolve server: %w", err)
	}

	// Network profile -> deterministic netsim.Sim per bot.
	prof, err := profileFromNet(*netProfile)
	if err != nil {
		return err
	}

	// Soak mode: keep a fixed load connected and log RSS/goroutines every 30s.
	if *soak > 0 {
		return runSoak(srv, prof, stageCounts[len(stageCounts)-1])
	}

	// Ramp mode: run each stage, escalating until failure or memguard trips.
	var lastBreak string
	for _, stage := range stageCounts {
		if *memGuard && systemCriticallyLoaded() {
			lastBreak = fmt.Sprintf("system critically loaded before stage %d (memguard)", stage)
			break
		}
		ctx, cancel := ctxOrTimeout()
		ok, failed, spawnHist, rttHist, rate, err := runStage(ctx, srv, stage, prof)
		cancel()
		if err != nil {
			return err
		}
		fmt.Printf("stage %d bots: %d ok, %d failed (of %d)\n", stage, ok, failed, stage)
		printHist("  time-to-spawn ms", spawnHist)
		printHist("  chat RTT ms", rttHist)
		fmt.Printf("  echo throughput: %.1f msg/s\n", rate)
		if failed > 0 || ok == 0 {
			lastBreak = fmt.Sprintf("stage %d failed: %d ok, %d failed", stage, ok, failed)
			break
		}
		if *memGuard && systemCriticallyLoaded() {
			lastBreak = fmt.Sprintf("system critically loaded after stage %d (memguard)", stage)
			break
		}
	}
	if lastBreak != "" {
		fmt.Printf("RAMP BREAKPOINT: %s\n", lastBreak)
	} else {
		fmt.Println("RAMP COMPLETE: all stages passed without breaking")
	}
	return nil
}

// runStage runs one batch of `n` bots concurrently and returns metrics.
func runStage(ctx context.Context, srv *net.UDPAddr, n int, prof *netsim.Profile) (ok, failed int, spawnHist, rttHist *bench.Histogram, rate float64, err error) {
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	results := make([]botResult, n)
	var meter bench.Meter
	sh, rh := &bench.Histogram{}, &bench.Histogram{}
	wallStart := time.Now()
	meter.Start(wallStart)
	for i := 0; i < n; i++ {
		if *stagger > 0 && i > 0 {
			select {
			case <-ctx.Done():
			case <-time.After(*stagger):
			}
		}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			res := runBot(ctx, srv, idx, *chatPer, newSimFor(prof, idx), sh, rh, &meter, false, nil)
			results[idx] = res
			if res.err != nil {
				errCh <- fmt.Errorf("bot %s: %w", res.name, res.err)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	wallEnd := time.Now()
	rate = meter.Rate(wallEnd)
	for e := range errCh {
		fmt.Fprintln(os.Stderr, "gosamp-bench:", e)
	}
	for _, r := range results {
		if r.err == nil && r.spawnMs > 0 {
			ok++
		} else {
			failed++
		}
	}
	if ok == 0 {
		return ok, failed, sh, rh, rate, fmt.Errorf("no bots spawned in stage")
	}
	return ok, failed, sh, rh, rate, nil
}

// soakSample is one formatted sampler tick.
type soakSample struct {
	elapsed    time.Duration
	rssBytes   uint64
	heapAlloc  uint64 // bytes
	heapInuse  uint64 // bytes
	heapObjs   uint64
	numGC      uint32
	goroutines int64
	dHeap      int64 // bytes since previous sample
	botsAlive  int64
	totalBots  int64
}

// formatSoakLine renders one sample with fixed field order and KB precision
// (no rounding to MB). Example:
// [soak 01h05m30s] rss=15.2MB heapAlloc=2143KB heapInuse=3078KB objs=24112 gc=1834 gor=6 dHeap30s=+0KB bots=50/50
func (s soakSample) line() string {
	sign := "+"
	d := s.dHeap
	if d < 0 {
		sign = "-"
		d = -d
	}
	return fmt.Sprintf("[soak %s] rss=%.1fMB heapAlloc=%dKB heapInuse=%dKB objs=%d gc=%d gor=%d dHeap30s=%s%dKB bots=%d/%d",
		formatElapsed(s.elapsed),
		float64(s.rssBytes)/1048576.0,
		s.heapAlloc/1024,
		s.heapInuse/1024,
		s.heapObjs,
		s.numGC,
		s.goroutines,
		sign, d/1024,
		s.botsAlive, s.totalBots)
}

// csvHeader is the fixed column order for soak-samples.csv.
const csvHeader = "elapsed_seconds,rss_mb,heap_alloc_kb,heap_inuse_kb,heap_objects,num_gc,goroutines,dheap30s_kb,bots_alive,bots_total"

func (s soakSample) csvRow() []string {
	return []string{
		strconv.FormatInt(int64(s.elapsed.Seconds()), 10),
		fmt.Sprintf("%.1f", float64(s.rssBytes)/1048576.0),
		strconv.FormatUint(s.heapAlloc/1024, 10),
		strconv.FormatUint(s.heapInuse/1024, 10),
		strconv.FormatUint(s.heapObjs, 10),
		strconv.FormatUint(uint64(s.numGC), 10),
		strconv.FormatInt(s.goroutines, 10),
		strconv.FormatInt(s.dHeap/1024, 10),
		strconv.FormatInt(s.botsAlive, 10),
		strconv.FormatInt(s.totalBots, 10),
	}
}

// formatElapsed renders a duration as HHhMMmSSs (hours may exceed 2 digits).
func formatElapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d / time.Hour)
	m := int((d % time.Hour) / time.Minute)
	sec := int((d % time.Minute) / time.Second)
	return fmt.Sprintf("%02dh%02dm%02ds", h, m, sec)
}

// hourStats accumulates min and sum for heapAlloc/heapObjects within one hour.
type hourStats struct {
	n        int
	allocMin uint64
	allocSum uint64
	objsMin  uint64
	objsSum  uint64
}

func (h *hourStats) add(s soakSample) {
	if h.n == 0 {
		h.allocMin, h.objsMin = s.heapAlloc, s.heapObjs
	}
	if s.heapAlloc < h.allocMin {
		h.allocMin = s.heapAlloc
	}
	if s.heapObjs < h.objsMin {
		h.objsMin = s.heapObjs
	}
	h.allocSum += s.heapAlloc
	h.objsSum += s.heapObjs
	h.n++
}

func (h *hourStats) allocAvg() uint64 {
	if h.n == 0 {
		return 0
	}
	return h.allocSum / uint64(h.n)
}

func (h *hourStats) objsAvg() uint64 {
	if h.n == 0 {
		return 0
	}
	return h.objsSum / uint64(h.n)
}

// hourlyLine renders the [hourly HH] summary. delta is vs the previous hour's
// minimum (signed); the *shift of the minimum* across hours is the leak signal.
func hourlyLine(hour int, cur, prev hourStats, havePrev bool) string {
	dAllocMin, dObjsMin := int64(0), int64(0)
	if havePrev {
		dAllocMin = int64(cur.allocMin) - int64(prev.allocMin)
		dObjsMin = int64(cur.objsMin) - int64(prev.objsMin)
	}
	return fmt.Sprintf("[hourly %02d] allocMin=%dKB allocAvg=%dKB objsMin=%d objsAvg=%d dAllocMin=%+dKB dObjsMin=%+d objs",
		hour, cur.allocMin/1024, cur.allocAvg()/1024, cur.objsMin, cur.objsAvg(), dAllocMin/1024, dObjsMin)
}

// runSoak keeps `n` bots connected for *soak duration, periodically sampling
// the SERVER process and logging detailed KB metrics, a per-hour min/avg
// summary, and a CSV of every sample.
func runSoak(srv *net.UDPAddr, prof *netsim.Profile, n int) error {
	ctx, cancel := context.WithTimeout(context.Background(), *soak)
	defer cancel()

	results := make([]botResult, n)
	var alive int64
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		if *stagger > 0 && i > 0 {
			select {
			case <-ctx.Done():
			case <-time.After(*stagger):
			}
		}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			res := runBot(ctx, srv, idx, *chatPer, newSimFor(prof, idx), &bench.Histogram{}, &bench.Histogram{}, &bench.Meter{}, true, &alive)
			results[idx] = res
		}(i)
	}

	// CSV sink for every sample.
	var csvw *csv.Writer
	var csvf *os.File
	if *csvPath != "" {
		f, err := os.Create(*csvPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "gosamp-bench: soak csv:", err)
		} else {
			csvf = f
			csvw = csv.NewWriter(f)
			_ = csvw.Write(strings.Split(csvHeader, ","))
			csvw.Flush()
		}
	}
	if csvf != nil {
		defer csvf.Close()
	}

	// Header line describes the columns.
	fmt.Println("# soak sampler: columns elapsed | rss(MB,1dp) | heapAlloc/heapInuse(KB) | objs | gc | gor | dHeap30s(KB,+/-) | bots(alive/total)")
	fmt.Printf("# csv: %s (empty if -csv unset)\n", *csvPath)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	start := time.Now()
	var prevHeap uint64
	peakRSS := uint64(0)
	if *metricsURL == "" {
		fmt.Fprintln(os.Stderr, "gosamp-bench: soak warning: -server-metrics unset; sampling disabled")
	}

	var cur hourStats
	var prev hourStats
	hour := 0
	havePrev := false
loop:
	for {
		select {
		case <-ticker.C:
			sm, ok := sampleServer(*metricsURL)
			if !ok {
				fmt.Printf("[soak %s] server metrics unavailable\n", formatElapsed(time.Since(start)))
				continue
			}
			if sm.rss > peakRSS {
				peakRSS = sm.rss
			}
			var growth int64
			if prevHeap > 0 {
				growth = int64(sm.heapAlloc) - int64(prevHeap)
			}
			prevHeap = sm.heapAlloc
			sample := soakSample{
				elapsed:    time.Since(start),
				rssBytes:   sm.rss,
				heapAlloc:  sm.heapAlloc,
				heapInuse:  sm.heapInuse,
				heapObjs:   sm.heapObjects,
				numGC:      sm.numGC,
				goroutines: sm.goroutines,
				dHeap:      growth,
				botsAlive:  atomic.LoadInt64(&alive),
				totalBots:  int64(n),
			}
			fmt.Println(sample.line())
			if csvw != nil {
				_ = csvw.Write(sample.csvRow())
				csvw.Flush()
			}
			cur.add(sample)
			// Emit an hourly summary when the wall clock crosses each hour.
			if elapsedHours := int(sample.elapsed / time.Hour); elapsedHours > hour {
				hour = elapsedHours
				fmt.Println(hourlyLine(hour, cur, prev, havePrev))
				prev = cur
				havePrev = true
				cur = hourStats{}
			}
		case <-ctx.Done():
			break loop
		}
	}
	if cur.n > 0 {
		fmt.Println(hourlyLine(hour+1, cur, prev, havePrev))
	}
	wg.Wait()
	var failed int
	for _, r := range results {
		if r.err != nil {
			failed++
			fmt.Fprintln(os.Stderr, "gosamp-bench:", r.err)
		}
	}
	fmt.Printf("soak complete: %d/%d bots survived, server peak RSS=%.1fMB\n",
		n-failed, n, float64(peakRSS)/1048576.0)
	if failed > 0 {
		return fmt.Errorf("%d bots failed during soak", failed)
	}
	return nil
}

// serverSample is the subset of the server's runtime.MemStats we track.
type serverSample struct {
	rss         uint64
	heapAlloc   uint64
	heapInuse   uint64
	heapObjects uint64
	numGC       uint32
	goroutines  int64
}

// sampleServer reads the server's expvar endpoint and extracts memstats.
func sampleServer(url string) (serverSample, bool) {
	var s serverSample
	if url == "" {
		return s, false
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return s, false
	}
	defer resp.Body.Close()
	var vars map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&vars); err != nil {
		return s, false
	}
	ms, ok := vars["memstats"].(map[string]any)
	if !ok {
		return s, false
	}
	get := func(k string) uint64 {
		if v, ok := ms[k].(float64); ok {
			return uint64(v)
		}
		return 0
	}
	s.rss = get("Sys")
	s.heapAlloc = get("HeapAlloc")
	s.heapInuse = get("HeapInuse")
	s.heapObjects = get("HeapObjects")
	s.numGC = uint32(get("NumGC"))
	if v, ok := vars["rss"].(float64); ok && uint64(v) > 0 {
		s.rss = uint64(v)
	}
	if v, ok := vars["goroutines"].(float64); ok {
		s.goroutines = int64(v)
	}
	return s, true
}

// ctxOrTimeout returns a context bounded by -timeout. The cancel is deferred
// by the caller scope naturally; for a CLI this is acceptable since each stage
// creates a fresh context.
func ctxOrTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), *timeout)
}

// profileFromNet maps the -net flag to a netsim.Profile. Each bot gets its own
// Sim (with a per-bot seed) because netsim.Sim's RNG is not goroutine-safe and
// bots run concurrently.
func profileFromNet(name string) (*netsim.Profile, error) {
	switch name {
	case "clean":
		p := netsim.ProfileClean()
		return &p, nil
	case "mobile3g":
		p := netsim.ProfileMobile3G()
		return &p, nil
	case "badmobile":
		p := netsim.ProfileBadMobile()
		return &p, nil
	case "flaky":
		p := netsim.ProfileFlakyWifi()
		return &p, nil
	default:
		return nil, fmt.Errorf("unknown -net profile %q (clean|mobile3g|badmobile|flaky)", name)
	}
}

// newSimFor returns a netsim.Sim for one bot (nil for clean network).
func newSimFor(prof *netsim.Profile, idx int) *netsim.Sim {
	if prof == nil {
		return nil
	}
	return netsim.New(*prof, int64(1700000000)+int64(idx))
}

// systemCriticallyLoaded reports whether the host is near OOM/CPU saturation,
// used by the ramp memory guard to stop escalation before the machine becomes
// unresponsive. It reads /proc/meminfo and /proc/loadavg.
func systemCriticallyLoaded() bool {
	mem := readMeminfo()
	load := readLoadavg()
	// Stop if free memory is dangerously low (<5%) or 1-min loadavg > 8x cores.
	if mem > 0 && mem < 5 {
		fmt.Fprintln(os.Stderr, "gosamp-bench: memguard: free memory below 5%")
		return true
	}
	if load > 8*float64(runtime.NumCPU()) {
		fmt.Fprintf(os.Stderr, "gosamp-bench: memguard: loadavg %.1f exceeds %d x cores\n", load, runtime.NumCPU())
		return true
	}
	return false
}

func readMeminfo() float64 {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	var total, available uint64
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(line, "MemTotal: %d", &total)
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			fmt.Sscanf(line, "MemAvailable: %d", &available)
		}
	}
	if total == 0 {
		return 0
	}
	return float64(available) / float64(total) * 100.0
}

func readLoadavg() float64 {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0
	}
	var one float64
	fmt.Sscanf(string(b), "%f", &one)
	return one
}

func printHist(label string, h *bench.Histogram) {
	if h.Count() == 0 {
		fmt.Printf("%s: n=0 (no samples)\n", label)
		return
	}
	fmt.Printf("%s: n=%d min=%.2f p50=%.2f p95=%.2f p99=%.2f max=%.2f mean=%.2f\n",
		label, h.Count(), h.Min(), h.Quantile(0.5), h.Quantile(0.95), h.Quantile(0.99), h.Max(), h.Mean())
}

func writeCSV(path string, results []botResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{"bot", "time_to_spawn_ms", "rtt_p50_ms", "rtt_p95_ms", "echoes"}); err != nil {
		return err
	}
	for _, r := range results {
		row := []string{
			r.name,
			fmt.Sprintf("%.3f", r.spawnMs),
			fmt.Sprintf("%.3f", r.rttP50()),
			fmt.Sprintf("%.3f", r.rttP95()),
			fmt.Sprintf("%d", r.echoes),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return w.Error()
}
