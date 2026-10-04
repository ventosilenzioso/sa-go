package protocol

import (
	"errors"
	"fmt"
	"math"

	"gosamp/internal/raknet"
)

// SA-MP player/vehicle synchronization packet codecs, ported from the wire
// format used by RakSAMP (server/src/netgame.cpp Packet_PlayerSync /
// Packet_VehicleSync) and cross-checked against open.mp NetCode structures.
//
// On-foot sync (ID 207) server -> clients broadcast layout:
//
//	[ID=207][playerId=u16][hasLR][lr?=u16][hasUD][ud?=u16]
//	[keys=u16][pos=3xf32][quat=4xf32][healthArmor=u8][weapon=u8]
//	[specialAction=u8][velocity][hasSurf][surf?][hasAnim][anim?]
//
// Vehicle sync (ID 200) server -> clients broadcast layout:
//
//	[ID=200][playerId=u16][vehicleId=u16][lr=u16][ud=u16][keys=u16]
//	[quat=4xf32][pos=3xf32][velocity][fCarHealth=f32][healthArmor=u8]
//	[weapon=u8][siren=u8][landingGear=u8][hasTrailer][trailer?=u16]

// Sync packet IDs.
const (
	PacketIDVehicleSync   = 200
	PacketIDAimSync       = 203
	PacketIDWeaponsUpdate = 204
	PacketIDStatsUpdate   = 205
	PacketIDBulletSync    = 206
	PacketIDOnFootSync    = 207
	PacketIDMarkersSync   = 208
	PacketIDUnoccupied    = 209
	PacketIDTrailerSync   = 210
	PacketIDPassengerSync = 211
	PacketIDSpectating    = 212
)

// FullKeyMask is the bits a client's Keys uint16 cannot hold; the 2-bit
// additional_key selector carries C-bug/other high keys.
const (
	// KeyCrouchBit is set when additional_key selects the C-bug key. Raw
	// additional_key == 3 maps to 0x40000 (NOT 0x30000 from a plain shift).
	additionalKeyShift     = 16
	additionalKeyCbug      = 3
	additionalKeyCbugValue = 0x40000
)

// FullKeys reconstructs the 32-bit key state from the 16-bit Keys and the
// 2-bit additional_key, per the SA-MP reference:
//
//	full_keys = keys | (additional_key == 3 ? 0x40000 : additional_key << 16)
//
// A plain shift of 3 would wrongly yield 0x30000; the raw value 3 is special.
func FullKeys(keys uint16, additionalKey uint8) uint32 {
	fk := uint32(keys)
	if additionalKey == additionalKeyCbug {
		return fk | additionalKeyCbugValue
	}
	return fk | uint32(additionalKey)<<additionalKeyShift
}

// SplitWeaponAdditional splits the packed weapon byte (WeaponID:6,
// AdditionalKey:2) used by on-foot and driver sync.
func SplitWeaponAdditional(b byte) (weapon, additionalKey uint8) {
	return b & 0x3F, (b >> 6) & 0x03
}

// PackWeaponAdditional packs WeaponID (6 bits) and AdditionalKey (2 bits).
func PackWeaponAdditional(weapon, additionalKey uint8) byte {
	return (weapon & 0x3F) | (additionalKey&0x03)<<6
}

// PlayerSyncData is the decoded on-foot sync payload from a client.
type PlayerSyncData struct {
	LeftRight                    uint16
	UpDown                       uint16
	Keys                         uint16
	X, Y, Z                      float32
	QuatW, QuatX, QuatY, QuatZ   float32
	Health                       uint8
	Armor                        uint8
	Weapon                       uint8
	AdditionalKey                uint8
	SpecialAction                uint8
	VelX, VelY, VelZ             float32
	HasSurf                      bool
	SurfVehicle                  uint16
	SurfOffX, SurfOffY, SurfOffZ float32
	HasAnim                      bool
	AnimID                       uint16
	AnimFlags                    uint16
}

// FullKeys reconstructs the full 32-bit key state from Keys and AdditionalKey.
func (d PlayerSyncData) FullKeys() uint32 { return FullKeys(d.Keys, d.AdditionalKey) }

// VehicleSyncData is the decoded vehicle (driver) sync payload from a client.
type VehicleSyncData struct {
	VehicleID                  uint16
	LeftRight                  uint16
	UpDown                     uint16
	Keys                       uint16
	QuatW, QuatX, QuatY, QuatZ float32
	X, Y, Z                    float32
	VelX, VelY, VelZ           float32
	VehicleHealth              float32
	Health, Armor              uint8
	Weapon                     uint8
	AdditionalKey              uint8
	Siren                      uint8
	LandingGear                uint8
	TrailerID                  uint16
	HydraThrustAngle           uint32
}

// PassengerSyncData is the decoded passenger sync payload.
type PassengerSyncData struct {
	VehicleID     uint16
	SeatID        uint8
	DriveBy       bool
	Cuffed        bool
	Weapon        uint8
	AdditionalKey uint8
	Health, Armor uint8
	LeftRight     uint16
	UpDown        uint16
	Keys          uint16
	X, Y, Z       float32
}

// ParseOnFootSync decodes a client -> server on-foot sync payload (already
// stripped of the leading ID byte). It returns the player's state.
func ParseOnFootSync(payload []byte) (PlayerSyncData, error) {
	var d PlayerSyncData
	bs := raknet.FromBytes(payload)
	var err error

	readU16 := func(dst *uint16) {
		if err != nil {
			return
		}
		v, e := bs.ReadUint16()
		if e != nil {
			err = e
			return
		}
		*dst = v
	}
	readF32 := func(dst *float32) {
		if err != nil {
			return
		}
		v, e := bs.ReadUint32()
		if e != nil {
			err = e
			return
		}
		*dst = math.Float32frombits(v)
	}

	readU16(&d.LeftRight)
	readU16(&d.UpDown)
	readU16(&d.Keys)
	readF32(&d.X)
	readF32(&d.Y)
	readF32(&d.Z)
	if err != nil {
		return d, err
	}
	d.QuatW, d.QuatX, d.QuatY, d.QuatZ, err = readNormQuat(bs)
	if err != nil {
		return d, err
	}
	// Client -> server sends health and armour as raw UINT8 values (open.mp
	// readCompressedPercentPair reads two bytes; the CEILDIV nibble packing is
	// only used on the server -> client broadcast).
	if d.Health, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if d.Armor, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	// Packed weapon byte: WeaponID (6 bits) + AdditionalKey (2 bits).
	var wa byte
	if wa, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	d.Weapon, d.AdditionalKey = SplitWeaponAdditional(wa)
	if d.SpecialAction, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if err = readCompressedVector(bs, &d.VelX, &d.VelY, &d.VelZ); err != nil {
		return d, err
	}
	// Surfing data: offset VEC3 + uint16 id, always present on the wire.
	if err = readVEC3(bs, &d.SurfOffX, &d.SurfOffY, &d.SurfOffZ); err != nil {
		return d, err
	}
	if d.SurfVehicle, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	d.HasSurf = d.SurfVehicle > 0
	if d.AnimID, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	if d.AnimFlags, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	d.HasAnim = d.AnimID > 0
	return d, nil
}

// BuildOnFootSync builds a client -> server on-foot sync payload (including
// the leading ID byte), the inverse of ParseOnFootSync. It matches open.mp
// PlayerFootSync::read: raw position, raw rotation, raw health+armour bytes,
// packed weapon/additional-key byte, special action, compressed velocity,
// surfing offset+id, and animation id+flags.
func BuildOnFootSync(d PlayerSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDOnFootSync)
	bs.WriteUint16(d.LeftRight)
	bs.WriteUint16(d.UpDown)
	bs.WriteUint16(d.Keys)
	writeVEC3(bs, d.X, d.Y, d.Z)
	writeNormQuat(bs, d.QuatW, d.QuatX, d.QuatY, d.QuatZ)
	bs.WriteUint8(d.Health)
	bs.WriteUint8(d.Armor)
	bs.WriteUint8(PackWeaponAdditional(d.Weapon, d.AdditionalKey))
	bs.WriteUint8(d.SpecialAction)
	writeCompressedVector(bs, d.VelX, d.VelY, d.VelZ)
	writeVEC3(bs, d.SurfOffX, d.SurfOffY, d.SurfOffZ)
	bs.WriteUint16(d.SurfVehicle)
	bs.WriteUint16(d.AnimID)
	bs.WriteUint16(d.AnimFlags)
	return bs.Bytes()[:bs.Len()]
}

// BuildOnFootSyncBroadcast rebuilds the server -> client on-foot sync packet
// in the RakSAMP broadcast format (player id + conditional optional fields).
func BuildOnFootSyncBroadcast(playerID uint16, d PlayerSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDOnFootSync)
	bs.WriteUint16(playerID)
	writeOptionalU16(bs, d.LeftRight)
	writeOptionalU16(bs, d.UpDown)
	bs.WriteUint16(d.Keys)
	writeVEC3(bs, d.X, d.Y, d.Z)
	writeNormQuat(bs, d.QuatW, d.QuatX, d.QuatY, d.QuatZ)
	bs.WriteUint8(encodeHealthArmor(d.Health, d.Armor))
	bs.WriteUint8(PackWeaponAdditional(d.Weapon, d.AdditionalKey))
	bs.WriteUint8(d.SpecialAction)
	writeCompressedVector(bs, d.VelX, d.VelY, d.VelZ)
	if d.HasSurf {
		bs.WriteBool(true)
		bs.WriteUint16(d.SurfVehicle)
		bs.WriteUint32(math.Float32bits(d.SurfOffX))
		bs.WriteUint32(math.Float32bits(d.SurfOffY))
		bs.WriteUint32(math.Float32bits(d.SurfOffZ))
	} else {
		bs.WriteBool(false)
	}
	if d.HasAnim {
		bs.WriteBool(true)
		bs.WriteUint16(d.AnimID)
		bs.WriteUint16(d.AnimFlags)
	} else {
		bs.WriteBool(false)
	}
	return bs.Bytes()[:bs.Len()]
}

// ParseVehicleSync decodes a client -> server vehicle sync payload (without
// the leading ID byte).
func ParseVehicleSync(payload []byte) (VehicleSyncData, error) {
	var d VehicleSyncData
	bs := raknet.FromBytes(payload)
	var err error

	readU16 := func(dst *uint16) {
		if err != nil {
			return
		}
		v, e := bs.ReadUint16()
		if e != nil {
			err = e
			return
		}
		*dst = v
	}
	readF32 := func(dst *float32) {
		if err != nil {
			return
		}
		v, e := bs.ReadUint32()
		if e != nil {
			err = e
			return
		}
		*dst = math.Float32frombits(v)
	}

	readU16(&d.VehicleID)
	readU16(&d.LeftRight)
	readU16(&d.UpDown)
	readU16(&d.Keys)
	if err != nil {
		return d, err
	}
	d.QuatW, d.QuatX, d.QuatY, d.QuatZ, err = readNormQuat(bs)
	if err != nil {
		return d, err
	}
	readF32(&d.X)
	readF32(&d.Y)
	readF32(&d.Z)
	if err != nil {
		return d, err
	}
	if err = readCompressedVector(bs, &d.VelX, &d.VelY, &d.VelZ); err != nil {
		return d, err
	}
	var vh uint32
	if vh, err = bs.ReadUint32(); err != nil {
		return d, err
	}
	d.VehicleHealth = math.Float32frombits(vh)
	// Client sends raw health, armour, packed weapon/additional-key, siren,
	// landing gear, trailer id, then hydra thrust angle (open.mp read).
	if d.Health, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if d.Armor, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	var wa byte
	if wa, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	d.Weapon, d.AdditionalKey = SplitWeaponAdditional(wa)
	if d.Siren, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if d.LandingGear, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if d.TrailerID, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	if d.HydraThrustAngle, err = bs.ReadUint32(); err != nil {
		return d, err
	}
	return d, nil
}

// BuildVehicleSync builds a client -> server driver sync (ID 200) payload with
// the leading ID byte (inverse of ParseVehicleSync).
func BuildVehicleSync(d VehicleSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDVehicleSync)
	bs.WriteUint16(d.VehicleID)
	bs.WriteUint16(d.LeftRight)
	bs.WriteUint16(d.UpDown)
	bs.WriteUint16(d.Keys)
	writeNormQuat(bs, d.QuatW, d.QuatX, d.QuatY, d.QuatZ)
	writeVEC3(bs, d.X, d.Y, d.Z)
	writeCompressedVector(bs, d.VelX, d.VelY, d.VelZ)
	bs.WriteUint32(math.Float32bits(d.VehicleHealth))
	bs.WriteUint8(d.Health)
	bs.WriteUint8(d.Armor)
	bs.WriteUint8(PackWeaponAdditional(d.Weapon, d.AdditionalKey))
	bs.WriteUint8(d.Siren)
	bs.WriteUint8(d.LandingGear)
	bs.WriteUint16(d.TrailerID)
	bs.WriteUint32(d.HydraThrustAngle)
	return bs.Bytes()[:bs.Len()]
}

// BuildVehicleSyncBroadcast rebuilds the server -> client vehicle sync packet.
func BuildVehicleSyncBroadcast(playerID uint16, d VehicleSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDVehicleSync)
	bs.WriteUint16(playerID)
	bs.WriteUint16(d.VehicleID)
	bs.WriteUint16(d.LeftRight)
	bs.WriteUint16(d.UpDown)
	bs.WriteUint16(d.Keys)
	writeNormQuat(bs, d.QuatW, d.QuatX, d.QuatY, d.QuatZ)
	writeVEC3(bs, d.X, d.Y, d.Z)
	writeCompressedVector(bs, d.VelX, d.VelY, d.VelZ)
	bs.WriteUint16(uint16(d.VehicleHealth))
	bs.WriteUint8(encodeHealthArmor(d.Health, d.Armor))
	bs.WriteUint8(PackWeaponAdditional(d.Weapon, d.AdditionalKey))
	bs.WriteBool(d.Siren != 0)
	bs.WriteBool(d.LandingGear != 0)
	bs.WriteBool(d.HydraThrustAngle > 0)
	if d.HydraThrustAngle > 0 {
		bs.WriteUint32(d.HydraThrustAngle)
	}
	hasTrailer := d.TrailerID > 0
	bs.WriteBool(hasTrailer)
	if hasTrailer {
		bs.WriteUint16(d.TrailerID)
	}
	return bs.Bytes()[:bs.Len()]
}

// ParseVehicleSyncBroadcast decodes the server -> client vehicle sync (ID 200)
// packet (including the packet id and player id), the inverse of
// BuildVehicleSyncBroadcast. Mirrors open.mp Packet::PlayerVehicleSync::write.
func ParseVehicleSyncBroadcast(payload []byte) (uint16, VehicleSyncData, error) {
	var d VehicleSyncData
	bs := raknet.FromBytes(payload)
	id, err := bs.ReadUint8()
	if err != nil {
		return 0, d, err
	}
	if id != PacketIDVehicleSync {
		return 0, d, fmt.Errorf("protocol: not a vehicle sync: id %d", id)
	}
	playerID, err := bs.ReadUint16()
	if err != nil {
		return 0, d, err
	}
	if d.VehicleID, err = bs.ReadUint16(); err != nil {
		return 0, d, err
	}
	if d.LeftRight, err = bs.ReadUint16(); err != nil {
		return 0, d, err
	}
	if d.UpDown, err = bs.ReadUint16(); err != nil {
		return 0, d, err
	}
	if d.Keys, err = bs.ReadUint16(); err != nil {
		return 0, d, err
	}
	if d.QuatW, d.QuatX, d.QuatY, d.QuatZ, err = readNormQuat(bs); err != nil {
		return 0, d, err
	}
	if err = readVEC3(bs, &d.X, &d.Y, &d.Z); err != nil {
		return 0, d, err
	}
	if err = readCompressedVector(bs, &d.VelX, &d.VelY, &d.VelZ); err != nil {
		return 0, d, err
	}
	var vh uint16
	if vh, err = bs.ReadUint16(); err != nil {
		return 0, d, err
	}
	d.VehicleHealth = float32(vh)
	ha, err := bs.ReadUint8()
	if err != nil {
		return 0, d, err
	}
	d.Health, d.Armor = decodeHealthArmor(ha)
	wa, err := bs.ReadUint8()
	if err != nil {
		return 0, d, err
	}
	d.Weapon, d.AdditionalKey = SplitWeaponAdditional(wa)
	siren, err := bs.ReadBool()
	if err != nil {
		return 0, d, err
	}
	if siren {
		d.Siren = 1
	}
	gear, err := bs.ReadBool()
	if err != nil {
		return 0, d, err
	}
	if gear {
		d.LandingGear = 1
	}
	hasHydra, err := bs.ReadBool()
	if err != nil {
		return 0, d, err
	}
	if hasHydra {
		if d.HydraThrustAngle, err = bs.ReadUint32(); err != nil {
			return 0, d, err
		}
	}
	hasTrailer, err := bs.ReadBool()
	if err != nil {
		return 0, d, err
	}
	if hasTrailer {
		if d.TrailerID, err = bs.ReadUint16(); err != nil {
			return 0, d, err
		}
	}
	return playerID, d, nil
}

// ParsePassengerSync decodes a client -> server passenger sync payload. The
// control word packs (low to high): SeatID:6, DriveBy:1, Cuffed:1,
// WeaponID:6, AdditionalKey:2. Health/armour are raw bytes.
func ParsePassengerSync(payload []byte) (PassengerSyncData, error) {
	var d PassengerSyncData
	bs := raknet.FromBytes(payload)
	var err error
	if d.VehicleID, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	var ctrl uint16
	if ctrl, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	d.SeatID = uint8(ctrl & 0x3F)
	d.DriveBy = (ctrl>>6)&1 == 1
	d.Cuffed = (ctrl>>7)&1 == 1
	d.Weapon = uint8((ctrl >> 8) & 0x3F)
	d.AdditionalKey = uint8((ctrl >> 14) & 0x03)
	if d.Health, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if d.Armor, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if d.LeftRight, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	if d.UpDown, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	if d.Keys, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	if err = readVEC3(bs, &d.X, &d.Y, &d.Z); err != nil {
		return d, err
	}
	return d, nil
}

// BuildPassengerSync builds a client -> server passenger sync payload with the
// leading ID byte (inverse of ParsePassengerSync).
func BuildPassengerSync(d PassengerSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDPassengerSync)
	bs.WriteUint16(d.VehicleID)
	ctrl := uint16(d.SeatID & 0x3F)
	if d.DriveBy {
		ctrl |= 1 << 6
	}
	if d.Cuffed {
		ctrl |= 1 << 7
	}
	ctrl |= uint16(d.Weapon&0x3F) << 8
	ctrl |= uint16(d.AdditionalKey&0x03) << 14
	bs.WriteUint16(ctrl)
	bs.WriteUint8(d.Health)
	bs.WriteUint8(d.Armor)
	bs.WriteUint16(d.LeftRight)
	bs.WriteUint16(d.UpDown)
	bs.WriteUint16(d.Keys)
	writeVEC3(bs, d.X, d.Y, d.Z)
	return bs.Bytes()[:bs.Len()]
}

// --- wire helpers ---

// CompressHealthArmor packs health (high nibble) and armor (low nibble) into
// the on-foot/vehicle sync byte. Formula: >=100 -> 0xF; else floor(v/7) for
// v>0, 0 for v==0. Verified with cmd/gosamp-interopprobe against gosamp-server.
func CompressHealthArmor(health, armor uint8) byte { return encodeHealthArmor(health, armor) }

// DecompressHealthArmor unpacks the health/armor byte (nibble * 7, clamped to
// 100).
func DecompressHealthArmor(ha byte) (health, armor uint8) { return decodeHealthArmor(ha) }

// encodeHealthArmor packs health (high nibble) and armor (low nibble).
//
// DECISION (Opsi A, floor division): health/armor >= 100 -> nibble 0xF;
// otherwise nibble = v / 7 (integer floor). This matches RakSAMP server
// netgame.cpp and SAMP.Lua. OPEN ITEM: not yet verified against the real
// 0.3.7 client via golden capture; see docs/protocol.md §7.
func encodeHealthArmor(health, armor uint8) byte {
	var hi, lo byte
	if health >= 100 {
		hi = 0xF
	} else if health > 0 {
		hi = health / 7
	}
	if armor >= 100 {
		lo = 0xF
	} else if armor > 0 {
		lo = armor / 7
	}
	return hi<<4 | lo
}

// decodeHealthArmor unpacks the health/armor byte (nibble * 7).
func decodeHealthArmor(ha byte) (health, armor uint8) {
	health = (ha >> 4) * 7
	if health > 100 {
		health = 100
	}
	armor = (ha & 0x0F) * 7
	if armor > 100 {
		armor = 100
	}
	return health, armor
}

// readVEC3 reads three raw float32 values.
func readVEC3(bs *raknet.BitStream, x, y, z *float32) error {
	for _, dst := range []*float32{x, y, z} {
		v, err := bs.ReadUint32()
		if err != nil {
			return err
		}
		*dst = math.Float32frombits(v)
	}
	return nil
}

// writeVEC3 writes three raw float32 values.
func writeVEC3(bs *raknet.BitStream, x, y, z float32) {
	bs.WriteUint32(math.Float32bits(x))
	bs.WriteUint32(math.Float32bits(y))
	bs.WriteUint32(math.Float32bits(z))
}

// readNormQuat reads a RakNet normalized quaternion: 4 sign bits followed by
// three uint16 components (fabs(v)*65535); w is reconstructed. Mirrors
// NetworkBitStream::ReadNormQuat and GTAQuat on the wire.
func readNormQuat(bs *raknet.BitStream) (w, x, y, z float32, err error) {
	signs := [4]bool{}
	for i := 0; i < 4; i++ {
		if signs[i], err = bs.ReadBool(); err != nil {
			return
		}
	}
	comp := [3]uint16{}
	for i := 0; i < 3; i++ {
		if comp[i], err = bs.ReadUint16(); err != nil {
			return
		}
	}
	x = float32(comp[0]) / 65535.0
	y = float32(comp[1]) / 65535.0
	z = float32(comp[2]) / 65535.0
	if signs[1] {
		x = -x
	}
	if signs[2] {
		y = -y
	}
	if signs[3] {
		z = -z
	}
	diff := 1.0 - x*x - y*y - z*z
	if diff < 0 {
		diff = 0
	}
	w = float32(math.Sqrt(float64(diff)))
	if signs[0] {
		w = -w
	}
	return
}

// writeNormQuat writes a RakNet normalized quaternion (inverse of readNormQuat).
func writeNormQuat(bs *raknet.BitStream, w, x, y, z float32) {
	bs.WriteBool(w < 0)
	bs.WriteBool(x < 0)
	bs.WriteBool(y < 0)
	bs.WriteBool(z < 0)
	bs.WriteUint16(uint16(math.Abs(float64(x)) * 65535.0))
	bs.WriteUint16(uint16(math.Abs(float64(y)) * 65535.0))
	bs.WriteUint16(uint16(math.Abs(float64(z)) * 65535.0))
}

// readCompressedVector reads a RakNet compressed vector: float32 magnitude
// followed by three compressed floats (int16 / 32767.5 - 1).
func readCompressedVector(bs *raknet.BitStream, x, y, z *float32) error {
	bs.AlignRead()
	magBytes, err := bs.ReadAlignedBytes(4)
	if err != nil {
		return err
	}
	mag := math.Float32frombits(uint32(magBytes[0]) | uint32(magBytes[1])<<8 | uint32(magBytes[2])<<16 | uint32(magBytes[3])<<24)
	if mag == 0 {
		*x, *y, *z = 0, 0, 0
		return nil
	}
	cf := func() (float32, error) {
		v, e := bs.ReadCompressedUint16()
		if e != nil {
			return 0, e
		}
		return float32(float64(v)/32767.5 - 1.0), nil
	}
	cx, e := cf()
	if e != nil {
		return e
	}
	cy, e := cf()
	if e != nil {
		return e
	}
	cz, e := cf()
	if e != nil {
		return e
	}
	*x = cx * mag
	*y = cy * mag
	*z = cz * mag
	return nil
}

// writeCompressedVector writes a RakNet compressed vector.
func writeCompressedVector(bs *raknet.BitStream, x, y, z float32) {
	bs.AlignWrite()
	mag := float32(math.Sqrt(float64(x*x + y*y + z*z)))
	bs.WriteUint32(math.Float32bits(mag))
	if mag > 0 {
		bs.WriteCompressedUint16(uint16((float64(x/mag) + 1.0) * 32767.5))
		bs.WriteCompressedUint16(uint16((float64(y/mag) + 1.0) * 32767.5))
		bs.WriteCompressedUint16(uint16((float64(z/mag) + 1.0) * 32767.5))
	}
}

// writeOptionalU16 writes a bool flag then the value only when non-zero.
func writeOptionalU16(bs *raknet.BitStream, v uint16) {
	if v != 0 {
		bs.WriteBool(true)
		bs.WriteUint16(v)
	} else {
		bs.WriteBool(false)
	}
}

// readOptionalU16 reads a bool flag then the value when present.
func readOptionalU16(bs *raknet.BitStream, dst *uint16) error {
	has, err := bs.ReadBool()
	if err != nil {
		return err
	}
	if !has {
		*dst = 0
		return nil
	}
	v, err := bs.ReadUint16()
	if err != nil {
		return err
	}
	*dst = v
	return nil
}

// ParseOnFootSyncBroadcast decodes a server -> client on-foot sync broadcast
// (the relayed form with player id and conditional optional flags). The
// leading ID byte (207) must be included in payload. It returns the player id
// and the decoded state.
func ParseOnFootSyncBroadcast(payload []byte) (playerID uint16, d PlayerSyncData, err error) {
	playerID, d, _, err = ParseOnFootSyncBroadcastHA(payload)
	return playerID, d, err
}

// ParseOnFootSyncBroadcastHA is ParseOnFootSyncBroadcast but also returns the
// raw health/armor byte as it appeared on the wire, so callers can inspect the
// server's nibble encoding (floor vs CEILDIV) without a decode round-trip.
func ParseOnFootSyncBroadcastHA(payload []byte) (playerID uint16, d PlayerSyncData, ha byte, err error) {
	bs := raknet.FromBytes(payload)
	var b byte
	if b, err = bs.ReadUint8(); err != nil {
		return 0, d, 0, err
	}
	if b != PacketIDOnFootSync {
		return 0, d, 0, errMalformedID
	}
	if playerID, err = bs.ReadUint16(); err != nil {
		return 0, d, 0, err
	}
	// Conditional analog flags.
	if err = readOptionalU16(bs, &d.LeftRight); err != nil {
		return 0, d, 0, err
	}
	if err = readOptionalU16(bs, &d.UpDown); err != nil {
		return 0, d, 0, err
	}
	if d.Keys, err = bs.ReadUint16(); err != nil {
		return 0, d, 0, err
	}
	readF32b := func(dst *float32) error {
		v, e := bs.ReadUint32()
		if e != nil {
			return e
		}
		*dst = math.Float32frombits(v)
		return nil
	}
	for _, f := range []*float32{&d.X, &d.Y, &d.Z} {
		if err = readF32b(f); err != nil {
			return 0, d, 0, err
		}
	}
	d.QuatW, d.QuatX, d.QuatY, d.QuatZ, err = readNormQuat(bs)
	if err != nil {
		return 0, d, 0, err
	}
	if ha, err = bs.ReadUint8(); err != nil {
		return 0, d, 0, err
	}
	d.Health, d.Armor = decodeHealthArmor(ha)
	var wa byte
	if wa, err = bs.ReadUint8(); err != nil {
		return 0, d, 0, err
	}
	d.Weapon, d.AdditionalKey = SplitWeaponAdditional(wa)
	if d.SpecialAction, err = bs.ReadUint8(); err != nil {
		return 0, d, 0, err
	}
	if err = readCompressedVector(bs, &d.VelX, &d.VelY, &d.VelZ); err != nil {
		return 0, d, 0, err
	}
	// Surf.
	if err = readOptionalU16(bs, &d.SurfVehicle); err != nil {
		return 0, d, 0, err
	}
	d.HasSurf = d.SurfVehicle > 0
	if d.HasSurf {
		readF32b(&d.SurfOffX)
		readF32b(&d.SurfOffY)
		readF32b(&d.SurfOffZ)
		if err != nil {
			return 0, d, 0, err
		}
	}
	// Anim.
	if err = readOptionalU16(bs, &d.AnimID); err != nil {
		return 0, d, 0, err
	}
	d.HasAnim = d.AnimID > 0
	if d.HasAnim {
		if d.AnimFlags, err = bs.ReadUint16(); err != nil {
			return 0, d, 0, err
		}
	}
	return playerID, d, ha, nil
}

var errMalformedID = errors.New("protocol: unexpected packet id")
