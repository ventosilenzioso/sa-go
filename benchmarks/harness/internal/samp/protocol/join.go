package protocol

import (
	"errors"
	"math"
	"math/big"

	"gosamp/internal/raknet"
)

// Client version numbers accepted by the server (open.mp
// RakNetLegacyNetwork::LegacyClientVersion).
const (
	ClientVersion037  = 4057
	ClientVersion03DL = 4062
)

// ConnectionRejected (RPC 130) reason codes (RakSAMP common.h).
const (
	RejectBadVersion  = 1
	RejectBadNickname = 2
	RejectBadMod      = 3
	RejectBadPlayerID = 4
)

// Player name rules (open.mp PlayerPool::isNameValid, SDK values.hpp).
const (
	MinPlayerName = 3
	MaxPlayerName = 24
)

// IsValidPlayerName reports whether name satisfies the SA-MP nickname rules:
// 3..24 characters from 0-9, a-z, A-Z and "][_$=()@.".
func IsValidPlayerName(name string) bool {
	if len(name) < MinPlayerName || len(name) > MaxPlayerName {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		ok := c >= '0' && c <= '9' ||
			c >= 'a' && c <= 'z' ||
			c >= 'A' && c <= 'Z' ||
			c == ']' || c == '[' || c == '_' || c == '$' || c == '=' ||
			c == '(' || c == ')' || c == '@' || c == '.'
		if !ok {
			return false
		}
	}
	return true
}

// IsValidSerial validates the GPCI auth string the way open.mp does: it must
// be a non-empty hex number below 50 characters whose value is divisible by
// 1001. Invalid serials get the peer kicked (and temporarily banned).
func IsValidSerial(key string) bool {
	if len(key) == 0 || len(key) >= 50 {
		return false
	}
	n, ok := new(big.Int).SetString(key, 16)
	if !ok {
		return false
	}
	return new(big.Int).Mod(n, big.NewInt(1001)).Sign() == 0
}

// ClientJoin is the parsed RPC_ClientJoin (25) payload.
type ClientJoin struct {
	Version           uint32
	Modded            uint8
	Name              string
	ChallengeResponse uint32
	Key               string // GPCI serial, hex
	VersionString     string
	Extra             *uint32 // open.mp clients append version/flags; absent on 0.3.7
}

// ParseClientJoin decodes a ClientJoin payload (already stripped of the RPC
// frame). It mirrors NetCode::RPC::PlayerConnect::read.
func ParseClientJoin(payload []byte) (ClientJoin, error) {
	var j ClientJoin
	bs := raknet.FromBytes(payload)
	var err error
	if j.Version, err = bs.ReadUint32(); err != nil {
		return j, err
	}
	if j.Modded, err = bs.ReadUint8(); err != nil {
		return j, err
	}
	if j.Name, err = readDynStr8(bs); err != nil {
		return j, err
	}
	if j.ChallengeResponse, err = bs.ReadUint32(); err != nil {
		return j, err
	}
	if j.Key, err = readDynStr8(bs); err != nil {
		return j, err
	}
	if j.VersionString, err = readDynStr8(bs); err != nil {
		return j, err
	}
	if bs.UnreadBits() >= 32 {
		v, err := bs.ReadUint32()
		if err != nil {
			return j, err
		}
		j.Extra = &v
	}
	return j, nil
}

// ValidateClientJoin checks version, challenge response, nickname and,
// when serialCheck is set, the GPCI serial. It returns a reject reason
// (0 = accept) and whether the peer must be kicked outright (bad serial /
// bad client version string, mirroring open.mp which kicks and temporarily
// bans instead of sending RPC 130).
func ValidateClientJoin(j ClientJoin, token uint32, serialCheck bool) (reject uint8, kick bool) {
	if j.Version != ClientVersion037 && j.Version != ClientVersion03DL {
		return RejectBadVersion, false
	}
	if j.ChallengeResponse^j.Version != token {
		return RejectBadVersion, false
	}
	if !IsValidPlayerName(j.Name) {
		return RejectBadNickname, false
	}
	if (serialCheck && !IsValidSerial(j.Key)) || len(j.VersionString) > 24 {
		return 0, true
	}
	return 0, false
}

// NPCJoin is the parsed RPC_NPCJoin (54) payload.
type NPCJoin struct {
	Version           uint32
	Modded            uint8
	Name              string
	ChallengeResponse uint32
}

// ParseNPCJoin decodes an NPCConnect payload (NetCode::RPC::NPCConnect::read).
func ParseNPCJoin(payload []byte) (NPCJoin, error) {
	var n NPCJoin
	bs := raknet.FromBytes(payload)
	var err error
	if n.Version, err = bs.ReadUint32(); err != nil {
		return n, err
	}
	if n.Modded, err = bs.ReadUint8(); err != nil {
		return n, err
	}
	if n.Name, err = readDynStr8(bs); err != nil {
		return n, err
	}
	if n.ChallengeResponse, err = bs.ReadUint32(); err != nil {
		return n, err
	}
	return n, nil
}

// SpawnInfo mirrors the packed PLAYER_SPAWN_INFO struct (46 bytes on the
// wire, #pragma pack(1) in RakSAMP common.h).
type SpawnInfo struct {
	Team    uint8
	Skin    int32
	Unknown uint8 // always 0 on the wire
	X, Y, Z float32
	Angle   float32
	Weapons [3]int32
	Ammo    [3]int32
}

// MarshalSpawnInfo encodes a SpawnInfo to its 46-byte wire form.
func MarshalSpawnInfo(s SpawnInfo) []byte {
	bs := raknet.New()
	bs.WriteUint8(s.Team)
	bs.WriteUint32(uint32(s.Skin))
	bs.WriteUint8(s.Unknown)
	bs.WriteUint32(math.Float32bits(s.X))
	bs.WriteUint32(math.Float32bits(s.Y))
	bs.WriteUint32(math.Float32bits(s.Z))
	bs.WriteUint32(math.Float32bits(s.Angle))
	for _, w := range s.Weapons {
		bs.WriteUint32(uint32(w))
	}
	for _, a := range s.Ammo {
		bs.WriteUint32(uint32(a))
	}
	return bs.Bytes()[:bs.Len()]
}

// ParseSpawnInfo decodes the 46-byte wire form.
func ParseSpawnInfo(b []byte) (SpawnInfo, error) {
	var s SpawnInfo
	if len(b) != 46 {
		return s, errors.New("protocol: spawn info must be 46 bytes")
	}
	bs := raknet.FromBytes(b)
	var err error
	if s.Team, err = bs.ReadUint8(); err != nil {
		return s, err
	}
	var v uint32
	if v, err = bs.ReadUint32(); err != nil {
		return s, err
	}
	s.Skin = int32(v)
	if s.Unknown, err = bs.ReadUint8(); err != nil {
		return s, err
	}
	reads := []*float32{&s.X, &s.Y, &s.Z, &s.Angle}
	for _, f := range reads {
		if v, err = bs.ReadUint32(); err != nil {
			return s, err
		}
		*f = math.Float32frombits(v)
	}
	for i := range s.Weapons {
		if v, err = bs.ReadUint32(); err != nil {
			return s, err
		}
		s.Weapons[i] = int32(v)
	}
	for i := range s.Ammo {
		if v, err = bs.ReadUint32(); err != nil {
			return s, err
		}
		s.Ammo[i] = int32(v)
	}
	return s, nil
}

// InitGameParams carries every field of the InitGame (139) payload. Field
// order follows the official 0.3.7 client parse (SAMP.Lua rpc_init_game_reader,
// cross-checked against open.mp NetCode::RPC::PlayerInit::write).
type InitGameParams struct {
	ZoneNames             bool
	UseCJWalk             bool
	AllowWeapons          bool
	LimitGlobalChatRadius bool
	GlobalChatRadius      float32
	StuntBonus            bool
	NametagDrawDistance   float32
	DisableEnterExits     bool
	NametagLOS            bool
	ManualEngineLights    bool
	ClassesAvailable      int32
	PlayerID              uint16
	ShowNametags          bool
	PlayerMarkersMode     int32
	WorldTime             uint8
	Weather               uint8
	Gravity               float32
	LanMode               bool
	DeathDropMoney        int32
	Instagib              bool
	OnfootRate            int32
	IncarRate             int32
	FiringRate            int32
	SendMultiplier        int32
	LagCompMode           int32
	Hostname              string
	VehicleModels         [212]byte
	VehicleFriendlyFire   bool
}

// BuildInitGame encodes the InitGame payload. The result is usually not a
// whole number of bytes (395 fixed bits before the hostname); the RPC frame
// carries the exact bit length so the client reads it back precisely.
func BuildInitGame(p InitGameParams) []byte {
	bs := raknet.New()
	bs.WriteBool(p.ZoneNames)
	bs.WriteBool(p.UseCJWalk)
	bs.WriteBool(p.AllowWeapons)
	bs.WriteBool(p.LimitGlobalChatRadius)
	bs.WriteUint32(math.Float32bits(p.GlobalChatRadius))
	bs.WriteBool(p.StuntBonus)
	bs.WriteUint32(math.Float32bits(p.NametagDrawDistance))
	bs.WriteBool(p.DisableEnterExits)
	bs.WriteBool(p.NametagLOS)
	bs.WriteBool(p.ManualEngineLights)
	bs.WriteUint32(uint32(p.ClassesAvailable))
	bs.WriteUint16(p.PlayerID)
	bs.WriteBool(p.ShowNametags)
	bs.WriteUint32(uint32(p.PlayerMarkersMode))
	bs.WriteUint8(p.WorldTime)
	bs.WriteUint8(p.Weather)
	bs.WriteUint32(math.Float32bits(p.Gravity))
	bs.WriteBool(p.LanMode)
	bs.WriteUint32(uint32(p.DeathDropMoney))
	bs.WriteBool(p.Instagib)
	bs.WriteUint32(uint32(p.OnfootRate))
	bs.WriteUint32(uint32(p.IncarRate))
	bs.WriteUint32(uint32(p.FiringRate))
	bs.WriteUint32(uint32(p.SendMultiplier))
	bs.WriteUint32(uint32(p.LagCompMode))
	writeDynStr8(bs, p.Hostname)
	// vehicleModels: 212 raw bytes at the current bit position (no align),
	// matching open.mp writeArray / SAMP.Lua's per-byte uint8 loop.
	bs.WriteBits(p.VehicleModels[:], len(p.VehicleModels)*8, true)
	if p.VehicleFriendlyFire {
		bs.WriteUint32(1)
	} else {
		bs.WriteUint32(0)
	}
	bs.AlignWrite()
	return bs.Bytes()[:bs.Len()]
}

// BuildRPCFrame wraps a payload in the legacy RPC frame: [ID_RPC][uint8 id]
// [compressed uint32 bit length][payload bits, right-aligned = false].
// No timestamp prefix is emitted (shiftTimestamp = false, as open.mp sends).
func BuildRPCFrame(rpcID byte, payload []byte) []byte {
	bs := raknet.New()
	bs.WriteUint8(raknet.IDRPC)
	bs.WriteUint8(rpcID)
	bs.WriteCompressedUint32(uint32(len(payload) * 8))
	if len(payload) > 0 {
		bs.WriteBits(payload, len(payload)*8, false)
	}
	bs.AlignWrite()
	return bs.Bytes()[:bs.Len()]
}

// ParseRPCFrame decodes an RPC frame into its id and payload. It mirrors
// RakNet's RPC read path: optional ID_TIMESTAMP prefix, raw uint8 id,
// compressed bit length, then payload bits with rightAligned = false.
func ParseRPCFrame(data []byte) (rpcID byte, payload []byte, err error) {
	bs := raknet.FromBytes(data)
	var b byte
	if b, err = bs.ReadUint8(); err != nil {
		return 0, nil, err
	}
	if b == raknet.IDTimestamp {
		// Skip the 4-byte timestamp.
		if _, err = bs.ReadUint32(); err != nil {
			return 0, nil, err
		}
		if b, err = bs.ReadUint8(); err != nil {
			return 0, nil, err
		}
	}
	if b != raknet.IDRPC {
		return 0, nil, errors.New("protocol: not an RPC frame")
	}
	if rpcID, err = bs.ReadUint8(); err != nil {
		return 0, nil, err
	}
	var nbits uint32
	if nbits, err = bs.ReadCompressedUint32(); err != nil {
		return 0, nil, err
	}
	nbytes := int((nbits + 7) / 8)
	if nbits == 0 {
		return rpcID, []byte{}, nil
	}
	payload = make([]byte, nbytes)
	if err = bs.ReadBits(payload, int(nbits), false); err != nil {
		return 0, nil, err
	}
	return rpcID, payload[:nbytes], nil
}

// BuildRequestClassResponse builds the server -> client RequestClass (128)
// payload: outcome byte followed by the 46-byte class spawn info (47 total).
func BuildRequestClassResponse(selectable bool, info SpawnInfo) []byte {
	out := make([]byte, 0, 47)
	if selectable {
		out = append(out, 1)
	} else {
		out = append(out, 0)
	}
	return append(out, MarshalSpawnInfo(info)...)
}

// BuildSetSpawnInfo builds the ScrSetSpawnInfo (68) payload (46 bytes).
func BuildSetSpawnInfo(info SpawnInfo) []byte {
	return MarshalSpawnInfo(info)
}

// ParseRequestClass extracts the requested class index. The official 0.3.7
// client sends an int32; open.mp reads a uint16. The low 16 bits agree for
// every valid class id, so reading two bytes accepts both shapes.
func ParseRequestClass(payload []byte) (int, error) {
	if len(payload) < 2 {
		return 0, errors.New("protocol: short RequestClass")
	}
	return int(uint16(payload[0]) | uint16(payload[1])<<8), nil
}

// BuildRequestSpawnResponse builds the server -> client RequestSpawn (129)
// payload: uint32 allow (1 = allow, 2 = spawn immediately).
func BuildRequestSpawnResponse(allow uint32) []byte {
	return []byte{byte(allow), byte(allow >> 8), byte(allow >> 16), byte(allow >> 24)}
}

// ParseChat decodes a client -> server Chat (101) payload (dynStr8).
func ParseChat(payload []byte) (string, error) {
	return readDynStr8(raknet.FromBytes(payload))
}

// ParseCommand decodes a client -> server ServerCommand (50) payload. The
// official 0.3.7 wire form is a 32-bit little-endian length followed by the
// command string (including the leading '/'). A few modded/mobile clients have
// been observed sending it as a dynStr8, so we fall back to that when the
// 32-bit length does not describe the remaining bytes.
func ParseCommand(payload []byte) (string, error) {
	if len(payload) >= 4 {
		n := int(payload[0]) | int(payload[1])<<8 | int(payload[2])<<16 | int(payload[3])<<24
		if n >= 0 && n <= len(payload)-4 {
			return string(payload[4 : 4+n]), nil
		}
	}
	return readDynStr8(raknet.FromBytes(payload))
}

// BuildServerCommand builds a client -> server ServerCommand (50) payload
// (uint32 length + string). Used by tests and probe tooling.
func BuildServerCommand(text string) []byte {
	out := make([]byte, 0, 4+len(text))
	n := uint32(len(text))
	out = append(out, byte(n), byte(n>>8), byte(n>>16), byte(n>>24))
	return append(out, text...)
}

// BuildChatBroadcast builds the server -> client Chat (101) payload:
// uint16 sender id followed by the text as dynStr8.
func BuildChatBroadcast(playerID uint16, text string) []byte {
	bs := raknet.New()
	bs.WriteUint16(playerID)
	writeDynStr8(bs, text)
	return bs.Bytes()[:bs.Len()]
}

// BuildClientMessage builds the ClientMessage (93) payload: uint32 RGBA color
// followed by the text as dynStr32.
func BuildClientMessage(color uint32, text string) []byte {
	bs := raknet.New()
	bs.WriteUint32(color)
	cp := UTF8ToCP1252(text)
	bs.WriteUint32(uint32(len(cp)))
	if len(cp) > 0 {
		bs.WriteBits(cp, len(cp)*8, true)
	}
	return bs.Bytes()[:bs.Len()]
}

// BuildWorldPlayerAdd builds the WorldPlayerAdd (32) payload announcing a
// streamed-in player.
func BuildWorldPlayerAdd(playerID uint16, team uint8, skin int32, x, y, z, angle float32, color uint32, fightingStyle uint8) []byte {
	bs := raknet.New()
	bs.WriteUint16(playerID)
	bs.WriteUint8(team)
	bs.WriteUint32(uint32(skin))
	bs.WriteUint32(math.Float32bits(x))
	bs.WriteUint32(math.Float32bits(y))
	bs.WriteUint32(math.Float32bits(z))
	bs.WriteUint32(math.Float32bits(angle))
	bs.WriteUint32(color)
	bs.WriteUint8(fightingStyle)
	return bs.Bytes()[:bs.Len()]
}

// BuildPlayerJoin builds the ServerJoin/PlayerJoin (137) payload announcing a
// newly connected player to the other clients.
func BuildPlayerJoin(playerID uint16, color uint32, isNPC bool, name string) []byte {
	bs := raknet.New()
	bs.WriteUint16(playerID)
	bs.WriteUint32(color)
	if isNPC {
		bs.WriteUint8(1)
	} else {
		bs.WriteUint8(0)
	}
	writeDynStr8(bs, name)
	return bs.Bytes()[:bs.Len()]
}

// BuildConnectionRejected builds the ConnectionRejected (130) payload.
func BuildConnectionRejected(reason uint8) []byte {
	return []byte{reason}
}

func readDynStr8(bs *raknet.BitStream) (string, error) {
	n, err := bs.ReadUint8()
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", nil
	}
	b := make([]byte, int(n))
	// RakNet reads the string bytes at the CURRENT bit position, without
	// byte-aligning. Using an aligned read would shift the cursor and make
	// the client read the hostname/strings from the wrong offset.
	if err := bs.ReadBits(b, int(n)*8, true); err != nil {
		return "", err
	}
	return CP1252ToUTF8(b), nil
}

func writeDynStr8(bs *raknet.BitStream, s string) {
	cp := UTF8ToCP1252(s)
	bs.WriteUint8(byte(len(cp)))
	if len(cp) > 0 {
		// Write the raw bytes at the current bit position (no byte-align),
		// matching RakNet BitStream::Write(ptr, len).
		bs.WriteBits(cp, len(cp)*8, true)
	}
}
