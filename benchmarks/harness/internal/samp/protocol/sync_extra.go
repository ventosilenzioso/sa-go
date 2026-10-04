package protocol

import (
	"fmt"
	"math"

	"gosamp/internal/raknet"
)

// SA-MP player sync packet codecs that were previously missing, ported from
// open.mp Shared/NetCode (core.hpp / vehicle.hpp) and cross-checked against the
// samp-packet-list wiki. These are hot-path packets sent many times per second.

// AimSyncData is the decoded aim sync (ID 203) payload.
type AimSyncData struct {
	CamMode     uint8
	CamFrontX   float32
	CamFrontY   float32
	CamFrontZ   float32
	CamPosX     float32
	CamPosY     float32
	CamPosZ     float32
	AimZ        float32
	CamZoom     uint8 // 6 bits
	WeaponState uint8 // 2 bits
	AspectRatio uint8
}

// ParseAimSync decodes a client -> server aim sync (ID 203) payload (no ID).
func ParseAimSync(payload []byte) (AimSyncData, error) {
	var d AimSyncData
	bs := raknet.FromBytes(payload)
	var err error
	if d.CamMode, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if err = readVEC3(bs, &d.CamFrontX, &d.CamFrontY, &d.CamFrontZ); err != nil {
		return d, err
	}
	if err = readVEC3(bs, &d.CamPosX, &d.CamPosY, &d.CamPosZ); err != nil {
		return d, err
	}
	var az uint32
	if az, err = bs.ReadUint32(); err != nil {
		return d, err
	}
	d.AimZ = math.Float32frombits(az)
	var zw byte
	if zw, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	d.CamZoom = zw & 0x3F
	d.WeaponState = (zw >> 6) & 0x03
	if d.AspectRatio, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	return d, nil
}

// BuildAimSync builds a client -> server aim sync (ID 203) payload (with ID).
func BuildAimSync(d AimSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDAimSync)
	bs.WriteUint8(d.CamMode)
	writeVEC3(bs, d.CamFrontX, d.CamFrontY, d.CamFrontZ)
	writeVEC3(bs, d.CamPosX, d.CamPosY, d.CamPosZ)
	bs.WriteUint32(math.Float32bits(d.AimZ))
	bs.WriteUint8((d.CamZoom & 0x3F) | (d.WeaponState&0x03)<<6)
	bs.WriteUint8(d.AspectRatio)
	return bs.Bytes()[:bs.Len()]
}

// BuildAimSyncBroadcast builds the server -> client aim sync (ID 203) packet:
// the client payload plus the sender's player id after the packet id
// (open.mp PlayerAimSync::write). Used to relay aim to nearby players.
func BuildAimSyncBroadcast(playerID uint16, d AimSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDAimSync)
	bs.WriteUint16(playerID)
	bs.WriteUint8(d.CamMode)
	writeVEC3(bs, d.CamFrontX, d.CamFrontY, d.CamFrontZ)
	writeVEC3(bs, d.CamPosX, d.CamPosY, d.CamPosZ)
	bs.WriteUint32(math.Float32bits(d.AimZ))
	bs.WriteUint8((d.CamZoom & 0x3F) | (d.WeaponState&0x03)<<6)
	bs.WriteUint8(d.AspectRatio)
	return bs.Bytes()[:bs.Len()]
}

// ParseAimSyncBroadcast decodes an aim sync broadcast (with the packet id and
// player id). Returns the sender id and the state.
func ParseAimSyncBroadcast(payload []byte) (playerID uint16, d AimSyncData, err error) {
	bs := raknet.FromBytes(payload)
	var id byte
	if id, err = bs.ReadUint8(); err != nil {
		return 0, d, err
	}
	if id != PacketIDAimSync {
		return 0, d, errMalformedID
	}
	if playerID, err = bs.ReadUint16(); err != nil {
		return 0, d, err
	}
	if d.CamMode, err = bs.ReadUint8(); err != nil {
		return 0, d, err
	}
	if err = readVEC3(bs, &d.CamFrontX, &d.CamFrontY, &d.CamFrontZ); err != nil {
		return 0, d, err
	}
	if err = readVEC3(bs, &d.CamPosX, &d.CamPosY, &d.CamPosZ); err != nil {
		return 0, d, err
	}
	az, err := bs.ReadUint32()
	if err != nil {
		return 0, d, err
	}
	d.AimZ = math.Float32frombits(az)
	zw, err := bs.ReadUint8()
	if err != nil {
		return 0, d, err
	}
	d.CamZoom = zw & 0x3F
	d.WeaponState = (zw >> 6) & 0x03
	if d.AspectRatio, err = bs.ReadUint8(); err != nil {
		return 0, d, err
	}
	return playerID, d, nil
}

// BulletSyncData is the decoded bullet sync (ID 206) payload.
type BulletSyncData struct {
	HitType                   uint8
	HitID                     uint16
	OriginX, OriginY, OriginZ float32
	HitX, HitY, HitZ          float32
	OffX, OffY, OffZ          float32
	WeaponID                  uint8
}

// ParseBulletSync decodes a client -> server bullet sync (ID 206) payload.
func ParseBulletSync(payload []byte) (BulletSyncData, error) {
	var d BulletSyncData
	bs := raknet.FromBytes(payload)
	var err error
	if d.HitType, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if d.HitID, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	if err = readVEC3(bs, &d.OriginX, &d.OriginY, &d.OriginZ); err != nil {
		return d, err
	}
	if err = readVEC3(bs, &d.HitX, &d.HitY, &d.HitZ); err != nil {
		return d, err
	}
	if err = readVEC3(bs, &d.OffX, &d.OffY, &d.OffZ); err != nil {
		return d, err
	}
	if d.WeaponID, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	return d, nil
}

// BuildBulletSync builds a client -> server bullet sync (ID 206) payload.
func BuildBulletSync(d BulletSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDBulletSync)
	bs.WriteUint8(d.HitType)
	bs.WriteUint16(d.HitID)
	writeVEC3(bs, d.OriginX, d.OriginY, d.OriginZ)
	writeVEC3(bs, d.HitX, d.HitY, d.HitZ)
	writeVEC3(bs, d.OffX, d.OffY, d.OffZ)
	bs.WriteUint8(d.WeaponID)
	return bs.Bytes()[:bs.Len()]
}

// BuildBulletSyncBroadcast builds the server -> client bullet sync (ID 206)
// packet: the client payload plus the shooter's player id after the packet id
// (open.mp PlayerBulletSync::write). Used to relay shots to nearby players.
func BuildBulletSyncBroadcast(playerID uint16, d BulletSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDBulletSync)
	bs.WriteUint16(playerID)
	bs.WriteUint8(d.HitType)
	bs.WriteUint16(d.HitID)
	writeVEC3(bs, d.OriginX, d.OriginY, d.OriginZ)
	writeVEC3(bs, d.HitX, d.HitY, d.HitZ)
	writeVEC3(bs, d.OffX, d.OffY, d.OffZ)
	bs.WriteUint8(d.WeaponID)
	return bs.Bytes()[:bs.Len()]
}

// ParseBulletSyncBroadcast decodes a bullet sync broadcast (with packet id and
// player id).
func ParseBulletSyncBroadcast(payload []byte) (playerID uint16, d BulletSyncData, err error) {
	bs := raknet.FromBytes(payload)
	var id byte
	if id, err = bs.ReadUint8(); err != nil {
		return 0, d, err
	}
	if id != PacketIDBulletSync {
		return 0, d, errMalformedID
	}
	if playerID, err = bs.ReadUint16(); err != nil {
		return 0, d, err
	}
	if d.HitType, err = bs.ReadUint8(); err != nil {
		return 0, d, err
	}
	if d.HitID, err = bs.ReadUint16(); err != nil {
		return 0, d, err
	}
	if err = readVEC3(bs, &d.OriginX, &d.OriginY, &d.OriginZ); err != nil {
		return 0, d, err
	}
	if err = readVEC3(bs, &d.HitX, &d.HitY, &d.HitZ); err != nil {
		return 0, d, err
	}
	if err = readVEC3(bs, &d.OffX, &d.OffY, &d.OffZ); err != nil {
		return 0, d, err
	}
	if d.WeaponID, err = bs.ReadUint8(); err != nil {
		return 0, d, err
	}
	return playerID, d, nil
}

// StatsSyncData is the decoded stats update (ID 205) payload.
type StatsSyncData struct {
	Money      int32
	DrunkLevel int32
}

// ParseStatsSync decodes a client -> server stats update (ID 205) payload.
func ParseStatsSync(payload []byte) (StatsSyncData, error) {
	var d StatsSyncData
	bs := raknet.FromBytes(payload)
	var err error
	m, err := bs.ReadUint32()
	if err != nil {
		return d, err
	}
	d.Money = int32(m)
	dl, err := bs.ReadUint32()
	if err != nil {
		return d, err
	}
	d.DrunkLevel = int32(dl)
	return d, nil
}

// BuildStatsSync builds a client -> server stats update (ID 205) payload.
func BuildStatsSync(d StatsSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDStatsUpdate)
	bs.WriteUint32(uint32(d.Money))
	bs.WriteUint32(uint32(d.DrunkLevel))
	return bs.Bytes()[:bs.Len()]
}

// WeaponSlot is one entry of a weapons update (slot + weapon id + ammo).
type WeaponSlot struct {
	Slot   uint8
	Weapon uint8
	Ammo   uint16
}

// WeaponsUpdateData is the decoded weapons update (ID 204) payload.
type WeaponsUpdateData struct {
	TargetPlayer uint16
	TargetActor  uint16
	Weapons      []WeaponSlot
}

// ParseWeaponsUpdate decodes a client -> server weapons update (ID 204).
func ParseWeaponsUpdate(payload []byte) (WeaponsUpdateData, error) {
	var d WeaponsUpdateData
	bs := raknet.FromBytes(payload)
	var err error
	if d.TargetPlayer, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	if d.TargetActor, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	for i := 0; i < 13; i++ {
		slot, err := bs.ReadUint8()
		if err != nil {
			return d, err
		}
		if slot == 0xFF {
			break
		}
		weapon, err := bs.ReadUint8()
		if err != nil {
			return d, err
		}
		ammo, err := bs.ReadUint16()
		if err != nil {
			return d, err
		}
		d.Weapons = append(d.Weapons, WeaponSlot{Slot: slot, Weapon: weapon, Ammo: ammo})
	}
	return d, nil
}

// BuildWeaponsUpdate builds a client -> server weapons update (ID 204) payload.
func BuildWeaponsUpdate(d WeaponsUpdateData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDWeaponsUpdate)
	bs.WriteUint16(d.TargetPlayer)
	bs.WriteUint16(d.TargetActor)
	for _, w := range d.Weapons {
		bs.WriteUint8(w.Slot)
		bs.WriteUint8(w.Weapon)
		bs.WriteUint16(w.Ammo)
	}
	bs.WriteUint8(0xFF) // terminator slot id
	return bs.Bytes()[:bs.Len()]
}

// UnoccupiedSyncData is the decoded unoccupied vehicle sync (ID 209) payload.
type UnoccupiedSyncData struct {
	VehicleID                 uint16
	SeatID                    uint8
	RollX, RollY, RollZ       float32
	RotX, RotY, RotZ          float32
	X, Y, Z                   float32
	VelX, VelY, VelZ          float32
	AngVelX, AngVelY, AngVelZ float32
	Health                    float32
}

// ParseUnoccupiedSync decodes a client -> server unoccupied sync (ID 209).
func ParseUnoccupiedSync(payload []byte) (UnoccupiedSyncData, error) {
	var d UnoccupiedSyncData
	bs := raknet.FromBytes(payload)
	var err error
	if d.VehicleID, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	if d.SeatID, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if err = readVEC3(bs, &d.RollX, &d.RollY, &d.RollZ); err != nil {
		return d, err
	}
	if err = readVEC3(bs, &d.RotX, &d.RotY, &d.RotZ); err != nil {
		return d, err
	}
	if err = readVEC3(bs, &d.X, &d.Y, &d.Z); err != nil {
		return d, err
	}
	if err = readVEC3(bs, &d.VelX, &d.VelY, &d.VelZ); err != nil {
		return d, err
	}
	if err = readVEC3(bs, &d.AngVelX, &d.AngVelY, &d.AngVelZ); err != nil {
		return d, err
	}
	h, err := bs.ReadUint32()
	if err != nil {
		return d, err
	}
	d.Health = math.Float32frombits(h)
	return d, nil
}

// BuildUnoccupiedSync builds a client -> server unoccupied sync (ID 209).
func BuildUnoccupiedSync(d UnoccupiedSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDUnoccupied)
	bs.WriteUint16(d.VehicleID)
	bs.WriteUint8(d.SeatID)
	writeVEC3(bs, d.RollX, d.RollY, d.RollZ)
	writeVEC3(bs, d.RotX, d.RotY, d.RotZ)
	writeVEC3(bs, d.X, d.Y, d.Z)
	writeVEC3(bs, d.VelX, d.VelY, d.VelZ)
	writeVEC3(bs, d.AngVelX, d.AngVelY, d.AngVelZ)
	bs.WriteUint32(math.Float32bits(d.Health))
	return bs.Bytes()[:bs.Len()]
}

// BuildUnoccupiedSyncBroadcast rebuilds the server -> client unoccupied sync
// (ID 209) packet: [209][playerID u16][vehicleID u16][seatID u8]
// [roll VEC3][rotation VEC3][position VEC3][velocity VEC3][angular VEC3][health float].
// Mirrors open.mp NetCode::Packet::PlayerUnoccupiedSync::write.
func BuildUnoccupiedSyncBroadcast(playerID uint16, d UnoccupiedSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDUnoccupied)
	bs.WriteUint16(playerID)
	bs.WriteUint16(d.VehicleID)
	bs.WriteUint8(d.SeatID)
	writeVEC3(bs, d.RollX, d.RollY, d.RollZ)
	writeVEC3(bs, d.RotX, d.RotY, d.RotZ)
	writeVEC3(bs, d.X, d.Y, d.Z)
	writeVEC3(bs, d.VelX, d.VelY, d.VelZ)
	writeVEC3(bs, d.AngVelX, d.AngVelY, d.AngVelZ)
	bs.WriteUint32(math.Float32bits(d.Health))
	return bs.Bytes()[:bs.Len()]
}

// ParseUnoccupiedSyncBroadcast decodes a server -> client unoccupied sync (ID
// 209) that includes the packet id and player id.
func ParseUnoccupiedSyncBroadcast(payload []byte) (uint16, UnoccupiedSyncData, error) {
	var d UnoccupiedSyncData
	bs := raknet.FromBytes(payload)
	id, err := bs.ReadUint8()
	if err != nil {
		return 0, d, err
	}
	if id != PacketIDUnoccupied {
		return 0, d, fmt.Errorf("protocol: not an unoccupied sync: id %d", id)
	}
	playerID, err := bs.ReadUint16()
	if err != nil {
		return 0, d, err
	}
	if d.VehicleID, err = bs.ReadUint16(); err != nil {
		return 0, d, err
	}
	if d.SeatID, err = bs.ReadUint8(); err != nil {
		return 0, d, err
	}
	if err = readVEC3(bs, &d.RollX, &d.RollY, &d.RollZ); err != nil {
		return 0, d, err
	}
	if err = readVEC3(bs, &d.RotX, &d.RotY, &d.RotZ); err != nil {
		return 0, d, err
	}
	if err = readVEC3(bs, &d.X, &d.Y, &d.Z); err != nil {
		return 0, d, err
	}
	if err = readVEC3(bs, &d.VelX, &d.VelY, &d.VelZ); err != nil {
		return 0, d, err
	}
	if err = readVEC3(bs, &d.AngVelX, &d.AngVelY, &d.AngVelZ); err != nil {
		return 0, d, err
	}
	h, err := bs.ReadUint32()
	if err != nil {
		return 0, d, err
	}
	d.Health = math.Float32frombits(h)
	return playerID, d, nil
}

// TrailerSyncData is the decoded trailer sync (ID 210) payload.
type TrailerSyncData struct {
	VehicleID                  uint16
	X, Y, Z                    float32
	QuatW, QuatX, QuatY, QuatZ float32
	VelX, VelY, VelZ           float32
	TurnX, TurnY, TurnZ        float32
}

// ParseTrailerSync decodes a client -> server trailer sync (ID 210).
func ParseTrailerSync(payload []byte) (TrailerSyncData, error) {
	var d TrailerSyncData
	bs := raknet.FromBytes(payload)
	var err error
	if d.VehicleID, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	if err = readVEC3(bs, &d.X, &d.Y, &d.Z); err != nil {
		return d, err
	}
	if err = readVEC4(bs, &d.QuatW, &d.QuatX, &d.QuatY, &d.QuatZ); err != nil {
		return d, err
	}
	if err = readVEC3(bs, &d.VelX, &d.VelY, &d.VelZ); err != nil {
		return d, err
	}
	if err = readVEC3(bs, &d.TurnX, &d.TurnY, &d.TurnZ); err != nil {
		return d, err
	}
	return d, nil
}

// BuildTrailerSync builds a client -> server trailer sync (ID 210).
func BuildTrailerSync(d TrailerSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDTrailerSync)
	bs.WriteUint16(d.VehicleID)
	writeVEC3(bs, d.X, d.Y, d.Z)
	writeVEC4(bs, d.QuatW, d.QuatX, d.QuatY, d.QuatZ)
	writeVEC3(bs, d.VelX, d.VelY, d.VelZ)
	writeVEC3(bs, d.TurnX, d.TurnY, d.TurnZ)
	return bs.Bytes()[:bs.Len()]
}

// BuildTrailerSyncBroadcast rebuilds the server -> client trailer sync (ID 210)
// packet: [210][playerID u16][vehicleID u16][position VEC3][quat VEC4]
// [velocity VEC3][turn VEC3]. Mirrors open.mp
// NetCode::Packet::PlayerTrailerSync::write.
func BuildTrailerSyncBroadcast(playerID uint16, d TrailerSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDTrailerSync)
	bs.WriteUint16(playerID)
	bs.WriteUint16(d.VehicleID)
	writeVEC3(bs, d.X, d.Y, d.Z)
	writeVEC4(bs, d.QuatW, d.QuatX, d.QuatY, d.QuatZ)
	writeVEC3(bs, d.VelX, d.VelY, d.VelZ)
	writeVEC3(bs, d.TurnX, d.TurnY, d.TurnZ)
	return bs.Bytes()[:bs.Len()]
}

// ParseTrailerSyncBroadcast decodes a server -> client trailer sync (ID 210)
// that includes the packet id and player id.
func ParseTrailerSyncBroadcast(payload []byte) (uint16, TrailerSyncData, error) {
	var d TrailerSyncData
	bs := raknet.FromBytes(payload)
	id, err := bs.ReadUint8()
	if err != nil {
		return 0, d, err
	}
	if id != PacketIDTrailerSync {
		return 0, d, fmt.Errorf("protocol: not a trailer sync: id %d", id)
	}
	playerID, err := bs.ReadUint16()
	if err != nil {
		return 0, d, err
	}
	if d.VehicleID, err = bs.ReadUint16(); err != nil {
		return 0, d, err
	}
	if err = readVEC3(bs, &d.X, &d.Y, &d.Z); err != nil {
		return 0, d, err
	}
	if err = readVEC4(bs, &d.QuatW, &d.QuatX, &d.QuatY, &d.QuatZ); err != nil {
		return 0, d, err
	}
	if err = readVEC3(bs, &d.VelX, &d.VelY, &d.VelZ); err != nil {
		return 0, d, err
	}
	if err = readVEC3(bs, &d.TurnX, &d.TurnY, &d.TurnZ); err != nil {
		return 0, d, err
	}
	return playerID, d, nil
}

// SpectatingSyncData is the decoded spectating sync (ID 212) payload.
type SpectatingSyncData struct {
	LeftRight uint16
	UpDown    uint16
	Keys      uint16
	X, Y, Z   float32
}

// ParseSpectatingSync decodes a client -> server spectating sync (ID 212).
func ParseSpectatingSync(payload []byte) (SpectatingSyncData, error) {
	var d SpectatingSyncData
	bs := raknet.FromBytes(payload)
	var err error
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

// BuildSpectatingSync builds a client -> server spectating sync (ID 212).
func BuildSpectatingSync(d SpectatingSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDSpectating)
	bs.WriteUint16(d.LeftRight)
	bs.WriteUint16(d.UpDown)
	bs.WriteUint16(d.Keys)
	writeVEC3(bs, d.X, d.Y, d.Z)
	return bs.Bytes()[:bs.Len()]
}

// MarkerEntry is one player entry in a markers sync (ID 208, server -> client).
type MarkerEntry struct {
	PlayerID uint16
	Stream   bool
	X, Y, Z  int16 // only sent when Stream is true
}

// MarkersSyncData is the server -> client markers sync (ID 208) payload.
type MarkersSyncData struct {
	Markers []MarkerEntry
}

// BuildMarkersSync builds a markers sync (ID 208) payload. Coordinates are
// INT16 and only written for entries with Stream == true, so a packet can mix
// streamed and non-streamed players.
func BuildMarkersSync(d MarkersSyncData) []byte {
	bs := raknet.New()
	bs.WriteUint8(PacketIDMarkersSync)
	bs.WriteUint32(uint32(len(d.Markers)))
	for _, m := range d.Markers {
		bs.WriteUint16(m.PlayerID)
		bs.WriteBool(m.Stream)
		if m.Stream {
			bs.WriteUint16(uint16(m.X))
			bs.WriteUint16(uint16(m.Y))
			bs.WriteUint16(uint16(m.Z))
		}
	}
	return bs.Bytes()[:bs.Len()]
}

// ParseMarkersSync decodes a markers sync (ID 208) payload.
func ParseMarkersSync(payload []byte) (MarkersSyncData, error) {
	var d MarkersSyncData
	bs := raknet.FromBytes(payload)
	var err error
	var id byte
	if id, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if id != PacketIDMarkersSync {
		return d, errMalformedID
	}
	count, err := bs.ReadUint32()
	if err != nil {
		return d, err
	}
	for i := uint32(0); i < count; i++ {
		var m MarkerEntry
		if m.PlayerID, err = bs.ReadUint16(); err != nil {
			return d, err
		}
		if m.Stream, err = bs.ReadBool(); err != nil {
			return d, err
		}
		if m.Stream {
			x, e := bs.ReadUint16()
			if e != nil {
				return d, e
			}
			y, e := bs.ReadUint16()
			if e != nil {
				return d, e
			}
			z, e := bs.ReadUint16()
			if e != nil {
				return d, e
			}
			m.X, m.Y, m.Z = int16(x), int16(y), int16(z)
		}
		d.Markers = append(d.Markers, m)
	}
	return d, nil
}

// readVEC4 reads four raw float32 values.
func readVEC4(bs *raknet.BitStream, w, x, y, z *float32) error {
	for _, dst := range []*float32{w, x, y, z} {
		v, err := bs.ReadUint32()
		if err != nil {
			return err
		}
		*dst = math.Float32frombits(v)
	}
	return nil
}

// writeVEC4 writes four raw float32 values.
func writeVEC4(bs *raknet.BitStream, w, x, y, z float32) {
	bs.WriteUint32(math.Float32bits(w))
	bs.WriteUint32(math.Float32bits(x))
	bs.WriteUint32(math.Float32bits(y))
	bs.WriteUint32(math.Float32bits(z))
}

// GiveTakeDamageData is the decoded GiveTakeDamage (RPC 115) payload,
// client -> server: player reports dealing or receiving damage.
type GiveTakeDamageData struct {
	GiveOrTake bool // true = target receives damage
	PlayerID   uint16
	Damage     float32
	WeaponID   uint32
	BodyPart   uint32
}

// ParseGiveTakeDamage decodes RPC 115.
func ParseGiveTakeDamage(payload []byte) (GiveTakeDamageData, error) {
	var d GiveTakeDamageData
	bs := raknet.FromBytes(payload)
	var err error
	if d.GiveOrTake, err = bs.ReadBool(); err != nil {
		return d, err
	}
	if d.PlayerID, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	dmg, err := bs.ReadUint32()
	if err != nil {
		return d, err
	}
	d.Damage = math.Float32frombits(dmg)
	if d.WeaponID, err = bs.ReadUint32(); err != nil {
		return d, err
	}
	if d.BodyPart, err = bs.ReadUint32(); err != nil {
		return d, err
	}
	return d, nil
}

// BuildGiveTakeDamage builds an RPC 115 payload (with no RPC id/frame).
func BuildGiveTakeDamage(d GiveTakeDamageData) []byte {
	bs := raknet.New()
	bs.WriteBool(d.GiveOrTake)
	bs.WriteUint16(d.PlayerID)
	bs.WriteUint32(math.Float32bits(d.Damage))
	bs.WriteUint32(d.WeaponID)
	bs.WriteUint32(d.BodyPart)
	return bs.Bytes()[:bs.Len()]
}
