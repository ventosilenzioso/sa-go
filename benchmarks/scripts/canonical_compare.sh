#!/usr/bin/env bash
# Canonical reproducible comparison: SA:GO vs open.mp, identical harness.
# Writes raw logs to the given output dir.
# Usage: canonical_compare.sh <outdir> <clients> <chat> <stagger>
set -uo pipefail
OUT="${1:-/root/gosamp/benchmarks/raw}"
CLIENTS="${2:-1000}"
CHAT="${3:-5}"
STAGGER="${4:-8ms}"
BENCH=/root/gosamp/bin/gosamp-bench
SAGO=/root/gosamp/bin/gosamp-server
OMP_DIR=/root/openmp/Server
mkdir -p "$OUT"

echo "=== host ===" | tee "$OUT/host.txt"
{ echo "date: $(date -u)"; echo "cpu: $(grep -m1 'model name' /proc/cpuinfo)"; echo "cores: $(nproc)";
  echo "mem: $(free -m | awk '/Mem/{print $2" MB total, "$7" MB available"}')"; uname -a; } | tee -a "$OUT/host.txt"

echo "=== versions ===" | tee "$OUT/versions.txt"
/root/gosamp/bin/gosamp-server -h 2>&1 | head -1 >> "$OUT/versions.txt" || true
echo "SA:GO: $(cat /root/gosamp/VERSION 2>/dev/null || echo 'built from source')" | tee -a "$OUT/versions.txt"
"$OMP_DIR/omp-server" --version 2>/dev/null | head -1 >> "$OUT/versions.txt" || echo "open.mp: $(grep -m1 'Starting open.mp' "$OUT/../omp-version.txt" 2>/dev/null || echo 1.5.8.3079)" | tee -a "$OUT/versions.txt"

# ---- SA:GO ----
cat > "$OUT/sago.cfg" <<EOF
port 7797
hostname "SA:GO Canonical"
maxplayers 1000
serialcheck 0
ratelimit_burst 100000
ratelimit_persec 100000
ipratelimit_burst 100000
ipratelimit_persec 100000
addclass 0 1958.3783 1343.1572 15.3746 270.1425 24 100 25 50 0 0
EOF
echo "[canonical] SA:GO $CLIENTS bots"
"$SAGO" -cfg "$OUT/sago.cfg" -loglevel info -http 127.0.0.1:6081 > "$OUT/sago.server.log" 2>&1 &
S=$!; sleep 2
( while kill -0 $S 2>/dev/null; do pidstat -p $S -u -r 1 1 >> "$OUT/sago.pidstat" 2>/dev/null; echo --- >> "$OUT/sago.pidstat"; done ) & M=$!
( while kill -0 $S 2>/dev/null; do curl -s 127.0.0.1:6081/debug/vars > "$OUT/sago.expvar.json" 2>/dev/null; sleep 1; done ) & M2=$!
"$BENCH" -server 127.0.0.1:7797 -clients "$CLIENTS" -chat-per-client "$CHAT" -stagger "$STAGGER" -timeout 600s > "$OUT/sago.bench.log" 2>&1
kill $S $M $M2 2>/dev/null; wait 2>/dev/null

# ---- open.mp ----
cd "$OMP_DIR"
python3 - "$OMP_DIR/config.json" <<'PY'
import json,sys
p=sys.argv[1]; d=json.load(open(p))
d["max_players"]=1000
d["network"]["messages_limit"]=50000
d["network"]["acks_limit"]=50000
d["network"]["bind"]="127.0.0.1"
json.dump(d,open(p,"w"),indent=4)
PY
cp "$OMP_DIR/config.json" "$OUT/omp.config.json"
echo "[canonical] open.mp $CLIENTS bots"
"$OMP_DIR/omp-server" > "$OUT/omp.server.log" 2>&1 &
O=$!; sleep 3
( while kill -0 $O 2>/dev/null; do pidstat -p $O -u -r 1 1 >> "$OUT/omp.pidstat" 2>/dev/null; echo --- >> "$OUT/omp.pidstat"; done ) & MO=$!
"$BENCH" -server 127.0.0.1:7888 -clients "$CLIENTS" -chat-per-client "$CHAT" -stagger "$STAGGER" -timeout 600s > "$OUT/omp.bench.log" 2>&1
pkill -9 omp-server 2>/dev/null; kill $MO 2>/dev/null; wait 2>/dev/null

echo "[canonical] done -> $OUT"
echo "--- SA:GO ---"; grep -E "stage|spawn|chat|throughput" "$OUT/sago.bench.log" | tail -4
echo "--- open.mp ---"; grep -E "stage|spawn|chat|throughput" "$OUT/omp.bench.log" | tail -4
true