package protocol

import (
	"math"

	"gosamp/internal/raknet"
)

// Cinematic camera RPCs (server -> client).
//
// Wire layouts (samp-packet-list RPC List):
//   SetCameraPos (157):      float lookposX, float lookposY, float lookposZ
//   SetCameraLookAt (158):   float lookposX, float lookposY, float lookposZ,
//                            uint8 cutType
//   InterpolateCamera (82):  bool bPosSet, float fromPosX/Y/Z, float toPosX/Y/Z,
//                            uint32 time, uint8 cutType
//   SetCameraBehind (162):   (no parameters)
//
// InterpolateCamera is the SA-MP cutscene primitive: it moves the camera from
// one point to another over `time` milliseconds. bPosSet selects whether the
// from/to vectors are the camera POSITION (true) or the LOOK-AT target (false).

// BuildSetCameraPos encodes SetCameraPos (RPC 157).
func BuildSetCameraPos(x, y, z float32) []byte {
	bs := raknet.New()
	bs.WriteUint32(math.Float32bits(x))
	bs.WriteUint32(math.Float32bits(y))
	bs.WriteUint32(math.Float32bits(z))
	return bs.Bytes()[:bs.Len()]
}

// ParseSetCameraPos decodes SetCameraPos (RPC 157).
func ParseSetCameraPos(payload []byte) (x, y, z float32, err error) {
	bs := raknet.FromBytes(payload)
	err = readVEC3(bs, &x, &y, &z)
	return
}

// BuildSetCameraLookAt encodes SetCameraLookAt (RPC 158).
func BuildSetCameraLookAt(x, y, z float32, cutType uint8) []byte {
	bs := raknet.New()
	bs.WriteUint32(math.Float32bits(x))
	bs.WriteUint32(math.Float32bits(y))
	bs.WriteUint32(math.Float32bits(z))
	bs.WriteUint8(cutType)
	return bs.Bytes()[:bs.Len()]
}

// ParseSetCameraLookAt decodes SetCameraLookAt (RPC 158).
func ParseSetCameraLookAt(payload []byte) (x, y, z float32, cutType uint8, err error) {
	bs := raknet.FromBytes(payload)
	if err = readVEC3(bs, &x, &y, &z); err != nil {
		return
	}
	cutType, err = bs.ReadUint8()
	return
}

// BuildInterpolateCamera encodes InterpolateCamera (RPC 82).
func BuildInterpolateCamera(posSet bool, fromX, fromY, fromZ, toX, toY, toZ float32, time uint32, cutType uint8) []byte {
	bs := raknet.New()
	bs.WriteBool(posSet)
	bs.WriteUint32(math.Float32bits(fromX))
	bs.WriteUint32(math.Float32bits(fromY))
	bs.WriteUint32(math.Float32bits(fromZ))
	bs.WriteUint32(math.Float32bits(toX))
	bs.WriteUint32(math.Float32bits(toY))
	bs.WriteUint32(math.Float32bits(toZ))
	bs.WriteUint32(time)
	bs.WriteUint8(cutType)
	return bs.Bytes()[:bs.Len()]
}

// ParseInterpolateCamera decodes InterpolateCamera (RPC 82).
func ParseInterpolateCamera(payload []byte) (posSet bool, fromX, fromY, fromZ, toX, toY, toZ float32, time uint32, cutType uint8, err error) {
	bs := raknet.FromBytes(payload)
	if posSet, err = bs.ReadBool(); err != nil {
		return
	}
	if err = readVEC3(bs, &fromX, &fromY, &fromZ); err != nil {
		return
	}
	if err = readVEC3(bs, &toX, &toY, &toZ); err != nil {
		return
	}
	if time, err = bs.ReadUint32(); err != nil {
		return
	}
	cutType, err = bs.ReadUint8()
	return
}

// BuildSetCameraBehind encodes SetCameraBehind (RPC 162): no parameters.
func BuildSetCameraBehind() []byte { return []byte{} }