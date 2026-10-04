#!/usr/bin/env bash
# Independent deep-monitor benchmark: drives N bots against a running sa-go
# server while sampling CPU, RAM, RSS/heap, goroutines, disk I/O, context
# switches and socket counts in parallel.
#
# Usage: ./scripts/deep_bench.sh <clients> <chat_per_client> [net_profile]
set -uo pipefail
cd "$(dirname "$0")/.."

CLIENTS="${1:-1000}"
CHAT="${2:-2}"
NET="${3:-clean}"
PORT=7789
HTTP=127.0.0.1:6069
OUT=/tmp/opencode/deepbench
BIN=/root/gosamp/bin/gosamp-server
BENCH=/root/gosamp/bin/gosamp-bench

rm -rf "$OUT"; mkdir -p "$OUT"

cat > "$OUT/bench.cfg" <<EOF
port $PORT
hostname "SA:GO DeepBench $CLIENTS"
maxplayers 1000
serialcheck 0
ratelimit_burst 100000
ratelimit_persec 100000
ipratelimit_burst 100000
ipratelimit_persec 100000
addclass 0 1958.3783 1343.1572 15.3746 270.1425 24 100 25 50 0 0
EOF

echo "[deepbench] starting server (port $PORT, $CLIENTS bots, net=$NET)"
nohup "$BIN" -cfg "$OUT/bench.cfg" -loglevel warn -http "$HTTP" > "$OUT/server.log" 2>&1 &
SRV_PID=$!
echo "$SRV_PID" > "$OUT/srv.pid"
sleep 2

# --- Parallel monitor loops -------------------------------------------------
# 1. Per-process CPU/RAM/threads/context-switches via pidstat (1s granularity).
( while kill -0 "$SRV_PID" 2>/dev/null; do
    pidstat -p "$SRV_PID" -u -r -w -t 1 1 >> "$OUT/pidstat.log" 2>/dev/null
    echo "---" >> "$OUT/pidstat.log"
  done ) &
MON1=$!

# 2. Global CPU/mem/io/context-switch snapshot via vmstat.
( while kill -0 "$SRV_PID" 2>/dev/null; do
    vmstat -s -t >> "$OUT/vmstat-s.log" 2>/dev/null
    echo "---" >> "$OUT/vmstat-s.log"
    vmstat -w -t 1 1 >> "$OUT/vmstat.log" 2>/dev/null
    echo "---" >> "$OUT/vmstat.log"
  done ) &
MON2=$!

# 3. Disk I/O per-device.
( while kill -0 "$SRV_PID" 2>/dev/null; do
    iostat -dxk 1 1 >> "$OUT/iostat.log" 2>/dev/null
    echo "---" >> "$OUT/iostat.log"
  done ) &
MON3=$!

# 4. Server expvar (heap objects, goroutines, RSS) via curl.
( while kill -0 "$SRV_PID" 2>/dev/null; do
    curl -s "$HTTP/debug/vars" >> "$OUT/expvar.json" 2>/dev/null
    echo "" >> "$OUT/expvar.json"
    sleep 1
  done ) &
MON4=$!

# 5. Socket/connection counters.
( while kill -0 "$SRV_PID" 2>/dev/null; do
    date +%s >> "$OUT/sockets.log"
    ss -u -a -n | wc -l >> "$OUT/sockets.log"
    sleep 1
  done ) &
MON5=$!

sleep 1
echo "[deepbench] launching bench"
"$BENCH" -server "127.0.0.1:$PORT" -clients "$CLIENTS" \
    -chat-per-client "$CHAT" -net "$NET" -timeout 600s 2>&1 | tee "$OUT/bench.log"

echo "[deepbench] stopping server"
kill "$SRV_PID" 2>/dev/null
wait "$SRV_PID" 2>/dev/null
for m in $MON1 $MON2 $MON3 $MON4 $MON5; do kill "$m" 2>/dev/null; done
wait 2>/dev/null

echo "[deepbench] done. Artifacts in $OUT"
ls -la "$OUT"