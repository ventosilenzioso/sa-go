package protocol

import (
	"math"

	"gosamp/internal/raknet"
)

// Actor RPCs. Actors are static, script-controlled ped models: they are NOT
// players/NPCs, do not occupy a player id slot, and have their own pool
// (MAX_ACTORS = 1000 in SA-MP). Since SA-MP 0.3.7 R2 actors are invulnerable by
// default unless the server sets otherwise.
//
// Wire layouts (open.mp Shared/NetCode/actor.hpp):
//   ShowActor (171):              uint16 actorID, uint32 skinID, VEC3 position,
//                                 float angle, float health, uint8 invulnerable
//   HideActor (172):              uint16 actorID
//   ApplyActorAnimation (173):    uint16 actorID, dynStr8 animLib,
//                                 dynStr8 animName, float delta, 4 bools
//                                 (loop, lockX, lockY, freeze), uint32 time
//   ClearActorAnimations (174):   uint16 actorID
//   SetActorFacingAngle (175):    uint16 actorID, float angle
//   SetActorPos (176):            uint16 actorID, VEC3 position
//   SetActorHealth (178):         uint16 actorID, float health
//   OnPlayerDamageActor (177):    bool unknown, uint16 actorID, float damage,
//                                 uint32 weaponID, uint32 bodypart (client -> server)

// MaxActors is the SA-MP actor pool size.
const MaxActors = 1000

// BuildShowActor encodes ShowActor (RPC 171).
func BuildShowActor(actorID uint16, skinID uint32, x, y, z, angle, health float32, invulnerable bool) []byte {
	bs := raknet.New()
	bs.WriteUint16(actorID)
	bs.WriteUint32(skinID)
	writeVEC3(bs, x, y, z)
	bs.WriteUint32(math.Float32bits(angle))
	bs.WriteUint32(math.Float32bits(health))
	if invulnerable {
		bs.WriteUint8(1)
	} else {
		bs.WriteUint8(0)
	}
	return bs.Bytes()[:bs.Len()]
}

// ParseShowActor decodes ShowActor (RPC 171).
func ParseShowActor(payload []byte) (actorID uint16, skinID uint32, x, y, z, angle, health float32, invulnerable bool, err error) {
	bs := raknet.FromBytes(payload)
	if actorID, err = bs.ReadUint16(); err != nil {
		return
	}
	if skinID, err = bs.ReadUint32(); err != nil {
		return
	}
	if err = readVEC3(bs, &x, &y, &z); err != nil {
		return
	}
	a, err := bs.ReadUint32()
	if err != nil {
		return
	}
	angle = math.Float32frombits(a)
	h, err := bs.ReadUint32()
	if err != nil {
		return
	}
	health = math.Float32frombits(h)
	inv, err := bs.ReadUint8()
	if err != nil {
		return
	}
	invulnerable = inv != 0
	return
}

// BuildHideActor encodes HideActor (RPC 172).
func BuildHideActor(actorID uint16) []byte {
	bs := raknet.New()
	bs.WriteUint16(actorID)
	return bs.Bytes()[:bs.Len()]
}

// ParseHideActor decodes HideActor (RPC 172).
func ParseHideActor(payload []byte) (uint16, error) {
	return raknet.FromBytes(payload).ReadUint16()
}

// BuildApplyActorAnimation encodes ApplyActorAnimation (RPC 173).
func BuildApplyActorAnimation(actorID uint16, animLib, animName string, delta float32, loop, lockX, lockY, freeze bool, time uint32) []byte {
	bs := raknet.New()
	bs.WriteUint16(actorID)
	writeDynStr8(bs, animLib)
	writeDynStr8(bs, animName)
	bs.WriteUint32(math.Float32bits(delta))
	bs.WriteBool(loop)
	bs.WriteBool(lockX)
	bs.WriteBool(lockY)
	bs.WriteBool(freeze)
	bs.WriteUint32(time)
	return bs.Bytes()[:bs.Len()]
}

// ParseApplyActorAnimation decodes ApplyActorAnimation (RPC 173).
func ParseApplyActorAnimation(payload []byte) (actorID uint16, animLib, animName string, delta float32, loop, lockX, lockY, freeze bool, time uint32, err error) {
	bs := raknet.FromBytes(payload)
	if actorID, err = bs.ReadUint16(); err != nil {
		return
	}
	if animLib, err = readDynStr8(bs); err != nil {
		return
	}
	if animName, err = readDynStr8(bs); err != nil {
		return
	}
	d, err := bs.ReadUint32()
	if err != nil {
		return
	}
	delta = math.Float32frombits(d)
	if loop, err = bs.ReadBool(); err != nil {
		return
	}
	if lockX, err = bs.ReadBool(); err != nil {
		return
	}
	if lockY, err = bs.ReadBool(); err != nil {
		return
	}
	if freeze, err = bs.ReadBool(); err != nil {
		return
	}
	time, err = bs.ReadUint32()
	return
}

// BuildClearActorAnimations encodes ClearActorAnimations (RPC 174).
func BuildClearActorAnimations(actorID uint16) []byte {
	bs := raknet.New()
	bs.WriteUint16(actorID)
	return bs.Bytes()[:bs.Len()]
}

// BuildSetActorFacingAngle encodes SetActorFacingAngle (RPC 175).
func BuildSetActorFacingAngle(actorID uint16, angle float32) []byte {
	bs := raknet.New()
	bs.WriteUint16(actorID)
	bs.WriteUint32(math.Float32bits(angle))
	return bs.Bytes()[:bs.Len()]
}

// ParseSetActorFacingAngle decodes SetActorFacingAngle (RPC 175).
func ParseSetActorFacingAngle(payload []byte) (actorID uint16, angle float32, err error) {
	bs := raknet.FromBytes(payload)
	if actorID, err = bs.ReadUint16(); err != nil {
		return
	}
	a, err := bs.ReadUint32()
	if err != nil {
		return 0, 0, err
	}
	angle = math.Float32frombits(a)
	return
}

// BuildSetActorPos encodes SetActorPos (RPC 176).
func BuildSetActorPos(actorID uint16, x, y, z float32) []byte {
	bs := raknet.New()
	bs.WriteUint16(actorID)
	writeVEC3(bs, x, y, z)
	return bs.Bytes()[:bs.Len()]
}

// ParseSetActorPos decodes SetActorPos (RPC 176).
func ParseSetActorPos(payload []byte) (actorID uint16, x, y, z float32, err error) {
	bs := raknet.FromBytes(payload)
	if actorID, err = bs.ReadUint16(); err != nil {
		return
	}
	err = readVEC3(bs, &x, &y, &z)
	return
}

// BuildSetActorHealth encodes SetActorHealth (RPC 178).
func BuildSetActorHealth(actorID uint16, health float32) []byte {
	bs := raknet.New()
	bs.WriteUint16(actorID)
	bs.WriteUint32(math.Float32bits(health))
	return bs.Bytes()[:bs.Len()]
}

// ParseSetActorHealth decodes SetActorHealth (RPC 178).
func ParseSetActorHealth(payload []byte) (actorID uint16, health float32, err error) {
	bs := raknet.FromBytes(payload)
	if actorID, err = bs.ReadUint16(); err != nil {
		return
	}
	h, err := bs.ReadUint32()
	if err != nil {
		return 0, 0, err
	}
	health = math.Float32frombits(h)
	return
}

// ActorDamage is the decoded OnPlayerDamageActor (RPC 177) client report.
type ActorDamage struct {
	Unknown  bool
	ActorID  uint16
	Damage   float32
	WeaponID uint32
	BodyPart uint32
}

// ParseActorDamage decodes OnPlayerDamageActor (RPC 177).
func ParseActorDamage(payload []byte) (ActorDamage, error) {
	var d ActorDamage
	bs := raknet.FromBytes(payload)
	var err error
	if d.Unknown, err = bs.ReadBool(); err != nil {
		return d, err
	}
	if d.ActorID, err = bs.ReadUint16(); err != nil {
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
	d.BodyPart, err = bs.ReadUint32()
	return d, err
}