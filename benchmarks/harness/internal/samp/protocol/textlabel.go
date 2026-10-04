package protocol

import (
	"math"

	"gosamp/internal/raknet"
)

// 3D text label RPCs. A 3D text label is floating text in the world. It may be
// attached to a player or vehicle; the CLIENT then keeps it following the
// target automatically, so the server never needs to update its position.
// MAX_3DTEXT_GLOBAL = 1024 in SA-MP.
//
// Wire layout (open.mp Shared/NetCode/textlabel.hpp PlayerShowTextLabel):
//   uint16 labelID, uint32 colour(RGBA), VEC3 position, float drawDistance,
//   uint8 testLOS, uint16 attachedPlayer, uint16 attachedVehicle,
//   compressed string text
// attachedPlayer/attachedVehicle use 0xFFFF for "no target".
//
//   PlayerHideTextLabel (58): uint16 labelID

// MaxTextLabels is the SA-MP global 3D text label pool size.
const MaxTextLabels = 1024

// NoAttach is the SA-MP sentinel for "no player/vehicle target" (0xFFFF).
const NoAttach = 0xFFFF

// BuildCreate3DTextLabel encodes Create3DTextLabel (RPC 36). colour is RGBA.
// attachPlayer/attachVehicle are 0xFFFF when unused. Text uses the same
// Huffman-compressed CSTRING encoding as ShowDialog's body.
func BuildCreate3DTextLabel(labelID uint16, colour uint32, x, y, z, drawDistance float32, testLOS bool, attachPlayer, attachVehicle uint16, text string) []byte {
	bs := raknet.New()
	bs.WriteUint16(labelID)
	bs.WriteUint32(colour)
	writeVEC3(bs, x, y, z)
	bs.WriteUint32(math.Float32bits(drawDistance))
	if testLOS {
		bs.WriteUint8(1)
	} else {
		bs.WriteUint8(0)
	}
	bs.WriteUint16(attachPlayer)
	bs.WriteUint16(attachVehicle)
	bs.WriteCompressedStr(UTF8ToCP1252(text))
	return bs.Bytes()[:bs.Len()]
}

// ParseCreate3DTextLabel decodes Create3DTextLabel (RPC 36).
func ParseCreate3DTextLabel(payload []byte) (labelID uint16, colour uint32, x, y, z, drawDistance float32, testLOS bool, attachPlayer, attachVehicle uint16, text string, err error) {
	bs := raknet.FromBytes(payload)
	if labelID, err = bs.ReadUint16(); err != nil {
		return
	}
	if colour, err = bs.ReadUint32(); err != nil {
		return
	}
	if err = readVEC3(bs, &x, &y, &z); err != nil {
		return
	}
	d, err := bs.ReadUint32()
	if err != nil {
		return
	}
	drawDistance = math.Float32frombits(d)
	los, err := bs.ReadUint8()
	if err != nil {
		return
	}
	testLOS = los != 0
	if attachPlayer, err = bs.ReadUint16(); err != nil {
		return
	}
	if attachVehicle, err = bs.ReadUint16(); err != nil {
		return
	}
	raw, err := bs.ReadCompressedStr()
	if err != nil {
		return
	}
	text = CP1252ToUTF8(raw)
	return
}

// BuildDelete3DTextLabel encodes Delete3DTextLabel (RPC 58).
func BuildDelete3DTextLabel(labelID uint16) []byte {
	bs := raknet.New()
	bs.WriteUint16(labelID)
	return bs.Bytes()[:bs.Len()]
}

// ParseDelete3DTextLabel decodes Delete3DTextLabel (RPC 58).
func ParseDelete3DTextLabel(payload []byte) (uint16, error) {
	return raknet.FromBytes(payload).ReadUint16()
}