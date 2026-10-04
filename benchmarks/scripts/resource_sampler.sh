#!/usr/bin/env bash
# Steady-state resource sampler: holds N bots connected for D seconds while
# sampling the server's VmRSS/VmThreads from /proc and system counters.
# Usage: resource_sampler.sh <bin> <cfg> <port> <clients> <seconds> <outfile>
set -uo pipefail
BIN="$1"; CFG="$2"; PORT="$3"; CLIENTS="$4"; SECS="$5"; OUT="$6"
BENCH=/root/gosamp/bin/gosamp-bench

"$BIN" -cfg "$CFG" > "$OUT.server.log" 2>&1 &
SRV=$!
sleep 2
: > "$OUT.samples"
( end=$((SECONDS+SECS))
  while [ $SECONDS -lt $end ] && kill -0 "$SRV" 2>/dev/null; do
    rss=$(awk '/VmRSS/{print $2}' /proc/$SRV/status 2>/dev/null)
    thr=$(awk '/Threads/{print $2}' /proc/$SRV/status 2>/dev/null)
    fd=$(ls /proc/$SRV/fd 2>/dev/null | wc -l)
    cpu=$(awk '{print $14+$15}' /proc/$SRV/stat 2>/dev/null)
    echo "$(date +%s) rss_kb=$rss threads=$thr fds=$fd cpu_ticks=$cpu" >> "$OUT.samples"
    sleep 1
  done ) & SAMP=$!

# Hold bots connected in soak mode.
"$BENCH" -server "127.0.0.1:$PORT" -clients "$CLIENTS" -soak "${SECS}s" \
    -chat-per-client 1 -net clean -stagger 8ms -timeout 0 >> "$OUT.bench.log" 2>&1 &
BP=$!
wait $BP 2>/dev/null
kill $SRV $SAMP 2>/dev/null
wait 2>/dev/null
echo "samples: $OUT.samples"