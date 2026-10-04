# SA:GO Scripting API Reference

This document provides a comprehensive reference for the Lua scripting environment in SA:GO. All native functions are exposed under the global `sa.*` namespace (with legacy compatibility aliases `gosamp.*` and `ffps.*`).

---

## Table of Contents
1. [Core Principles](#core-principles)
2. [Player Functions](#player-functions)
3. [Vehicle Functions](#vehicle-functions)
4. [Weapon Functions](#weapon-functions)
5. [User Interface: Dialogs](#user-interface-dialogs)
6. [User Interface: TextDraws](#user-interface-textdraws)
7. [World: Checkpoints](#world-checkpoints)
8. [World: Objects](#world-objects)
9. [World: Actors (NPCs)](#world-actors-npcs)
10. [World: Gang Zones](#world-gang-zones)
11. [World: 3D Text Labels](#world-3d-text-labels)
12. [World: Map Icons](#world-map-icons)
13. [Camera & Cinematics](#camera--cinematics)
14. [Database Operations](#database-operations)
15. [Cryptography & Security](#cryptography--security)
16. [Logging & Utility](#logging--utility)
17. [Callback Events](#callback-events)
18. [Constants Reference](#constants-reference)

---

## Core Principles

- **Events convey facts; natives execute decisions.** The server engine synchronizes raw networking data and dispatches events to Lua. Gamemodes make gameplay decisions and invoke natives to apply changes.
- **Fail-safe execution:** Runtime Lua errors are logged cleanly without crashing the server or kicking players.
- **Non-blocking asynchronous I/O:** Long-running database operations provide asynchronous variants with callbacks that execute on the server tick loop.

---

## Player Functions

### `sa.PlayerName(playerID)`
Returns the connected player's nickname.
- **Parameters:** `playerID` (integer)
- **Returns:** `name` (string), `ok` (boolean)

### `sa.PlayerCount()`
Returns the total number of connected players.
- **Returns:** `count` (integer)

### `sa.SetPlayerPos(playerID, x, y, z)`
Teleports a player to world coordinates.
- **Parameters:** `playerID` (integer), `x, y, z` (floats)
- **Returns:** `ok` (boolean)

### `sa.GetPlayerPos(playerID)`
Retrieves a player's last recorded world position.
- **Parameters:** `playerID` (integer)
- **Returns:** `x, y, z` (floats)

### `sa.SetPlayerFacingAngle(playerID, angle)`
Sets the player's horizontal facing angle (0 to 360 degrees).
- **Parameters:** `playerID` (integer), `angle` (float)
- **Returns:** `ok` (boolean)

### `sa.SetPlayerHealth(playerID, health)`
Sets the player's health (0.0 to 100.0).
- **Parameters:** `playerID` (integer), `health` (float)
- **Returns:** `ok` (boolean)

### `sa.SetPlayerArmour(playerID, armour)`
Sets the player's armour level (0.0 to 100.0).
- **Parameters:** `playerID` (integer), `armour` (float)
- **Returns:** `ok` (boolean)

### `sa.SetPlayerSkin(playerID, skinID)`
Changes the player's character skin model.
- **Parameters:** `playerID` (integer), `skinID` (integer)
- **Returns:** `ok` (boolean)

### `sa.SetPlayerTeam(playerID, teamID)`
Assigns the player to a team index.
- **Parameters:** `playerID` (integer), `teamID` (integer)
- **Returns:** `ok` (boolean)

### `sa.SetSpawnInfo(playerID, team, skin, x, y, z, angle)`
Sets the spawn configuration used when the player spawns.
- **Parameters:** `playerID` (int), `team` (int), `skin` (int), `x, y, z, angle` (floats)
- **Returns:** `ok` (boolean)

### `sa.ForceClassSelection(playerID)`
Forces the player into the character/class selection screen.
- **Parameters:** `playerID` (integer)
- **Returns:** `ok` (boolean)

### `sa.SetPlayerVelocity(playerID, vx, vy, vz)`
Applies directional velocity to the player.
- **Parameters:** `playerID` (integer), `vx, vy, vz` (floats)
- **Returns:** `ok` (boolean)

### `sa.RemovePlayerFromVehicle(playerID)`
Ejects the player from their current vehicle.
- **Parameters:** `playerID` (integer)
- **Returns:** `ok` (boolean)

### `sa.SetPlayerAdmin(playerID, isAdmin)`
Marks or clears a player's administrative status.
- **Parameters:** `playerID` (integer), `isAdmin` (boolean)
- **Returns:** `ok` (boolean)

### `sa.IsPlayerAdmin(playerID)`
Checks if a player has administrative privileges.
- **Parameters:** `playerID` (integer)
- **Returns:** `isAdmin` (boolean)

### `sa.Kick(playerID)` / `sa.KickPlayer(playerID, [reason])`
Disconnects a player with an optional reason logged.
- **Parameters:** `playerID` (integer), `reason` (string, optional)
- **Returns:** `ok` (boolean)

### `sa.Ban(playerID)` / `sa.BanPlayer(playerID, [reason])`
Bans a player's IP address with an optional reason logged.
- **Parameters:** `playerID` (integer), `reason` (string, optional)
- **Returns:** `ok` (boolean)

---

## Vehicle Functions

### `sa.CreateVehicle(modelID, x, y, z, angle, color1, color2)`
Spawns a new vehicle in the game world.
- **Parameters:** `modelID` (int, 400–611), `x, y, z, angle` (floats), `color1, color2` (ints, 0–255)
- **Returns:** `vehicleID` (integer) or `nil` on failure

### `sa.DestroyVehicle(vehicleID)`
Removes a spawned vehicle from the world.
- **Parameters:** `vehicleID` (integer)
- **Returns:** `ok` (boolean)

### `sa.SetVehicleHealth(vehicleID, health)`
Sets vehicle engine health (typically 0.0 to 1000.0).
- **Parameters:** `vehicleID` (integer), `health` (float)
- **Returns:** `ok` (boolean)

### `sa.GetVehicleHealth(vehicleID)`
Returns the current engine health of a vehicle.
- **Parameters:** `vehicleID` (integer)
- **Returns:** `health` (float)

### `sa.SetVehicleDamageStatus(vehicleID, panels, doors, lights, tyres)`
Sets visual and mechanical vehicle damage components.
- **Parameters:** `vehicleID` (int), `panels` (int), `doors` (int), `lights` (int), `tyres` (int)
- **Returns:** `ok` (boolean)

### `sa.GetVehicleDamageStatus(vehicleID)`
Retrieves vehicle damage bitmasks.
- **Parameters:** `vehicleID` (integer)
- **Returns:** `panels, doors, lights, tyres` (integers)

---

## Weapon Functions

### `sa.GivePlayerWeapon(playerID, weaponID, ammo)`
Adds a weapon with ammunition to a player's inventory.
- **Parameters:** `playerID` (integer), `weaponID` (integer), `ammo` (integer)
- **Returns:** `ok` (boolean)

### `sa.SetPlayerAmmo(playerID, weaponID, ammo)`
Updates the ammunition count for an existing weapon.
- **Parameters:** `playerID` (integer), `weaponID` (integer), `ammo` (integer)
- **Returns:** `ok` (boolean)

### `sa.ResetPlayerWeapons(playerID)`
Clears all weapons and ammunition from the player.
- **Parameters:** `playerID` (integer)
- **Returns:** `ok` (boolean)

### `sa.GetPlayerWeaponData(playerID, slot)`
Reads weapon ID and ammo stored in a specific inventory slot (0 to 12).
- **Parameters:** `playerID` (integer), `slot` (integer)
- **Returns:** `weaponID` (integer), `ammo` (integer)

### `sa.GetWeaponName(weaponID)`
Returns the official display name of a weapon.
- **Parameters:** `weaponID` (integer)
- **Returns:** `name` (string)

---

## User Interface: Dialogs

### `sa.ShowPlayerDialog(playerID, dialogID, style, title, body, button1, button2)`
Displays an interactive modal dialog box to a player.
- **Parameters:**
  - `playerID` (integer)
  - `dialogID` (integer, developer-defined)
  - `style` (integer: 0=MsgBox, 1=Input, 2=List, 3=Password, 4=Tablist, 5=TablistHeaders)
  - `title` (string)
  - `body` (string)
  - `button1` (string, primary)
  - `button2` (string, secondary/cancel)
- **Returns:** `ok` (boolean)

### `sa.HidePlayerDialog(playerID)`
Closes any active dialog prompt for the player.
- **Parameters:** `playerID` (integer)
- **Returns:** `ok` (boolean)

---

## Global Server Flags

### `sa.DisableInteriorEnterExits()`
Sets the global `InitGame` (139) `DisableEnterExits` flag, removing the yellow
interior enter/exit markers everywhere — the SA:GO equivalent of SA-MP's
`DisableInteriorEnterExits()`.

- **Returns:** `ok` (boolean)

**Important behavior — call it once, at the top of your gamemode:**
- The flag is **global** and is carried in the `InitGame` packet the server
  sends to each player on connect.
- Changing it only affects players who connect **after** the call. Players who
  are already connected are **not** affected (their client keeps the setting it
  received at join time). This mirrors the original SA-MP semantics exactly.
- Therefore it should be invoked **once during gamemode initialization**, never
  mid-game.

```lua
-- top of your gamemode, before any player has connected
sa.DisableInteriorEnterExits()
```

---

## Player Primitives

Generic natives covering the full SA-MP "Player" RPC set. The core only sends
the packet and keeps minimal bookkeeping; your gamemode decides when and why.

### Identity & appearance
- `sa.SetPlayerName(playerID, name)`: Renames the player (RPC 11, broadcast).
- `sa.SetPlayerColor(playerID, colorRGBA)`: Sets nametag color (RPC 72).
- `sa.ShowPlayerNameTagForPlayer(playerID, showPlayerID, show)`: Toggles a target's nametag for a viewer (RPC 80).
- `sa.SetPlayerSkin(playerID, skinID)`: Changes the skin (RPC 153).

### Position & movement
- `sa.SetPlayerPos(playerID, x, y, z)`: Teleports (RPC 12).
- `sa.SetPlayerPosFindZ(playerID, x, y, z)`: Teleports, client finds ground Z (RPC 13).
- `sa.SetPlayerFacingAngle(playerID, angle)`: Facing angle (RPC 19).
- `sa.SetPlayerVelocity(playerID, x, y, z)`: Velocity (RPC 90).
- `sa.TogglePlayerControllable(playerID, controllable)`: Locks/unlocks control (RPC 15).

### Status & condition
- `sa.SetPlayerSkillLevel(playerID, skillID, level)`: Weapon skill level (RPC 34).
- `sa.SetPlayerDrunkLevel(playerID, level)`: Drunk level (RPC 35).
- `sa.SetPlayerWantedLevel(playerID, level)`: Wanted stars (RPC 133).
- `sa.SetPlayerFightingStyle(playerID, style)`: Fighting style (RPC 89).
- `sa.SetPlayerSpecialAction(playerID, actionID)`: Special action (RPC 88).

### Money
- `sa.GivePlayerMoney(playerID, amount)`: Adds/subtracts money (RPC 18, amount may be negative).
- `sa.ResetPlayerMoney(playerID)`: Resets money to 0 (RPC 20).

### Animation
- `sa.ApplyPlayerAnimation(playerID, animLib, animName, delta, loop, lockX, lockY, freeze, time)`: Plays an animation (RPC 86).
- `sa.ClearPlayerAnimation(playerID)`: Stops the animation (RPC 87).

### Per-player world & environment
- `sa.SetPlayerTime(playerID, hour, minute)`: Personal time (RPC 29).
- `sa.TogglePlayerClock(playerID, toggle)`: Shows the clock (RPC 30).
- `sa.SetPlayerInterior(playerID, interiorID)`: Interior (RPC 156).
- `sa.SetPlayerWorldBounds(playerID, maxX, minX, maxY, minY)`: World bounds (RPC 17).
- `sa.TogglePlayerSpectating(playerID, toggle)`: Spectate mode (RPC 124).
- `sa.PlayerSpectatePlayer(playerID, targetPlayerID, [mode])`: Spectate a player (RPC 126, default mode 1).
- `sa.PlayerSpectateVehicle(playerID, vehicleID, [mode])`: Spectate a vehicle (RPC 127, default mode 1).
- `sa.TogglePlayerWidescreen(playerID, enable)`: Widescreen (RPC 111).

### Sound & messages
- `sa.PlayerPlaySound(playerID, soundID, x, y, z)`: Plays a sound (RPC 16).
- `sa.PlayCrimeReportForPlayer(playerID, suspectID, inVehicle, vehicleModel, vehicleColor, crime, x, y, z)`: Crime report (RPC 112).

### Misc
- `sa.ForceClassSelection(playerID)`: Forces class selection (RPC 74).
- `sa.SetPlayerShopName(playerID, name)`: Shop name (RPC 33, fixed 32-byte field; empty unloads).
- `sa.CreateExplosionForPlayer(playerID, x, y, z, type, radius)`: Explosion for one player (RPC 79).
- `sa.RemoveBuildingForPlayer(playerID, modelID, x, y, z, radius)`: Removes a building for one player (RPC 43).
- `sa.SetPlayerScore(playerID, score)`: Sets score and broadcasts the score/ping table (RPC 155). Ping is taken from the connection state.

Constants: `WEAPONSKILL_*` (0..10), `FIGHT_STYLE_*`, `SPECIAL_ACTION_*`,
`SPECTATE_MODE_NORMAL/FIXED/SIDE` (1/2/3).

---

## User Interface: TextDraws

TextDraws operate on the canonical 640.0 x 480.0 virtual canvas.

### Global TextDraws (Shared pool: IDs 0–2047)
- `sa.TextDrawCreate(id, x, y, text)`: Creates a global TextDraw.
- `sa.TextDrawDestroy(id)`: Removes a global TextDraw.
- `sa.TextDrawFont(id, font)`: Sets typography font (0–3, 4=sprite, 5=preview).
- `sa.TextDrawLetterSize(id, x, y)`: Sets font character width and height.
- `sa.TextDrawTextSize(id, x, y)`: Sets bounding box or click area size.
- `sa.TextDrawAlignment(id, alignment)`: Alignment (1=left, 2=center, 3=right).
- `sa.TextDrawColor(id, color)`: Primary text RGBA color (e.g. `0xFFFFFFFF`).
- `sa.TextDrawBackgroundColor(id, color)`: Outline / background RGBA color.
- `sa.TextDrawBoxColor(id, color)`: Background box fill RGBA color.
- `sa.TextDrawUseBox(id, useBox)`: Enables or disables the background box.
- `sa.TextDrawSetShadow(id, size)`: Sets shadow blur depth.
- `sa.TextDrawSetOutline(id, size)`: Sets text stroke outline size.
- `sa.TextDrawSetProportional(id, proportional)`: Enables proportional letter spacing.
- `sa.TextDrawSetSelectable(id, selectable)`: Makes the TextDraw clickable via cursor.
- `sa.TextDrawSetPreviewModel(id, model)`: Sets preview model index.
- `sa.TextDrawSetPreviewRot(id, rx, ry, rz, zoom)`: Sets preview model 3D rotation and zoom.
- `sa.TextDrawSetString(id, text)`: Updates text content dynamically.
- `sa.TextDrawShowForPlayer(playerID, id)`: Displays TextDraw to a specific player.
- `sa.TextDrawHideForPlayer(playerID, id)`: Hides TextDraw from a specific player.
- `sa.TextDrawShowForAll(id)`: Displays TextDraw to all connected players.
- `sa.TextDrawHideForAll(id)`: Hides TextDraw from all players.
- `sa.SelectTextDraw(playerID, hoverColor)`: Toggles mouse cursor selection mode.

### Player TextDraws (Per-player pool: IDs 0–255)
Every global TextDraw method has a matching `sa.PlayerTextDraw*` variant accepting `playerID` as its first parameter (e.g., `sa.PlayerTextDrawCreate(playerID, id, x, y, text)`).

---

## World: Checkpoints

### `sa.SetPlayerCheckpoint(playerID, x, y, z, radius)`
Displays a standard red cylinder checkpoint.
- **Parameters:** `playerID` (integer), `x, y, z` (floats), `radius` (float)
- **Returns:** `ok` (boolean)

### `sa.DisablePlayerCheckpoint(playerID)`
Removes the active standard checkpoint.
- **Parameters:** `playerID` (integer)
- **Returns:** `ok` (boolean)

### `sa.SetPlayerRaceCheckpoint(playerID, type, x, y, z, nextX, nextY, nextZ, radius)`
Displays a directional race checkpoint with directional arrows.
- **Parameters:** `playerID` (int), `type` (int, 0–8), `x, y, z` (floats), `nextX, nextY, nextZ` (floats), `radius` (float)
- **Returns:** `ok` (boolean)

### `sa.DisablePlayerRaceCheckpoint(playerID)`
Removes the active race checkpoint.
- **Parameters:** `playerID` (integer)
- **Returns:** `ok` (boolean)

---

## World: Objects

Object pool supports up to 2,000 global models (IDs 0–1999).

- `sa.CreateObject(model, x, y, z, rx, ry, rz, drawDistance)`: Spawns a world object.
- `sa.DestroyObject(objectID)`: Deletes an object.
- `sa.SetObjectPos(objectID, x, y, z)`: Teleports an object.
- `sa.SetObjectRotation(objectID, rx, ry, rz)`: Rotates an object.
- `sa.MoveObject(objectID, toX, toY, toZ, speed, rotX, rotY, rotZ)`: Smoothly interpolates an object to a new location.
- `sa.StopObject(objectID)`: Halts ongoing object movement.

### Attached Objects (RPC 113)

An attached object is a model pinned to one of a player's bones (hat on the
head, weapon on the back, ...). Each player has **10 slots** (index 0–9, since
SA-MP 0.3d). Attach and remove share a single RPC distinguished by a `create`
flag: a create packet carries the full attachment data, a remove packet carries
only the opening fields (shorter payload).

- `sa.SetPlayerAttachedObject(playerID, index, modelID, bone, [offsetX, offsetY, offsetZ, rotX, rotY, rotZ, scaleX, scaleY, scaleZ, color1, color2])`: Attaches a model to a bone (RPC 113, create=true). Defaults: offsets/rotations `0.0`, scale `1.0`, colors `0` (native model colors). Returns `ok` (boolean).
- `sa.RemovePlayerAttachedObject(playerID, index)`: Clears a slot (RPC 113, create=false). Returns `ok` (boolean).
- `sa.IsPlayerAttachedObjectSlotUsed(playerID, index)`: Reads core bookkeeping for whether a slot is occupied. Returns `used` (boolean).

An `index` outside **0..9** raises a Lua error instead of sending a packet the
client would misinterpret. The packet is streamed to nearby players (including
the target). `sa.MAX_ATTACHED_OBJECT_SLOTS` = `10`.

#### Bone IDs (official SA-MP set: only 1..18)

| ID | Bone | Lua constant |
|---|---|---|
| 1 | Spine | `sa.BONE_SPINE` |
| 2 | Head | `sa.BONE_HEAD` |
| 3 | Left upper arm | `sa.BONE_LEFT_UPPER_ARM` |
| 4 | Right upper arm | `sa.BONE_RIGHT_UPPER_ARM` |
| 5 | Left hand | `sa.BONE_LEFT_HAND` |
| 6 | Right hand | `sa.BONE_RIGHT_HAND` |
| 7 | Left thigh | `sa.BONE_LEFT_THIGH` |
| 8 | Right thigh | `sa.BONE_RIGHT_THIGH` |
| 9 | Left foot | `sa.BONE_LEFT_FOOT` |
| 10 | Right foot | `sa.BONE_RIGHT_FOOT` |
| 11 | Right calf | `sa.BONE_RIGHT_CALF` |
| 12 | Left calf | `sa.BONE_LEFT_CALF` |
| 13 | Left forearm | `sa.BONE_LEFT_FOREARM` |
| 14 | Right forearm | `sa.BONE_RIGHT_FOREARM` |
| 15 | Left clavicle (shoulder) | `sa.BONE_LEFT_CLAVICLE` |
| 16 | Right clavicle (shoulder) | `sa.BONE_RIGHT_CLAVICLE` |
| 17 | Neck | `sa.BONE_NECK` |
| 18 | Jaw | `sa.BONE_JAW` |

> Source: SA-MP Wiki "Bone IDs" and open.mp docs `scripting/resources/boneid`
> (identical). SA-MP defines bones only 1..18.

```lua
sa.SetPlayerAttachedObject(playerid, 0, 1609, sa.BONE_HEAD) -- hat on head
if sa.IsPlayerAttachedObjectSlotUsed(playerid, 0) then
    sa.RemovePlayerAttachedObject(playerid, 0)
end
```

---

## World: Actors (NPCs)

Actors are ambient non-player peds that do not consume player slots (IDs 0–999).

- `sa.CreateActor(model, x, y, z, angle)`: Spawns an actor ped.
- `sa.DestroyActor(actorID)`: Deletes an actor ped.
- `sa.ApplyActorAnimation(actorID, animLib, animName, delta, loop, lockX, lockY, freeze, time)`: Plays an animation.
- `sa.ClearActorAnimation(actorID)`: Cancels active animations.
- `sa.SetActorFacingAngle(actorID, angle)`: Rotates the actor.
- `sa.SetActorPos(actorID, x, y, z)`: Repositions the actor.
- `sa.SetActorHealth(actorID, health)`: Sets actor health.

---

## World: Gang Zones

- `sa.GangZoneCreate(minX, minY, maxX, maxY)`: Defines a radar bounding box.
- `sa.GangZoneDestroy(zoneID)`: Removes a gang zone.
- `sa.GangZoneShowForPlayer(playerID, zoneID, color)`: Displays zone with RGBA color.
- `sa.GangZoneShowForAll(zoneID, color)`: Displays zone to all players.
- `sa.GangZoneHideForPlayer(playerID, zoneID)`: Hides zone for a player.
- `sa.GangZoneFlashForPlayer(playerID, zoneID, flashColor)`: Initiates flashing animation.
- `sa.GangZoneFlashForAll(zoneID, flashColor)`: Initiates flashing for all players.
- `sa.GangZoneStopFlashForPlayer(playerID, zoneID)`: Stops flashing animation.
- `sa.GangZoneStopFlashForAll(zoneID)`: Stops flashing for all players.

---

## World: 3D Text Labels

- `sa.Create3DTextLabel(text, color, x, y, z, drawDistance, worldID, testLOS)`: Creates a world floating label.
- `sa.Create3DTextLabelAttachedToPlayer(text, color, playerID, ox, oy, oz, drawDistance, testLOS)`: Attaches label to a player.
- `sa.Create3DTextLabelAttachedToVehicle(text, color, vehicleID, ox, oy, oz, drawDistance, testLOS)`: Attaches label to a vehicle.
- `sa.Delete3DTextLabel(labelID)`: Removes a 3D label.

---

## World: Map Icons

- `sa.SetPlayerMapIcon(playerID, iconID, x, y, z, markerType, color, style)`: Creates a radar map icon (0–99).
- `sa.RemovePlayerMapIcon(playerID, iconID)`: Removes a radar map icon.

---

## Camera & Cinematics

- `sa.SetPlayerCameraPos(playerID, x, y, z)`: Places the camera at fixed world coordinates.
- `sa.SetPlayerCameraLookAt(playerID, x, y, z, cutType)`: Aims the camera (cutType: 1=instant, 2=smooth).
- `sa.InterpolateCameraPos(playerID, fx, fy, fz, tx, ty, tz, timeMs, cutType)`: Glides camera position over time.
- `sa.InterpolateCameraLookAt(playerID, fx, fy, fz, tx, ty, tz, timeMs, cutType)`: Glides camera focus target over time.
- `sa.SetPlayerCameraBehindPlayer(playerID)`: Returns the camera to normal follow mode.

---

## Database Operations

SA:GO includes high-performance built-in SQLite database support with prepared statements.

### Synchronous (Blocking)
- `sa.sqlite_open(path)`: Opens or creates a database file. Returns `handle`.
- `sa.sqlite_close(handle)`: Closes an open database handle.
- `sa.sqlite_query(handle, sql, ...)`: Executes a `SELECT` query. Returns a Lua array of row tables.
- `sa.sqlite_exec(handle, sql, ...)`: Executes `INSERT`, `UPDATE`, or `DELETE`. Returns `rowsAffected, lastInsertID, err`.

### Asynchronous (Non-blocking worker)
- `sa.sqlite_query_async(handle, sql, callback, ...)`: Runs query in background worker; invokes `callback(rows, err)` on main tick.
- `sa.sqlite_exec_async(handle, sql, callback, ...)`: Runs exec in background worker; invokes `callback(affected, lastID, err)` on main tick.

```lua
-- Example: Safe async query
sa.sqlite_query_async(DB, "SELECT skin, money FROM accounts WHERE name = ?", function(rows, err)
    if rows and rows[1] then
        sa.Log("Loaded skin: " .. rows[1].skin)
    end
end, playerName)
```

---

## Cryptography & Security

- `sa.hash_password(plaintext)`: Generates a secure bcrypt password hash (`$2a$...`).
- `sa.verify_password(plaintext, hash)`: Verifies plaintext password against a bcrypt hash. Returns boolean.
- `sa.random_token([bytes])`: Generates a cryptographically random hex token string (default 16 bytes).

---

## Logging & Utility

- `sa.Log(text, ...)`: Outputs an informational message to the server console and log file.
- `sa.SendClientMessage(playerID, colorRGBA, message)`: Sends colored text to a player (RPC 93).
- `sa.SendClientMessageToAll(colorRGBA, message)`: Broadcasts colored text to all players.

## Messaging, GameText, Death Feed & Chat Bubbles

### `sa.SendPlayerMessageToPlayer(playerID, senderID, text)`
Sends a chat-style line (RPC 101) to one player as if spoken by `senderID`.
- **Returns:** `ok` (boolean)

### `sa.SendPlayerMessageToAll(senderID, text)`
Broadcasts a chat-style line from `senderID` to every player.

### `sa.GameTextForPlayer(playerID, style, timeMS, text)`
Shows styled on-screen text (RPC 73) to one player for `timeMS` milliseconds.
- **Parameters:** `style` (see GameText styles), `timeMS` (integer)
- **Returns:** `ok` (boolean)

### `sa.GameTextForAll(style, timeMS, text)`
Shows styled on-screen text to every player.

### `sa.SendDeathMessage(killerID, killeeID, reason)`
Broadcasts a kill-feed entry (RPC 55) to all players. `reason` is a weapon id; pass `-1` for killer when there is none (mapped to the 0xFFFF sentinel).
- **Returns:** `ok` (boolean)

### `sa.SendDeathMessageToPlayer(playerID, killerID, killeeID, reason)`
Sends a kill-feed entry to a single player.
- **Returns:** `ok` (boolean)

### `sa.SetPlayerChatBubble(playerID, text, colorRGBA, drawDistance, expireMS)`
Shows a 3D chat bubble (RPC 59) above a player for `expireMS` milliseconds, streamed to players within `drawDistance` units. A `drawDistance` of 0 defaults to 100.
- **Returns:** `ok` (boolean)

#### GameText Styles
- `sa.GAMETEXT_STYLE_CENTER` = `3`
- `sa.GAMETEXT_STYLE_LIGHT` = `4`
- `sa.GAMETEXT_STYLE_MIDDLE` = `5`
- `sa.GAMETEXT_STYLE_INFO` = `6`

---

## Callback Events

To handle game events, simply define these functions in your Lua gamemode.

### Connection & Authentication
- `onPlayerConnect(playerID)`: Fired when a player establishes a connection.
- `onPlayerDisconnect(playerID, reason)`: Fired on disconnect (reason: 0=timeout, 1=quit, 2=kicked).
- `onPlayerRequestClass(playerID, classID)`: Return `false` to deny class selection.
- `onPlayerRequestClassDone(playerID, classID, skin)`: Fired after engine confirms class preview; place character and camera here.
- `onPlayerRequestSpawn(playerID)`: Return `false` to prevent spawning.
- `onPlayerSpawn(playerID)`: Fired when the player enters the world.
- `onPlayerDeath(playerID, killerID, reason)`: Fired when a player dies.

### Interaction & Chat
- `onPlayerText(playerID, text)`: Return `true` to suppress default chat relay.
- `onPlayerCommand(playerID, command, params)`: Return `true` if command handled; unhandled commands show "Unknown command".
- `onDialogResponse(playerID, dialogID, response, listItem, inputText)`: Fired on dialog interaction.
- `onPlayerClickTextDraw(playerID, textDrawID)`: Fired when clicking a selectable TextDraw (id 65535 = cancelled).

### Gameplay & Synchronization
- `onPlayerUpdate(playerID, data)`: Fired on movement tick. `data` table contains: `newPos`, `oldPos`, `velocity`, `deltaTime`, `health`, `armour`, `weapon`, `keys`, `inVehicle`, `vehicleID`, `teleported`.
- `onPlayerAim(playerID, data)`: Fired on weapon aiming vector update.
- `onPlayerWeaponShot(playerID, weaponID, hitType, hitID, hitX, hitY, hitZ, originX, originY, originZ, offsetX, offsetY, offsetZ)`: Fired when a bullet is discharged.
- `onPlayerGiveTakeDamage(playerID, giveOrTake, targetID, amount, weaponID, bodyPart)`: Fired on combat damage.
- `onPlayerStats(playerID, money, drunkLevel)`: Fired on player stat updates.

### Vehicle Events
- `onPlayerEnterVehicle(playerID, vehicleID, isPassenger)`: Player begins entering a vehicle.
- `onPlayerExitVehicle(playerID, vehicleID)`: Player exits a vehicle.
- `onVehicleDamageStatusUpdate(vehicleID, playerID, panels, doors, lights, tyres)`: Vehicle sustains visual/mechanical damage.
- `onVehicleDeath(vehicleID, killerID)`: Vehicle reaches zero health.
- `onVehicleSync(playerID, data)`: Raw driver synchronization frame.
- `onUnoccupiedVehicleSync(playerID, data)`: Raw unoccupied vehicle synchronization.
- `onTrailerSync(playerID, data)`: Raw trailer synchronization frame.

### Checkpoints & World
- `onPlayerEnterCheckpoint(playerID)`: Player enters standard checkpoint radius.
- `onPlayerLeaveCheckpoint(playerID)`: Player exits standard checkpoint.
- `onPlayerEnterRaceCheckpoint(playerID)`: Player enters race checkpoint radius.
- `onPlayerLeaveRaceCheckpoint(playerID)`: Player exits race checkpoint.
- `onActorDamage(actorID, playerID, amount, weaponID, bodyPart)`: Player damages an ambient actor NPC.

---

## Constants Reference

### Dialog Styles
- `sa.DIALOG_STYLE_MSGBOX` = `0`
- `sa.DIALOG_STYLE_INPUT` = `1`
- `sa.DIALOG_STYLE_LIST` = `2`
- `sa.DIALOG_STYLE_PASSWORD` = `3`
- `sa.DIALOG_STYLE_TABLIST` = `4`
- `sa.DIALOG_STYLE_TABLIST_HEADERS` = `5`

### Map Icons
- `sa.MAPICON_LOCAL` = `0`
- `sa.MAPICON_GLOBAL` = `1`
- `sa.MAPICON_LOCAL_CHECKPOINT` = `2`
- `sa.MAPICON_GLOBAL_CHECKPOINT` = `3`

### Checkpoint Types
- `sa.CP_TYPE_NORMAL` = `0`
- `sa.CP_TYPE_FINISH` = `1`
- `sa.CP_TYPE_NOTHING` = `2`
- `sa.CP_TYPE_AIR_NORMAL` = `3`
- `sa.CP_TYPE_AIR_FINISH` = `4`
