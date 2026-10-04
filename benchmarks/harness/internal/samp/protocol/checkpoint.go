package protocol

import (
	"math"

	"gosamp/internal/raknet"
)

// Checkpoint RPCs (server -> client). SA-MP supports at most ONE active
// checkpoint of each kind per player at a time; calling Set again replaces the
// previous one asynchronously (no need to disable first). This is a protocol
// fact, not a gameplay rule.
//
// Wire layouts (open.mp Shared/NetCode/checkpoint.hpp):
//   SetCheckpoint (107):   VEC3 position, float size
//   DisableCheckpoint (37): (no parameters)
//   SetRaceCheckpoint (38): uint8 type, VEC3 position, VEC3 nextPosition, float size
//   DisableRaceCheckpoint (39): (no parameters)

// Race checkpoint types (open.mp CP_TYPE).
const (
	RaceCPGroundNormal = 0
	RaceCPGroundFinish = 1
	RaceCPGroundEmpty  = 2
	RaceCPAirNormal    = 3
	RaceCPAirFinish    = 4
	RaceCPAirRotating  = 5
	RaceCPAirStrobing  = 6
	RaceCPAirSwinging  = 7
	RaceCPAirBobbing   = 8
)

// BuildSetCheckpoint encodes SetCheckpoint (RPC 107): VEC3 + float radius.
func BuildSetCheckpoint(x, y, z, radius float32) []byte {
	bs := raknet.New()
	writeVEC3(bs, x, y, z)
	bs.WriteUint32(math.Float32bits(radius))
	return bs.Bytes()[:bs.Len()]
}

// ParseSetCheckpoint decodes SetCheckpoint (RPC 107).
func ParseSetCheckpoint(payload []byte) (x, y, z, radius float32, err error) {
	bs := raknet.FromBytes(payload)
	if err = readVEC3(bs, &x, &y, &z); err != nil {
		return
	}
	r, err := bs.ReadUint32()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	radius = math.Float32frombits(r)
	return
}

// BuildDisableCheckpoint encodes DisableCheckpoint (RPC 37): no parameters.
func BuildDisableCheckpoint() []byte { return []byte{} }

// BuildSetRaceCheckpoint encodes SetRaceCheckpoint (RPC 38):
// uint8 type, VEC3 position, VEC3 nextPosition, float radius.
func BuildSetRaceCheckpoint(cpType uint8, x, y, z, nextX, nextY, nextZ, radius float32) []byte {
	bs := raknet.New()
	bs.WriteUint8(cpType)
	writeVEC3(bs, x, y, z)
	writeVEC3(bs, nextX, nextY, nextZ)
	bs.WriteUint32(math.Float32bits(radius))
	return bs.Bytes()[:bs.Len()]
}

// ParseSetRaceCheckpoint decodes SetRaceCheckpoint (RPC 38).
func ParseSetRaceCheckpoint(payload []byte) (cpType uint8, x, y, z, nextX, nextY, nextZ, radius float32, err error) {
	bs := raknet.FromBytes(payload)
	if cpType, err = bs.ReadUint8(); err != nil {
		return
	}
	if err = readVEC3(bs, &x, &y, &z); err != nil {
		return
	}
	if err = readVEC3(bs, &nextX, &nextY, &nextZ); err != nil {
		return
	}
	r, err := bs.ReadUint32()
	if err != nil {
		return 0, 0, 0, 0, 0, 0, 0, 0, err
	}
	radius = math.Float32frombits(r)
	return
}

// BuildDisableRaceCheckpoint encodes DisableRaceCheckpoint (RPC 39): no params.
func BuildDisableRaceCheckpoint() []byte { return []byte{} }