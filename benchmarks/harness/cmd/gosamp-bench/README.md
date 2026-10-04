# gosamp-bench

Standalone SA-MP 0.3.7 protocol bot harness used for SA:GO benchmarking.
It speaks the real RakNet/SA-MP wire protocol over UDP, so it drives any
SA-MP 0.3.7-compatible server (SA:GO, open.mp, samp03svr).

Build:
    go build -o gosamp-bench ./cmd/gosamp-bench

Run (1000 bots, 5 chat lines each, staggered):
    ./gosamp-bench -server 127.0.0.1:7777 -clients 1000 -chat-per-client 5 -stagger 8ms
