![SA:GO](sa_go.png)

# SA:GO — The Modern, Lightweight, and Rock-Solid SA-MP 0.3.7 Server

Welcome to a new era of San Andreas Multiplayer server hosting. **SA:GO** is a clean, modern SA-MP 0.3.7 server engineered from the ground up for server owners and community operators who demand stellar performance, uncompromising stability, and minimal hardware footprint. Whether you are running a tight-knit roleplay community or scaling a bustling multi-gamemode hub, SA:GO empowers you to deliver an unforgettable player experience without the overhead of massive system resources.

For server operators, the biggest headaches are often runaway hosting bills, frustrating player lag, or servers that mysteriously degrade in performance after hours of continuous runtime. SA:GO eliminates these concerns completely. Your server runs smoothly on the most modest VPS instances, delivering instantaneous, low-latency, and crash-resilient gameplay around the clock.

---

## Proven Reliability & Benchmark Comparison

SA:GO's performance advantages are backed by rigorous, real-world head-to-head stress testing against **open.mp (v1.5.8.3079)** using the identical multi-bot benchmarking harness on the same host environment. Every figure below comes from a reproducible run.

### Head-to-Head Performance: SA:GO vs open.mp

| Benchmark Scenario | Server | Connected Bots | Spawn Time (p50) | Chat Latency (p50) | Result |
|---|---|---|---|---|---|
| **20 Bots**<br>*(Baseline Load)* | **SA:GO** | **20 / 20 (100%)** | **5.3 ms** | **0.07 ms** | **Stable** |
| | open.mp | 20 / 20 (100%) | 144.7 ms | 10.6 ms | ~27× slower spawn |
| **50 Bots**<br>*(Heavy Concurrency)* | **SA:GO** | **50 / 50 (100%)** | **5.1 ms** | **0.06 ms** | **Stable** |
| | open.mp | 50 / 50 (100%) | 195.5 ms | 16.5 ms | ~38× slower spawn |
| **100 Bots**<br>*(Extreme Stress)* | **SA:GO** | **100 / 100 (100%)** | **5.2 ms** | **0.07 ms** | **Stable** |
| | open.mp | 100 / 100 (100%) | 236.8 ms | 21.9 ms | ~46× slower spawn |
| **50 Bots**<br>*(Flaky WAN: 5% Loss + Jitter)* | **SA:GO** | **50 / 50 (100%)** | **269.0 ms** | **63.2 ms** | **Stable** |
| | open.mp | 41 / 50 (82%) | 320.8 ms | 74.1 ms | 9 bots dropped |

### The Thousand-Player Showcase

The headline event: **1000 concurrent players** joined a single SA:GO server, exchanged chat, and stayed connected — with a **100% success rate** and not a single dropped player. Both servers were driven by the same harness on the same host, with identical staggering.

Both servers ran a gamemode (SA:GO: `benchmarks/bench.lua`; open.mp: its stock `gungame.amx`) and both were staggered identically.

| Metric | SA:GO | open.mp 1.5.8.3079 | SA:GO Advantage |
|---|---|---|---|
| **Players connected** | **1000 / 1000 (100%)** | 1000 / 1000 (100%) | Tied at full capacity |
| **Spawn p50** | **5.0 ms** | 136–602 ms\* | **~27–120× faster** |
| **Spawn p95** | **5.7 ms** | 139–804 ms | — |
| **Spawn p99** | **6.2 ms** | 140–844 ms | — |
| **Spawn max** | **6.8 ms** | 141–876 ms | — |
| **Chat RTT p50** | **0.06 ms** | 8.3–140 ms\* | — |
| **Chat RTT p95** | **0.17 ms** | 13–210 ms | — |
| **Chat RTT p99** | **0.27 ms** | 14–232 ms | — |
| **Chat RTT max** | **1.04 ms** | 19–240 ms | — |
| **Server CPU** | **~11%** (max 13%) | ~31% (max 38%) | **~3× lighter** |
| **Resident memory (RSS)** | **~17 MB** (idle ~11 MB) | ~104 MB (max 113 MB) | **~6× less RAM** |
| **Threads** | **10** | 2 (plus internal pools) | — |
| **File descriptors** | **6–7 (one UDP socket)** | 5 | Single-socket design |
| **Major page faults** | **0** | 0 | No disk paging |
| **Involuntary context switches** | **0** | — | No scheduler contention |
| **Disk utilization** | **0.03%** | — | Effectively no disk I/O |
| **Goroutines (steady state)** | **5** | — | Tiny runtime footprint |

\* open.mp's numbers varied markedly **between runs** (spawn p50 anywhere from
136 ms to 602 ms at 1000 bots, depending on its internal tick/GC state). We show
the observed range rather than cherry-pick the best run. SA:GO stayed at ~5 ms
across every run.

**Five-minute soak at 1000 players:** resident memory stayed flat at **~17–21 MB**, goroutines held at **5**, and the server logged **zero timeouts, zero rate-limit drops, and zero panics** from the first minute to the last. A thousand players cost SA:GO about a tenth of one CPU core and under 20 MB of RAM — a load any entry-level VPS can carry.

Resource figures were captured with per-second process sampling (`pidstat`, `/proc`, `vmstat`, `iostat`).

#### Don't take our word for it — reproduce it

Every number above is backed by a complete, self-contained benchmark package in
[`benchmarks/`](benchmarks/). It contains the **full source of the measurement
harness** (a standalone Go program that speaks the real RakNet/SA-MP 0.3.7 wire
protocol over UDP), the runner scripts, the exact server configs, and the **raw
logs** from the canonical runs. Build it, point it at any SA-MP 0.3.7 server, and
verify or falsify the results yourself.

```bash
cd benchmarks/harness && go build -o gosamp-bench ./cmd/gosamp-bench
./gosamp-bench -server 127.0.0.1:7777 -clients 1000 -chat-per-client 5 -stagger 8ms
```

See [`benchmarks/README.md`](benchmarks/README.md) for the methodology, fairness
notes, and raw evidence.

#### Key Performance Takeaways
- **~27–46× Faster World Entry (clean network):** SA:GO places players in the game world in a consistent ~5 ms from 20 up to 1000 players, versus ~145–600 ms on open.mp.
- **Sub-millisecond Internal Latency:** Median chat round-trip of ~0.06–0.10 ms versus ~10–140 ms on open.mp.
- **~6× Lower Memory Footprint:** A full 1000-player server holds steady at roughly 17 MB of RAM versus ~104 MB on open.mp.
- **Packet Loss Tolerance (honest caveat):** Under simulated 5% packet loss + reordering + 5–60 ms jitter, SA:GO retained 100% of connections while open.mp dropped some bots — but the latency *ratio* shrinks to about 1.2× because at that point the impairment, not the engine, dominates the numbers.
- **Zero-Leak Endurance:** Continuous soak testing shows a flat memory footprint with no leakage or throughput degradation.

#### What we do NOT claim
- We do **not** claim a "116×" or "255×" universal speedup. Those figures come from dividing very small chat numbers and are not a fair overall summary. The honest headline is the spawn ratio (~27–46× on a clean network), which is stable across bot counts.
- We do **not** claim SA:GO is "faster than open.mp at everything." Under heavy network impairment the advantage collapses to near parity.
- We do **not** claim dominance on all hardware. Results depend on CPU, kernel and NIC — reproduce on yours with the bundled harness.
- We have **not** benchmarked against the original `samp03svr`; only SA:GO vs open.mp v1.5.8.3079 is measured here.
- open.mp's own numbers vary substantially between runs (observed spawn p50 136–602 ms at 1000 bots) depending on its internal state; a single run is not definitive for either server.

#### Why is SA:GO faster, and where is it not?

The gap comes from **engine architecture, not gamemode tricks**: SA:GO's player
join path and chat relay are native compiled code driven directly by arriving
packets, whereas open.mp routes through its scripting scheduler and server tick,
which adds a scheduling floor (visible as the ~90 ms minimum spawn). SA:GO is
also a native 64-bit build rather than a 32-bit binary with a compatibility
layer. In the SA:GO technical docs we explain this openly, including the cases
where the advantage disappears (heavy packet loss, real WAN latency). The
benchmark package's own README covers this in detail.

---

## Feature-Rich World Building

SA:GO provides a comprehensive suite of built-in features, enabling developers to build any gamemode imaginable—from hardcore roleplay and cops-and-robbers to intense gang wars and street racing:

- **Secure Authentication & Cinematic Character Selection:** Protect player accounts with cryptographic bcrypt password hashing out of the box. Welcome players with cinematic camera angles that smoothly frame character previews as players cycle through skins with PREV and NEXT buttons.
- **Versatile Dialog System:** Communicate interactively using full dialog support, including informational message boxes, single-line text inputs, item selection lists, multi-column tablists, and masked password prompts.
- **Custom TextDraws & HUD Elements:** Design bespoke visual interfaces on the classic 640x480 virtual canvas. Create speedometers, radar overlays, hunger bars, notification badges, and interactive click-to-select menus using both global and per-player TextDraws.
- **Flexible Team Framework:** Establish police departments, emergency medical services, rival street gangs, or tactical squads with customizable nametag colors, friendly fire rules, and team mechanics.
- **Crisp Combat & Precision Bullet Sync:** Firefights feel tight and satisfying thanks to high-fidelity weapon shot tracking, hit position reporting, and synchronized aiming vectors with accurate hit registration.
- **Complete SA-MP Weapon Arsenal:** Fully supports every weapon category in GTA: San Andreas—melee weapons, handguns, shotguns, submachine guns, assault rifles, heavy ordnance, thrown explosives, and utility gear like parachutes and fire extinguishers.
- **Fluid Vehicle & Trailer Synchronization:** High-speed pursuits and convoys run smoothly without jitter, warping, or rubberbanding. Synchronization accommodates drivers, passengers in every seat, and hitch-connected trailers.
- **Realistic Vehicle Damage & Destruction:** Features granular damage tracking covering individual door panels, engine hoods, bumpers, headlights, and punctured tires, culminating in dramatic vehicle explosions when health reaches zero.
- **Standard & Race Checkpoints:** Direct player journeys, delivery routes, and competitive tournaments with traditional ground cylinders or arrow-guided race checkpoints.
- **Living World Actors (NPCs):** Populate streets, shopfronts, and checkpoints with ambient non-player character models that play looping animations, face custom angles, and take damage without consuming player slots.
- **Dynamic Gang Zones:** Mark neighborhood territories, hazard perimeters, and safe zones on the radar map with custom RGBA colors and attention-grabbing flashing animations.
- **3D Text World Labels:** Attach floating informative labels to world coordinates, player heads, or vehicle roofs with customizable view distances and line-of-sight tests.
- **Cinematic Interpolated Cameras:** Direct breathtaking flybys, intro cutscenes, and dramatic transitions using smooth point-to-point camera position and look-at interpolation.
- **Built-in Database Storage:** Persist player profiles, vehicle ownership, inventory, and economy data reliably with out-of-the-box support for embedded SQLite as well as MySQL drivers.

---

## Out-of-the-Box Commands & Lua Scripting

SA:GO comes ready to run with an official starter gamemode (`Gamemodes/bare.lua`) featuring essential gameplay and moderation commands:

- `/veh <model> <color1> <color2>` — Spawns any vehicle model directly at the player's position.
- `/giveweapon <id> <ammo>` — Equips a player with any valid weapon ID and ammunition count.
- `/heal` — Replenishes health and armour to 100%.
- `/setadmin` — Enables administrative privileges for testing.
- `/kick <id> [reason]` — Disconnects rule-breaking players cleanly.
- `/ban <id> [reason]` — Blocks persistent offenders from reconnecting.
- `/say <message>` — Broadcasts administrative server announcements to all players.
- `/help` — Displays available commands to connected players.

Every single behavior is customizable and extensible through a clean Lua scripting interface under the `sa.*` namespace. Developers have full programmatic access across eight core categories:
1. **Player Operations** (attributes, health, armour, skins, velocity, positioning, admin state)
2. **Vehicle Operations** (creation, health, visual damage status, passenger management)
3. **Weaponry & Combat** (inventory manipulation, ammo limits, weapon metadata)
4. **Interface Elements** (interactive dialogs, global & player TextDraws, click selection)
5. **World & Environment** (checkpoints, race checkpoints, actors, gang zones, 3D text labels)
6. **Cinematics** (static cameras, behind-player tracking, floating smooth interpolation)
7. **Database Storage** (synchronous and non-blocking asynchronous database queries)
8. **Security & Cryptography** (bcrypt password hashing, verification, secure token generation)

For the complete API reference with parameter types and code examples, see [SCRIPTING.md](SCRIPTING.md).

---

## Quick Start

Getting your SA:GO server running takes less than a minute on any modern platform:

### Linux (Ubuntu / Debian / CentOS / Arch)
1. Extract or clone the release package to your server directory.
2. Grant execution permissions to the server binary:
   ```bash
   chmod +x Bin/sa-go-server
   ```
3. Edit `server.cfg` to set your desired server name, player slots, and RCON password.
4. Start the server:
   ```bash
   ./Bin/sa-go-server -cfg server.cfg -gamemode Gamemodes/bare.lua
   ```

### Windows
1. Extract the release folder to your preferred directory.
2. Edit `server.cfg` using any text editor.
3. Launch the server from Command Prompt, PowerShell, or by double-clicking:
   ```cmd
   Bin\sa-go-server.exe -cfg server.cfg -gamemode Gamemodes/bare.lua
   ```

---

## License & Usage

SA:GO is proprietary, closed-source software provided free of charge as ready-to-run binaries. Server operators and community owners are granted full permission to deploy, run, and host SA:GO for their multiplayer communities, whether for hobbyist groups or large-scale public deployments.

Build your community on a faster, lighter, and more dependable foundation with SA:GO. Welcome to San Andreas!
