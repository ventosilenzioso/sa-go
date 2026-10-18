-- ============================================================
-- SA:GO Lua Gamemode Skeleton
-- ============================================================
-- Kerangka gamemode: semua callback SA-MP + pola umum yang paling
-- sering dipakai. Hapus/isi bagian yang tidak perlu.
--
-- Aturan kontrak engine:
--   * Callback menerima FAKTA MENTAH. Gamemode yang memutuskan.
--   * Callback `on*` yang mengembalikan `false` = veto (connect/class/spawn/text).
--   * Runtime error di Lua TIDAK men-kick pemain; engine mencatat dan lanjut.
--   * Semua native lewat table global `sa` (mis. sa.SendClientMessage).
-- ============================================================

-- ------------------------------------------------------------
-- 1. KONFIGURASI (konstanta di Lua, bukan di core)
-- ------------------------------------------------------------
local SPAWN = { x = 1958.4, y = 1343.2, z = 15.4, angle = 270.0 }
local SKINS = { 0, 7, 21, 23, 29, 100, 250, 287 }

local DIALOG_LOGIN    = 100
local DIALOG_REGISTER = 101

-- ------------------------------------------------------------
-- 2. STATE (tabel keyed by playerID)
-- ------------------------------------------------------------
local loggedIn = {}
local score    = {}
local playerClass = {}

local function clearState(id)
    loggedIn[id]    = nil
    score[id]       = nil
    playerClass[id] = nil
end

-- ------------------------------------------------------------
-- 3. INIT (dijalankan sekali saat gamemode start)
-- ------------------------------------------------------------
sa.DisableInteriorEnterExits()          -- HARUS di awal (hanya koneksi baru)
sa.SetGameModeText("SA:GO Skeleton")
sa.ShowNameTags(true)
sa.ShowPlayerMarkers(1)                  -- 0=off 1=global 2=streamed
sa.SetWeather(10)
sa.SetWorldTime(12)
sa.SetGravity(0.008)

-- Kelas spawn (mirip AddPlayerClass di Pawn)
for _, skin in ipairs(SKINS) do
    sa.AddPlayerClass(skin, SPAWN.x, SPAWN.y, SPAWN.z, SPAWN.angle,
        sa.WEAPON_COLT45, 50, 0, 0, 0, 0)
end

-- Contoh timer berulang tiap 1000 ms (resolved by-name saat fire)
function onOneSecond()
    -- contoh: broadcast waktu tiap pemain terhubung
    for id in pairs(loggedIn) do
        -- sa.SendClientMessage(id, 0xFFFFFFFF, "tick")
    end
end
local timerID = sa.SetTimer("onOneSecond", 1000, true)

-- ------------------------------------------------------------
-- 4. CALLBACK KONEKSI & AUTENTIKASI
-- ------------------------------------------------------------
function onPlayerConnect(id)
    loggedIn[id] = false
    score[id] = 0
    sa.SendClientMessage(id, 0xFFFF00FF, "Selamat datang di SA:GO!")
    -- Tampilkan dialog login/register (lihat ShowPlayerDialog)
    sa.ShowPlayerDialog(id, DIALOG_LOGIN, sa.DIALOG_STYLE_PASSWORD,
        "Login", "Masukkan password:", "Login", "Keluar")
end

function onPlayerDisconnect(id, reason)
    clearState(id)
end

function onDialogResponse(id, dialogID, response, listItem, inputText)
    if dialogID == DIALOG_LOGIN then
        if response == 1 then
            -- TODO: verifikasi password. JANGAN simpan plaintext.
            loggedIn[id] = true
            sa.SendClientMessage(id, 0x00FF00FF, "Login berhasil.")
            sa.ForceClassSelection(id)
        else
            sa.Kick(id)
        end
    end
end

-- ------------------------------------------------------------
-- 5. CALLBACK CLASS SELECTION & SPAWN
-- ------------------------------------------------------------
function onPlayerRequestClass(id, classID)
    playerClass[id] = classID
    -- Kamera preview menghadap karakter
    sa.SetPlayerCameraPos(id, SPAWN.x, SPAWN.y - 4.0, SPAWN.z + 2.5)
    sa.SetPlayerCameraLookAt(id, SPAWN.x, SPAWN.y, SPAWN.z + 1.0)
    return true                              -- false = tolak kelas
end

function onPlayerRequestSpawn(id)
    return loggedIn[id] == true              -- false = tolak spawn
end

function onPlayerSpawn(id)
    sa.SetPlayerHealth(id, 100.0)
    sa.SetPlayerArmour(id, 0.0)
    sa.SetPlayerInterior(id, 0)
    sa.GivePlayerWeapon(id, sa.WEAPON_DEAGLE, 50)
    sa.SendClientMessage(id, 0xFFFFFFFF, "Kamu spawn. Skor: " .. (score[id] or 0))
end

-- ------------------------------------------------------------
-- 6. CALLBACK KEMATIAN & RESPAWN
-- ------------------------------------------------------------
function onPlayerDeath(id, killerID, reason)
    score[id] = (score[id] or 0) - 1
    sa.SendDeathMessage(killerID, id, reason)
    sa.SetTimerEx("respawnPlayer", 3000, false, "i", id)
end

function respawnPlayer(id)
    sa.ForceClassSelection(id)
end

-- ------------------------------------------------------------
-- 7. CHAT & COMMAND
-- ------------------------------------------------------------
function onPlayerText(id, text)
    return true                              -- false = blokir chat
end

function onPlayerCommand(id, command, params)
    if command == "heal" then
        sa.SetPlayerHealth(id, 100.0)
        sa.SendClientMessage(id, 0x00FF00FF, "Diheal.")
        return true                          -- true = sudah ditangani
    elseif command == "pos" then
        local x, y, z = sa.GetPlayerPos(id)
        sa.SendClientMessage(id, 0xFFFFFFFF,
            string.format("Posisi: %.1f, %.1f, %.1f", x, y, z))
        return true
    elseif command == "weapon" then
        local w = tonumber(params) or sa.WEAPON_DEAGLE
        sa.GivePlayerWeapon(id, w, 100)
        return true
    elseif command == "kickme" then
        sa.Kick(id)
        return true
    end
    return false                             -- false = biarkan engine?
end

-- ------------------------------------------------------------
-- 8. SYNC CALLBACKS (data mentah)
-- ------------------------------------------------------------
function onPlayerUpdate(id, data)
    -- data.newPos / oldPos {x,y,z}, data.deltaTime, data.health, data.armour,
    -- data.keys, data.teleported (true saat engine memindahkan pemain), dll.
    -- Contoh check sederhana: log posisi mencurigakan
    -- if data.health < 0 then sa.Kick(id) end
end

function onPlayerAim(id, data)
    -- data.camMode, data.camZoom, data.weaponState, data.camPos{x,y,z}, ...
end

function onPlayerWeaponShot(id, weaponID, hitType, hitID, hx, hy, hz, ox, oy, oz, offx, offy, offz)
    -- Fakta tembakan; validasi damage = kebijakan gamemode
end

function onPlayerStats(id, money, drunkLevel)
    -- update lokal
end

function onPlayerGiveTakeDamage(id, giveOrTake, targetID, amount, weaponID, bodyPart)
    -- return false untuk membatalkan? (lihat docs)
end

function onVehicleSync(id, data)
    -- data.vehicleID, data.x/y/z, data.velX/Y/Z, data.vehicleHealth, ...
end

function onUnoccupiedVehicleSync(id, data) end
function onTrailerSync(id, data) end
function onVehicleDamageStatusUpdate(vehicleID, playerID, panels, doors, lights, tyres) end
function onVehicleDeath(vehicleID, killerID) end

-- ------------------------------------------------------------
-- 9. ENTER/EXIT KENDARAAN & CHECKPOINT
-- ------------------------------------------------------------
function onPlayerEnterVehicle(id, vehicleID, isPassenger) end
function onPlayerExitVehicle(id, vehicleID) end

function onPlayerEnterCheckpoint(id)
    sa.SendClientMessage(id, 0xFFFF00FF, "Checkpoint tercapai!")
    sa.DisablePlayerCheckpoint(id)
end
function onPlayerLeaveCheckpoint(id) end
function onPlayerEnterRaceCheckpoint(id) end
function onPlayerLeaveRaceCheckpoint(id) end

-- ------------------------------------------------------------
-- 10. OBJECT / PICKUP / ACTOR / TEXTDRAW / MENU EVENTS
-- ------------------------------------------------------------
function onPlayerSelectObject(id, objecttype, objectid, modelid, x, y, z)
    sa.SendClientMessage(id, 0xFFFFFFFF, "Pilih objek " .. objectid)
end

function onPlayerEditObject(id, playerObject, objectid, response, offx, offy, offz, rx, ry, rz)
    -- SA-MP: response 0=CANCEL 1=FINAL 2=UPDATE
end

function onPlayerCancelObjectEdit(id) end

function onPlayerPickUpPickup(id, pickupid)
    if pickupid == healthPickup then
        sa.SetPlayerHealth(id, 100.0)
    end
end

function onActorDamage(actorID, playerID, amount, weaponID, bodyPart) end

function onPlayerClickTextDraw(id, textDrawID) end

-- ------------------------------------------------------------
-- 11. CONTOH: dunia + timer + svar/pvar
-- ------------------------------------------------------------
local crate = sa.CreateObject(1225, SPAWN.x + 3.0, SPAWN.y, SPAWN.z, 0, 0, 0, 200.0)
local healthPickup = sa.CreatePickup(1240, 2, SPAWN.x + 2.0, SPAWN.y + 2.0, SPAWN.z)

local function demoVars()
    sa.SetSVarInt("round", 1)                       -- server-wide
    sa.SetPVarInt(0, "job", 4)                       -- per-player
    local r = sa.GetSVarInt("round")
    local j = sa.GetPVarInt(0, "job")
end

-- ------------------------------------------------------------
-- 12. BERSIH-BERSIH (opsional, saat gamemode reload)
-- ------------------------------------------------------------
function onGameModeExit()
    if timerID then sa.KillTimer(timerID) end
    if DB then sa.sqlite_close(DB) end
end