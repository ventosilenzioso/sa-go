-- Benchmark gamemode: permissive, no login gate, minimal logic.
-- Its only purpose is to allow bots to join -> class -> spawn so SA:GO is
-- measured with a gamemode loaded (comparable to open.mp running gungame).
-- It deliberately does NOT echo chat (chat relay stays in the core) and does
-- no anticheat, so it matches open.mp's gungame in having spawn allowed.

function onPlayerConnect(id) end
function onPlayerDisconnect(id, reason) end
function onPlayerRequestClass(id, classID) return true end
function onPlayerRequestSpawn(id) return true end
function onPlayerSpawn(id) end
function onPlayerText(id, text) return false end
