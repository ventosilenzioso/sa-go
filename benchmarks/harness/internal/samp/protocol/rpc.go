package protocol

// SA-MP 0.3.7 RPC identifiers, verified against RakSAMP SAMPRPC.cpp (client
// 0.3.7). One namespace shared by both directions: the same number identifies
// the same remote procedure whichever side sends it. The literal 255 entries
// in SAMPRPC.cpp are placeholders for functions unused by the vanilla client;
// they are omitted here.

// Client -> server RPCs.
const (
	RPCClickPlayer          = 23
	RPCClientJoin           = 25
	RPCEnterVehicle         = 26
	RPCRequestClass         = 128
	RPCRequestSpawn         = 129
	RPCSpawn                = 52
	RPCDeath                = 53
	RPCExitVehicle          = 154
	RPCChat                 = 101
	RPCMenuSelect           = 132
	RPCScmEvent             = 96
	RPCDialogResponse       = 62
	RPCGiveTakeDamage       = 115
	RPCEditObject           = 117
	RPCEnterEditObject      = 27
	RPCMapMarker            = 119
	RPCUpdateScoresPingsIPs = 155
	RPCVehicleDamage        = 106
	RPCToggleClock          = 30
	RPCServerCommand        = 50
	RPCPickedUpPickup       = 131
	RPCClickTextDraw        = 83
	RPCPlayerUpdate         = 60
	RPCMenuQuit             = 140
)

// Server -> client RPCs.
const (
	RPCServerJoin            = 137
	RPCServerQuit            = 138
	RPCInitGame              = 139
	RPCNPCJoin               = 54
	RPCConnectionRejected    = 130
	RPCGameModeRestart       = 40
	RPCClientMessage         = 93
	RPCWorldTime             = 94
	RPCPickup                = 95
	RPCDestroyPickup         = 63
	RPCDestroyWeaponPickup   = 97
	RPCVehicleDestroyed      = 136
	RPCWorldPlayerAdd        = 32
	RPCWorldPlayerDeath      = 166
	RPCWorldPlayerRemove     = 163
	RPCWorldVehicleAdd       = 164
	RPCWorldVehicleRemove    = 165
	RPCSetCheckpoint         = 107
	RPCDisableCheckpoint     = 37
	RPCSetRaceCheckpoint     = 38
	RPCDisableRaceCheckpoint = 39
	RPCSvrStats              = 102
	RPCWeather               = 152
	RPCPlayAudioStream       = 41
	RPCStopAudioStream       = 42
	RPCMapMarkerLegacy       = 119

	// Scr* (scripting RPCs, prefixed as in SAMPRPC.cpp).
	RPCScrSetSpawnInfo         = 68
	RPCScrSetPlayerTeam        = 69
	RPCScrSetPlayerSkin        = 153
	RPCScrSetPlayerName        = 11
	RPCScrSetPlayerPos         = 12
	RPCScrSetPlayerPosFindZ    = 13
	RPCScrSetPlayerHealth      = 14
	RPCScrPutPlayerInVehicle   = 70
	RPCScrRemoveFromVehicle    = 71
	RPCScrSetPlayerColor       = 72
	RPCScrDisplayGameText      = 73
	RPCScrSetInterior          = 156
	RPCScrSetCameraPos         = 157
	RPCScrSetCameraLookAt      = 158
	RPCScrInterpolateCamera    = 82
	RPCScrSetVehiclePos        = 159
	RPCScrSetVehicleZAngle     = 160
	RPCScrVehicleParams        = 161
	RPCScrCamBehindPlayer      = 162
	RPCScrToggleControllable   = 15
	RPCScrPlaySound            = 16
	RPCScrSetWorldBounds       = 17
	RPCScrHaveSomeMoney        = 18
	RPCScrSetFacingAngle       = 19
	RPCScrResetMoney           = 20
	RPCScrResetWeapons         = 21
	RPCScrGivePlayerWeapon     = 22
	RPCScrWeaponsUpdate        = 204
	RPCScrLinkVehicle          = 65
	RPCScrSetPlayerArmour      = 66
	RPCScrDeathMessage         = 55
	RPCScrSetMapIcon           = 56
	RPCScrDisableMapIcon       = 144
	RPCScrSetWeaponAmmo        = 145
	RPCScrSetGravity           = 146
	RPCScrSetVehicleHealth     = 147
	RPCScrSetVehicleTireStatus = 98
	RPCScrAttachTrailer        = 148
	RPCScrDetachTrailer        = 149
	RPCScrCreateObject         = 44
	RPCScrSetObjectPos         = 45
	RPCScrSetObjectRotation    = 46
	RPCScrDestroyObject        = 47
	RPCScrCreateExplosion      = 79
	RPCScrShowNameTag          = 80
	RPCScrStopFlashGangZone    = 85
	RPCScrApplyAnimation       = 86
	RPCScrClearAnimations      = 87
	RPCScrSetSpecialAction     = 88
	RPCScrFightingStyle        = 89
	RPCScrSetPlayerVelocity    = 90
	RPCScrSetVehicleVelocity   = 91
	RPCScrEnableStuntBonus     = 104
	RPCScrEditTextDraw         = 105
	RPCScrMoveObject           = 99
	RPCScrStopObject           = 122
	RPCScrNumberPlate          = 123
	RPCScrToggleSpectating     = 124
	RPCScrSpectatePlayer       = 126
	RPCScrSpectateVehicle      = 127
	RPCScrSetWantedLevel       = 133
	RPCScrShowTextDraw         = 134
	RPCScrHideTextDraw         = 135
	RPCScrRemoveComponent      = 57
	RPCScrForceSpawnSelection  = 74
	RPCScrAttachObjectToPlayer = 75
	RPCScrInitMenu             = 76
	RPCScrShowMenu             = 77
	RPCScrHideMenu             = 78
	RPCScrAddGangZone          = 108
	RPCScrRemoveGangZone       = 120
	RPCScrFlashGangZone        = 121
	RPCScrDialogBox            = 61
	RPCScrCreate3DTextLabel    = 36
	RPCScrDelete3DTextLabel    = 58
	RPCScrSetDrunkLevel        = 35

	// Actor RPCs (server -> client, plus one client -> server damage report).
	RPCScrShowActor              = 171
	RPCScrHideActor              = 172
	RPCScrApplyActorAnimation    = 173
	RPCScrClearActorAnimations   = 174
	RPCScrSetActorFacingAngle    = 175
	RPCScrSetActorPos            = 176
	RPCGiveActorDamage           = 177 // client -> server
	RPCScrSetActorHealth         = 178
)
