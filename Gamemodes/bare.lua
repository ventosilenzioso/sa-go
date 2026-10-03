-- SA:GO Official Starter Gamemode (bare.lua)
--
-- This starter gamemode provides a complete, production-ready foundation:
--   1. Account registration and authentication with SQLite and bcrypt.
--   2. Built-in SA-MP character selection with preview camera framing.
--   3. Spawn, death, and respawn handling.
--   4. Baseline anticheat speed guards with vehicle exit grace.
--   5. Standard administrative and gameplay commands.

-- ============================================================================
-- Configuration & Coordinates
-- ============================================================================

local SPAWN = { x = 1958.4, y = 1343.2, z = 15.4, angle = 270.0 }
local SELECT = { x = 1978.0, y = 1343.0, z = 15.0 }
local VISTA = { x = 1978.0, y = 1343.0, z = 35.0 }
local SKINS = { 0, 7, 21, 23, 29, 100, 250, 287 }

local SELECT_CAM = { x = SELECT.x, y = SELECT.y - 4.0, z = SELECT.z + 2.5 }
local SELECT_LOOK = { x = SELECT.x, y = SELECT.y, z = SELECT.z + 1.0 }

local DIALOG_LOGIN = 200
local DIALOG_REGISTER = 201
local DIALOG_REGISTER_CONFIRM = 202

local MAX_LOGIN_FAILS = 5
local LOGIN_RESET_AFTER = 180

-- Anticheat parameters
local MAX_RUN_SPEED = 60.0
local SPEED_SLACK = 3.0
local TELEPORT_JUMP = 300.0
local STRIKE_WINDOW = 5.0
local MAX_STRIKES = 5

-- ============================================================================
-- State & Database
-- ============================================================================

local loggedIn = {}
local accountName = {}
local chosenSkin = {}
local loginFails = {}
local pendingPassword = {}
local strikes = {}
local lastGoodPos = {}
local exitGrace = {}

local DB = sa.sqlite_open("scriptfiles/accounts.db")
if DB then
    sa.sqlite_exec(DB, [[CREATE TABLE IF NOT EXISTS accounts (
        name TEXT PRIMARY KEY,
        password_hash TEXT NOT NULL,
        skin INTEGER NOT NULL DEFAULT 0,
        created_at INTEGER NOT NULL
    )]])
else
    sa.Log("SA:GO: accounts database could not be opened")
end

local function resetBaseline(id)
    strikes[id] = nil
    lastGoodPos[id] = nil
    exitGrace[id] = nil
end

local function recordStrike(id, now)
    local s = strikes[id]
    if not s then
        s = { times = {} }
        strikes[id] = s
    end
    table.insert(s.times, now)
    local kept = {}
    for _, t in ipairs(s.times) do
        if now - t <= STRIKE_WINDOW then
            table.insert(kept, t)
        end
    end
    s.times = kept
    return #kept
end

-- ============================================================================
-- Authentication Flow
-- ============================================================================

local function showRegister(id)
    sa.ShowPlayerDialog(id, DIALOG_REGISTER, 3, "SA:GO Register",
        "Create password (minimum 4 characters):", "OK", "Cancel")
end

local function showLogin(id)
    sa.ShowPlayerDialog(id, DIALOG_LOGIN, 3, "SA:GO Login",
        "Enter your password:", "Login", "Cancel")
end

local function beginSelection(id)
    sa.SendClientMessage(id, 0x00FF00FF, "Login successful. Choose character with next/prev, then spawn.")
    sa.SetPlayerPos(id, SELECT.x, SELECT.y, SELECT.z)
    sa.SetPlayerFacingAngle(id, 0.0)
    sa.SetPlayerCameraPos(id, SELECT_CAM.x, SELECT_CAM.y, SELECT_CAM.z)
    sa.SetPlayerCameraLookAt(id, SELECT_LOOK.x, SELECT_LOOK.y, SELECT_LOOK.z, 2)
    sa.ForceClassSelection(id)
end

local function authOK(id, name)
    loggedIn[id] = true
    accountName[id] = name
    loginFails[id] = nil
    pendingPassword[id] = nil
    beginSelection(id)
end

local function loginFail(id, name)
    local f = loginFails[id]
    if not f or os.time() - f.first >= LOGIN_RESET_AFTER then
        f = { count = 0, first = os.time() }
        loginFails[id] = f
    end
    f.count = f.count + 1
    if f.count >= MAX_LOGIN_FAILS then
        sa.KickPlayer(id, "login brute force")
    else
        sa.SendClientMessage(id, 0xFF4444FF,
            string.format("Wrong password (%d/%d).", f.count, MAX_LOGIN_FAILS))
        showLogin(id)
    end
end

function onPlayerConnect(id)
    loggedIn[id] = false
    chosenSkin[id] = 1
    pendingPassword[id] = nil
    resetBaseline(id)

    -- Static vista camera while authentication dialog is active
    sa.SetPlayerCameraPos(id, VISTA.x, VISTA.y, VISTA.z)
    sa.SetPlayerCameraLookAt(id, SELECT.x, SELECT.y, SELECT.z, 1)

    local name, ok = sa.PlayerName(id)
    if not ok then return end

    sa.SendClientMessage(id, 0xFFFFFFFF, "Welcome to SA:GO, " .. name .. "!")
    if not DB then
        sa.KickPlayer(id, "database unavailable")
        return
    end

    local rows = sa.sqlite_query(DB, "SELECT name FROM accounts WHERE name = ?", name)
    if rows and rows[1] then
        showLogin(id)
    else
        showRegister(id)
    end
end

function onPlayerDisconnect(id, reason)
    loggedIn[id] = nil
    accountName[id] = nil
    chosenSkin[id] = nil
    pendingPassword[id] = nil
    loginFails[id] = nil
    resetBaseline(id)
end

function onDialogResponse(id, dialogID, response, listitem, inputtext)
    if dialogID == DIALOG_REGISTER then
        if response ~= 1 then
            sa.KickPlayer(id, "registration cancelled")
            return
        end
        if #inputtext < 4 then
            sa.SendClientMessage(id, 0xFF4444FF, "Password too short.")
            showRegister(id)
            return
        end
        pendingPassword[id] = inputtext
        sa.ShowPlayerDialog(id, DIALOG_REGISTER_CONFIRM, 3, "Confirm password",
            "Type password again:", "OK", "Cancel")
        return
    end

    if dialogID == DIALOG_REGISTER_CONFIRM then
        if response ~= 1 then
            sa.KickPlayer(id, "registration cancelled")
            return
        end
        if inputtext ~= pendingPassword[id] then
            sa.SendClientMessage(id, 0xFF4444FF, "Passwords do not match.")
            showRegister(id)
            return
        end
        local name = sa.PlayerName(id)
        local hash, err = sa.hash_password(inputtext)
        if not hash then
            sa.KickPlayer(id, "password hashing failed")
            return
        end
        local _, _, dbErr = sa.sqlite_exec(DB,
            "INSERT INTO accounts(name,password_hash,skin,created_at) VALUES(?,?,?,?)",
            name, hash, SKINS[1], os.time())
        if dbErr then
            sa.KickPlayer(id, "account creation failed")
            return
        end
        authOK(id, name)
        return
    end

    if dialogID == DIALOG_LOGIN then
        if response ~= 1 then
            sa.KickPlayer(id, "login cancelled")
            return
        end
        local name = sa.PlayerName(id)
        local rows = sa.sqlite_query(DB, "SELECT password_hash, skin FROM accounts WHERE name = ?", name)
        local hash = rows and rows[1] and rows[1].password_hash
        if hash and sa.verify_password(inputtext, hash) then
            chosenSkin[id] = (rows[1].skin or SKINS[1])
            authOK(id, name)
        else
            loginFail(id, name)
        end
        return
    end
end

-- ============================================================================
-- Character Selection Flow
-- ============================================================================

-- Gate character selection behind successful authentication.
function onPlayerRequestClass(id, classID)
    if not loggedIn[id] then return false end
    chosenSkin[id] = SKINS[classID + 1] or SKINS[1]
    return true
end

-- Reposition character model and frame camera when client cycles classes.
function onPlayerRequestClassDone(id, classID, skin)
    if not loggedIn[id] then return end
    sa.SetPlayerPos(id, SELECT.x, SELECT.y, SELECT.z)
    sa.SetPlayerSkin(id, skin)
    sa.SetPlayerFacingAngle(id, 0.0)
    sa.SetPlayerCameraPos(id, SELECT_CAM.x, SELECT_CAM.y, SELECT_CAM.z)
    sa.SetPlayerCameraLookAt(id, SELECT_LOOK.x, SELECT_LOOK.y, SELECT_LOOK.z, 2)
end

-- ============================================================================
-- Spawn & Death Flow
-- ============================================================================

function onPlayerRequestSpawn(id)
    if not loggedIn[id] then return false end
    sa.SetSpawnInfo(id, 0, chosenSkin[id] or SKINS[1], SPAWN.x, SPAWN.y, SPAWN.z, SPAWN.angle)
    return true
end

function onPlayerSpawn(id)
    sa.SetPlayerCameraBehindPlayer(id)
    resetBaseline(id)

    -- Top-right branding textdraw
    sa.TextDrawCreate(1, 530.0, 8.0, "SA:GO")
    sa.TextDrawLetterSize(1, 0.4, 1.6)
    sa.TextDrawColor(1, 0xFFFFFFFF)
    sa.TextDrawShowForPlayer(id, 1)

    -- Top-left help prompt textdraw
    sa.TextDrawCreate(2, 8.0, 8.0, "Type /help for commands")
    sa.TextDrawLetterSize(2, 0.28, 1.1)
    sa.TextDrawColor(2, 0x00FF00FF)
    sa.TextDrawShowForPlayer(id, 2)

    -- Spawn map icon
    sa.SetPlayerMapIcon(id, 10, SPAWN.x, SPAWN.y, SPAWN.z, 0, 0x00FF00FF, 1)
end

function onPlayerDeath(id, killerID, reason)
    resetBaseline(id)
    sa.SendClientMessage(id, 0xFF4444FF, "You died. You will respawn shortly.")
end

-- ============================================================================
-- Movement & Anticheat Guards
-- ============================================================================

function onPlayerUpdate(id, data)
    if data.teleported or data.inVehicle then
        lastGoodPos[id] = data.newPos
        return
    end

    -- Skip velocity checks during the short vehicle-exit transition window
    if exitGrace[id] and exitGrace[id] > 0 then
        exitGrace[id] = exitGrace[id] - 1
        lastGoodPos[id] = data.newPos
        return
    end

    if data.deltaTime < 0.03 or data.deltaTime > 1.0 then
        lastGoodPos[id] = data.newPos
        return
    end

    local old = lastGoodPos[id]
    if not old then
        lastGoodPos[id] = data.newPos
        return
    end

    local dx, dy = data.newPos.x - old.x, data.newPos.y - old.y
    local dist = math.sqrt(dx * dx + dy * dy)
    if dist ~= dist or dist == math.huge then
        lastGoodPos[id] = data.newPos
        return
    end

    if dist > TELEPORT_JUMP or dist / data.deltaTime > MAX_RUN_SPEED * SPEED_SLACK then
        local count = recordStrike(id, os.clock())
        sa.Log(string.format("anticheat: player %d dist=%.1f strikes=%d/%d", id, dist, count, MAX_STRIKES))
        sa.SetPlayerPos(id, old.x, old.y, old.z)
        if count >= MAX_STRIKES then
            sa.SendClientMessage(id, 0xFF0000FF, "Kicked: speed hack detected.")
            sa.KickPlayer(id, "speed hack")
            resetBaseline(id)
        end
        return
    end

    lastGoodPos[id] = data.newPos
end

function onPlayerEnterVehicle(id, vehicleID, passenger)
    resetBaseline(id)
end

function onPlayerExitVehicle(id, vehicleID)
    resetBaseline(id)
    exitGrace[id] = 5
end

function onPlayerText(id, text)
    return false
end

-- ============================================================================
-- Commands
-- ============================================================================

function onPlayerCommand(id, command, params)
    -- --- General / Help Commands ---
    if command == "help" or command == "helpers" then
        sa.SendClientMessage(id, 0xFFFFFFFF, "Commands: /veh /giveweapon /heal /setadmin /kick /ban /say")
        return true
    end

    if not loggedIn[id] then
        return false
    end

    -- --- Gameplay Commands ---
    if command == "veh" then
        local model, c1, c2 = params:match("^(%d+)%s+(%d+)%s+(%d+)$")
        if not model then
            sa.SendClientMessage(id, 0xFF4444FF, "Usage: /veh <model> <color1> <color2>")
            return true
        end
        local x, y, z = sa.GetPlayerPos(id)
        local vid = sa.CreateVehicle(tonumber(model), x, y, z + 1, 0, tonumber(c1), tonumber(c2))
        if vid then
            sa.SendClientMessage(id, 0x00FF00FF, "Vehicle spawned.")
        end
        return true
    end

    if command == "giveweapon" then
        local wid, ammo = params:match("^(%d+)%s+(%d+)$")
        if wid and sa.GivePlayerWeapon(id, tonumber(wid), tonumber(ammo)) then
            sa.SendClientMessage(id, 0x00FF00FF, "Weapon given.")
        else
            sa.SendClientMessage(id, 0xFF4444FF, "Usage: /giveweapon <id> <ammo>")
        end
        return true
    end

    if command == "heal" then
        sa.SetPlayerHealth(id, 100)
        sa.SetPlayerArmour(id, 100)
        sa.SendClientMessage(id, 0x00FF00FF, "Healed.")
        return true
    end

    if command == "setadmin" then
        sa.SetPlayerAdmin(id, true)
        sa.SendClientMessage(id, 0x00FF00FF, "Admin enabled.")
        return true
    end

    -- --- Administrative Commands (requires admin privileges) ---
    if not sa.IsPlayerAdmin(id) then
        return false
    end

    if command == "kick" then
        local target, reason = params:match("^(%d+)%s*(.*)$")
        if target then
            sa.KickPlayer(tonumber(target), reason ~= "" and reason or "kicked by admin")
        end
        return true
    end

    if command == "ban" then
        local target, reason = params:match("^(%d+)%s*(.*)$")
        if target then
            sa.BanPlayer(tonumber(target), reason ~= "" and reason or "banned by admin")
        end
        return true
    end

    if command == "say" then
        if params ~= "" then
            sa.SendClientMessageToAll(0xFFFF00FF, "* Admin: " .. params)
        end
        return true
    end

    return false
end
