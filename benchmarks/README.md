# SA:GO Benchmark Package

This folder exists to make SA:GO's performance claims **verifiable by a third
party**. Everything needed to reproduce, inspect, or falsify the numbers in the
project README is here: the measurement harness source, the runner scripts, and
the raw logs from the canonical runs.

If you doubt a number, you do not have to trust us — build the harness, point it
at SA:GO and open.mp yourself, and compare.

---

## 1. What is measured

| Metric | Definition |
|---|---|
| **Time-to-spawn** | Wall-clock from bot start until it receives the server's `RequestClass` reply (cookie → auth → join → class). |
| **Chat RTT** | Round-trip time for one chat line: bot sends `RPC_Chat`, waits for the server's broadcast echo carrying the same text. |
| **Echo throughput** | Broadcast chat messages delivered per second during the run. |
| **CPU / RSS / threads / FDs** | Sampled per second from the OS (`pidstat`, `/proc`, `vmstat`, `iostat`). |

RTT is stopped **the instant the matching echo arrives** (not after a fixed
read-drain window), so it reflects real server latency rather than a polling
floor. See `harness/cmd/gosamp-bench/main.go` (`exchangeUntil`).

---

## 2. The harness

`harness/` is a standalone Go module (`module gosamp`). It speaks the **real
RakNet / SA-MP 0.3.7 wire protocol over UDP** — cookie handshake, auth
challenge, connection request, `ClientJoin`, class selection, spawn, chat. It
does not use any SA:GO-specific shortcut, so it drives SA:GO, open.mp, or the
original `samp03svr` identically.

Build:

```bash
cd harness
go build -o gosamp-bench ./cmd/gosamp-bench
```

Run 1000 bots (5 chat lines each, staggered launches):

```bash
./gosamp-bench -server 127.0.0.1:7777 -clients 1000 -chat-per-client 5 -stagger 8ms
```

Key flags:

| Flag | Meaning |
|---|---|
| `-clients N` | Concurrent bots |
| `-chat-per-client N` | Chat lines each bot sends |
| `-stagger D` | Delay between bot launches (avoids a synthetic thundering herd) |
| `-soak D` | Hold bots connected for duration D |
| `-soak-chat-every D` | Chat interval during soak (chat is O(N²)) |
| `-net clean\|flaky\|...` | Network impairment profile |

---

## 3. Environment (canonical run)

Taken from `raw/host.txt`:

- **CPU:** AMD EPYC 9575F, **4 cores** visible to the test host
- **RAM:** ~8 GB total (~4.5 GB available)
- **OS:** Ubuntu, Linux 7.0.0-34-generic, x86_64
- **Both servers:** loopback (127.0.0.1), run **alternately, never together**
- **open.mp:** v1.5.8.3079, config `raw/omp.config.json` (`max_players 1000`,
  `messages_limit 50000`, `acks_limit 50000`)

---

## 4. Fairness notes (read this before quoting a ratio)

1. **Both servers run a gamemode.** SA:GO runs `bench.lua` (this folder);
   open.mp runs its stock `gungame.amx`. Earlier ad-hoc runs compared *bare*
   SA:GO against gungame, which unfairly favoured SA:GO; the canonical script
   now loads a gamemode for both.
2. **Staggered launches.** Bots are launched 8 ms apart. Without staggering,
   N bots from one host create a thundering herd of cold handshakes that
   saturates the *harness host*, not the server — an artifact, not a real
   server limit. Real players join spread over time.
3. **Loopback, single host.** Client and server share one machine. This
   measures server processing latency, not real WAN latency.
4. **open.mp is stateful.** Its spawn/chat numbers vary between runs (observed
   p50 spawn 135–595 ms) depending on internal tick/GC state. SA:GO's numbers
   were stable (5 ms) across repeated runs. Do not read a single open.mp run as
   its definitive performance.
5. **One SA-MP. 0.3.7 protocol.** Chat is broadcast to all N bots, so aggregate
   chat load is O(N²); a per-bot chat interval that is comfortable at 50 bots
   will swamp a 1000-bot run. The `-soak-chat-every` flag exists for this.

---

## 5. Raw evidence

| File | Contents |
|---|---|
| `raw/host.txt` | Host hardware/OS at run time |
| `raw/sago-1000.bench.log` | SA:GO 1000-bot histogram output |
| `raw/omp-1000.bench.log` | open.mp 1000-bot histogram output |
| `raw/sago-1000.pidstat` | SA:GO per-second CPU/RSS |
| `raw/omp-1000.pidstat` | open.mp per-second CPU/RSS |
| `raw/sago-1000.expvar.json` | SA:GO Go runtime (goroutines, heap) |
| `raw/sago-1000.server.log` | SA:GO server log excerpt (join/class/spawn; shows gamemode loaded) |
| `raw/omp.config.json` | Exact open.mp config used |

This is a **representative** set, not every run. Note open.mp's 1000-bot result
here (p50 ~597 ms) is on the slow end of its observed 136–602 ms range; SA:GO's
p50 ~5.1 ms was stable across all runs.

---

## 6. Reproduce

```bash
# one-shot head-to-head, writes fresh logs
./scripts/canonical_compare.sh /tmp/bench-out 1000 5 8ms
```

Or manually, against a server you started yourself:

```bash
# terminal 1: server (SA:GO)
bin/gosamp-server -cfg server.cfg -gamemode benchmarks/bench.lua

# terminal 2: harness
benchmarks/harness/gosamp-bench -server 127.0.0.1:7777 -clients 1000 -chat-per-client 5 -stagger 8ms
```

---

## 7. Why the gap exists (and where it may not)

Fair question: open.mp is widely regarded as *more* optimized than the original
`samp03svr`. So why is SA:GO faster in this comparison? A few concrete,
checkable reasons — not secret sauce:

1. **Native vs interpreted hot path.** open.mp executes gamemode logic in an
   interpreted Pawn VM and routes player join/chat through its scripting
   scheduler. SA:GO's built-in chat relay and join path are native compiled
   code; the Lua gamemode is only invoked when it actually registers a handler.
   The measured `bench.lua`/`gungame` handlers do almost nothing, so this
   isolates engine overhead.
2. **No fixed tick gate on relay.** SA:GO flushes reliability/ACKs and relays
   packets as they are processed; open.mp's networking is driven by its server
   tick, which adds a scheduling floor (visible as the ~90 ms minimum spawn).
3. **Process model.** open.mp ships a 32-bit binary with a compatibility layer;
   SA:GO is a native 64-bit build.

**Where SA:GO is NOT 100× faster:** once the network itself is the bottleneck
(heavy packet loss, real WAN RTT, weak client devices), the engine's advantage
collapses toward parity — see the flaky-WAN row where the ratio is ~1.2×. The
honest, defensible headline is the **clean-network spawn ratio (~27–46×)**.

This comparison is **SA:GO vs open.mp only**. It says nothing about the original
`samp03svr`, which has not been measured here.

---

## 8. What this package does NOT claim

- It does **not** claim SA:GO is "116× faster" in every dimension. Chat RTT
  ratios (100×+) are large because they divide two small numbers; spawn ratios
  are the fair headline (~27–46× clean, ~1.2× under heavy simulated packet
  loss).
- It does **not** claim a universal result. Your CPU, kernel, NIC and network
  will differ. Reproduce it on your own hardware.
- It does **not** compare against the original `samp03svr` here. Only SA:GO vs
  open.mp has been measured.