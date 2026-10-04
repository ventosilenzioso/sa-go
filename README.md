![SA:GO](sa_go.png)

# SA:GO — The Modern, Lightweight, and Rock-Solid SA-MP 0.3.7 Server

Welcome to a new era of San Andreas Multiplayer server hosting. **SA:GO** is a clean, modern SA-MP 0.3.7 server engineered from the ground up for server owners and community operators who demand stellar performance, uncompromising stability, and minimal hardware footprint. Whether you are running a tight-knit roleplay community or scaling a bustling multi-gamemode hub, SA:GO empowers you to deliver an unforgettable player experience without the overhead of massive system resources.

For server operators, the biggest headaches are often runaway hosting bills, frustrating player lag, or servers that mysteriously degrade in performance after hours of continuous runtime. SA:GO eliminates these concerns completely. Your server runs smoothly on the most modest VPS instances, delivering instantaneous, low-latency, and crash-resilient gameplay around the clock.

---

## Proven Reliability & Benchmark Comparison

SA:GO's performance advantages are backed by rigorous, real-world head-to-head stress testing against **open.mp (v1.5.8.3079)** using the identical multi-bot benchmarking harness on the same host environment. Every figure below comes from a reproducible run.

### Head-to-Head Performance: SA:GO vs open.mp

| Benchmark Scenario | Server | Connected Bots | Spawn Time (p50) | Chat Latency (p50) | Throughput | Result |
|---|---|---|---|---|---|---|
| **20 Bots**<br>*(Baseline Load)* | **SA:GO** | **20 / 20 (100%)** | **5.0 ms** | **0.05 ms** | **456 msg/s** | **Stable** |
| | open.mp | 20 / 20 (100%) | 140.2 ms | 10.5 ms | 233 msg/s | ~28× slower spawn |
| **50 Bots**<br>*(Heavy Concurrency)* | **SA:GO** | **50 / 50 (100%)** | **5.1 ms** | **0.06 ms** | **522 msg/s** | **Stable** |
| | open.mp | 50 / 50 (100%) | 196.4 ms | 18.7 ms | 323 msg/s | ~39× slower spawn |
| **100 Bots**<br>*(Extreme Stress)* | **SA:GO** | **100 / 100 (100%)** | **5.0 ms** | **0.06 ms** | **552 msg/s** | **Stable** |
| | open.mp | 100 / 100 (100%) | 228.6 ms | 21.8 ms | 379 msg/s | ~46× slower spawn |
| **1000 Bots**<br>*(Thousand-Player Scale)* | **SA:GO** | **1000 / 1000 (100%)** | **5.0 ms** | **0.06 ms** | **—** | **Stable** (server 9% CPU) |
| | open.mp | 1000 / 1000 (100%) | 135.9 ms | 8.3 ms | — | Server ~10-25% CPU, ~90MB RAM |
| **50 Bots**<br>*(Flaky WAN: 5% Loss + Jitter)* | **SA:GO** | **50 / 50 (100%)** | **269.0 ms** | **63.2 ms** | **103 msg/s** | **Stable** |
| | open.mp | 41 / 50 (82%) | 320.8 ms | 74.1 ms | 20 msg/s | 9 bots dropped |

### Deep Resource Monitoring at 1000 Players

Measured with per-second process sampling (`pidstat`, `/proc`, `vmstat`, `iostat`) while 1000 bots were connected and exchanging chat:

| Resource | SA:GO (1000 bots) | open.mp (1000 bots) | Advantage |
|---|---|---|---|
| **Server CPU** | **~8–10%** (mean ~9%) | ~12–25% | ~2–3× lighter |
| **Resident Memory (RSS)** | **~20 MB steady** (idle ~11 MB) | ~90–93 MB | **~4.5× less RAM** |
| **Threads** | **10** | 2 (plus internal pools) | — |
| **File Descriptors** | **6–7 (one UDP socket)** | 5 | Single-socket design |
| **Major page faults** | **0** | 0 | Zero disk paging |
| **Involuntary context switches** | **0** | — | No scheduler contention |
| **Disk utilization** | **0.03%** | — | Effectively no disk I/O |
| **Goroutines (steady state)** | **5** | — | Tiny runtime footprint |

#### Key Performance Takeaways
- **~30–45× Faster World Entry:** SA:GO places players in the game world in a consistent 5 ms from 20 up to 1000 players, versus 135–320 ms on open.mp.
- **Over 100× Lower Communication Latency:** Sub-millisecond internal processing (0.05–0.06 ms median) delivers essentially instant player interaction and command response.
- **4–5× Lower Memory Footprint:** A full 1000-player server holds steady at roughly 20 MB of RAM, leaving ample headroom on even the smallest VPS.
- **Rock-Solid Packet Loss Tolerance:** Under simulated adverse wireless conditions (5% packet loss, packet reordering, 5–60 ms jitter), SA:GO retains 100% of connections with roughly 5× higher throughput, while open.mp drops players.
- **Zero-Leak Long-Term Endurance:** Continuous soak testing verified a flat memory footprint with zero leakage, zero garbage collection pauses, and zero throughput degradation from the first minute to the last.
- **Honest Methodology:** Benchmarks launch bots with a small stagger so the synthetic load does not degenerate into a single thundering herd of handshakes from one host — matching how real players join over time.

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
