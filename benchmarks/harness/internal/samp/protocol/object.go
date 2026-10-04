package protocol

import (
	"math"

	"gosamp/internal/raknet"
)

// Object RPCs (server -> client). SA-MP object ids are 0-based with a pool of
// MAX_OBJECTS = 2000; INVALID_OBJECT_ID = 0xFFFF.
//
// Wire layout (SA-MP 0.3.7 / samp-packet-list, CreateObject RPC 44):
//   uint16 objectID, uint32 modelID, VEC3 position, VEC3 rotation,
//   float drawDistance, uint8 noCameraCollision,
//   uint16 attachedObject, uint16 attachedVehicle,
//   VEC3 attachOffset, VEC3 attachRotation, uint8 syncRotation
//
// Note: open.mp's Shared/NetCode/object.hpp adds a trailing material count +
// per-slot material block and orders attachedVehicle before attachedObject. That
// is the open.mp-client form. Because this server targets the SA-MP 0.3.7 client
// (Thunder-SAMP), we emit the 0.3.7 form: no material block, attachedObject then
// attachedVehicle, and the attachment offset/rotation written unconditionally.
// For non-attached objects both orderings send 0xFFFF/0xFFFF (indistinguishable).
//
//   DestroyObject (47):        uint16 objectID
//   SetObjectPos (45):         uint16 objectID, VEC3 position
//   SetObjectRotation (46):    uint16 objectID, VEC3 rotation
//   MoveObject (99):           uint16 objectID, VEC3 currentPos, VEC3 targetPos,
//                              float speed, VEC3 targetRot
//   StopObject (122):          uint16 objectID

// MaxObjects is the SA-MP global object pool size.
const MaxObjects = 2000

// NoAttachObject is the "not attached" sentinel for object attachment ids.
const NoAttachObject = 0xFFFF

// ObjectSpawn describes an object for CreateObject (RPC 44).
type ObjectSpawn struct {
	ModelID           int32
	X, Y, Z           float32
	RotX, RotY, RotZ  float32
	DrawDistance      float32
	NoCameraCollision bool
	AttachObject      uint16
	AttachVehicle     uint16
	AttachOffsetX     float32
	AttachOffsetY     float32
	AttachOffsetZ     float32
	AttachRotX        float32
	AttachRotY        float32
	AttachRotZ        float32
	SyncRotation      uint8
}

// BuildCreateObject encodes CreateObject (RPC 44) in the SA-MP 0.3.7 layout.
func BuildCreateObject(objectID uint16, d ObjectSpawn) []byte {
	bs := raknet.New()
	bs.WriteUint16(objectID)
	bs.WriteUint32(uint32(d.ModelID))
	writeVEC3(bs, d.X, d.Y, d.Z)
	writeVEC3(bs, d.RotX, d.RotY, d.RotZ)
	bs.WriteUint32(math.Float32bits(d.DrawDistance))
	if d.NoCameraCollision {
		bs.WriteUint8(1)
	} else {
		bs.WriteUint8(0)
	}
	bs.WriteUint16(d.AttachObject)
	bs.WriteUint16(d.AttachVehicle)
	writeVEC3(bs, d.AttachOffsetX, d.AttachOffsetY, d.AttachOffsetZ)
	writeVEC3(bs, d.AttachRotX, d.AttachRotY, d.AttachRotZ)
	bs.WriteUint8(d.SyncRotation)
	return bs.Bytes()[:bs.Len()]
}

// ParseCreateObject decodes CreateObject (RPC 44).
func ParseCreateObject(payload []byte) (objectID uint16, d ObjectSpawn, err error) {
	bs := raknet.FromBytes(payload)
	if objectID, err = bs.ReadUint16(); err != nil {
		return
	}
	model, err := bs.ReadUint32()
	if err != nil {
		return
	}
	d.ModelID = int32(model)
	if err = readVEC3(bs, &d.X, &d.Y, &d.Z); err != nil {
		return
	}
	if err = readVEC3(bs, &d.RotX, &d.RotY, &d.RotZ); err != nil {
		return
	}
	dd, err := bs.ReadUint32()
	if err != nil {
		return
	}
	d.DrawDistance = math.Float32frombits(dd)
	ncc, err := bs.ReadUint8()
	if err != nil {
		return
	}
	d.NoCameraCollision = ncc != 0
	if d.AttachObject, err = bs.ReadUint16(); err != nil {
		return
	}
	if d.AttachVehicle, err = bs.ReadUint16(); err != nil {
		return
	}
	if err = readVEC3(bs, &d.AttachOffsetX, &d.AttachOffsetY, &d.AttachOffsetZ); err != nil {
		return
	}
	if err = readVEC3(bs, &d.AttachRotX, &d.AttachRotY, &d.AttachRotZ); err != nil {
		return
	}
	d.SyncRotation, err = bs.ReadUint8()
	return
}

// BuildDestroyObject encodes DestroyObject (RPC 47).
func BuildDestroyObject(objectID uint16) []byte {
	bs := raknet.New()
	bs.WriteUint16(objectID)
	return bs.Bytes()[:bs.Len()]
}

// ParseDestroyObject decodes DestroyObject (RPC 47).
func ParseDestroyObject(payload []byte) (uint16, error) {
	return raknet.FromBytes(payload).ReadUint16()
}

// BuildSetObjectPos encodes SetObjectPos (RPC 45).
func BuildSetObjectPos(objectID uint16, x, y, z float32) []byte {
	bs := raknet.New()
	bs.WriteUint16(objectID)
	writeVEC3(bs, x, y, z)
	return bs.Bytes()[:bs.Len()]
}

// ParseSetObjectPos decodes SetObjectPos (RPC 45).
func ParseSetObjectPos(payload []byte) (objectID uint16, x, y, z float32, err error) {
	bs := raknet.FromBytes(payload)
	if objectID, err = bs.ReadUint16(); err != nil {
		return
	}
	err = readVEC3(bs, &x, &y, &z)
	return
}

// BuildSetObjectRotation encodes SetObjectRotation (RPC 46).
func BuildSetObjectRotation(objectID uint16, rx, ry, rz float32) []byte {
	bs := raknet.New()
	bs.WriteUint16(objectID)
	writeVEC3(bs, rx, ry, rz)
	return bs.Bytes()[:bs.Len()]
}

// ParseSetObjectRotation decodes SetObjectRotation (RPC 46).
func ParseSetObjectRotation(payload []byte) (objectID uint16, rx, ry, rz float32, err error) {
	bs := raknet.FromBytes(payload)
	if objectID, err = bs.ReadUint16(); err != nil {
		return
	}
	err = readVEC3(bs, &rx, &ry, &rz)
	return
}

// BuildMoveObject encodes MoveObject (RPC 99).
func BuildMoveObject(objectID uint16, curX, curY, curZ, toX, toY, toZ, speed, toRX, toRY, toRZ float32) []byte {
	bs := raknet.New()
	bs.WriteUint16(objectID)
	writeVEC3(bs, curX, curY, curZ)
	writeVEC3(bs, toX, toY, toZ)
	bs.WriteUint32(math.Float32bits(speed))
	writeVEC3(bs, toRX, toRY, toRZ)
	return bs.Bytes()[:bs.Len()]
}

// ParseMoveObject decodes MoveObject (RPC 99).
func ParseMoveObject(payload []byte) (objectID uint16, curX, curY, curZ, toX, toY, toZ, speed, toRX, toRY, toRZ float32, err error) {
	bs := raknet.FromBytes(payload)
	if objectID, err = bs.ReadUint16(); err != nil {
		return
	}
	if err = readVEC3(bs, &curX, &curY, &curZ); err != nil {
		return
	}
	if err = readVEC3(bs, &toX, &toY, &toZ); err != nil {
		return
	}
	sp, err := bs.ReadUint32()
	if err != nil {
		return
	}
	speed = math.Float32frombits(sp)
	err = readVEC3(bs, &toRX, &toRY, &toRZ)
	return
}

// BuildStopObject encodes StopObject (RPC 122).
func BuildStopObject(objectID uint16) []byte {
	bs := raknet.New()
	bs.WriteUint16(objectID)
	return bs.Bytes()[:bs.Len()]
}
