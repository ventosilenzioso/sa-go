package protocol

import (
	"gosamp/internal/raknet"
)

// Map icon (radar blip) RPCs, per-player. Up to 100 icons per player (id 0..99).
//
// Wire layout (samp-packet-list RPC List):
//   SetMapIcon (56):    uint8 iconID, VEC3 position, uint8 markerType,
//                       uint32 colour(RGBA), uint8 style
//   RemoveMapIcon (144): uint8 iconID
//
// MAPICON styles: 0 = local (shows in local area), 1 = global (always),
// 2 = local + checkpoint marker, 3 = global + checkpoint marker.

// MaxMapIcons is the per-player map icon limit.
const MaxMapIcons = 100

// Map icon styles.
const (
	MapIconLocal            = 0
	MapIconGlobal           = 1
	MapIconLocalCheckpoint  = 2
	MapIconGlobalCheckpoint = 3
)

// BuildSetMapIcon encodes SetMapIcon (RPC 56).
func BuildSetMapIcon(iconID uint8, x, y, z float32, markerType uint8, colour uint32, style uint8) []byte {
	bs := raknet.New()
	bs.WriteUint8(iconID)
	writeVEC3(bs, x, y, z)
	bs.WriteUint8(markerType)
	bs.WriteUint32(colour)
	bs.WriteUint8(style)
	return bs.Bytes()[:bs.Len()]
}

// ParseSetMapIcon decodes SetMapIcon (RPC 56).
func ParseSetMapIcon(payload []byte) (iconID uint8, x, y, z float32, markerType uint8, colour uint32, style uint8, err error) {
	bs := raknet.FromBytes(payload)
	if iconID, err = bs.ReadUint8(); err != nil {
		return
	}
	if err = readVEC3(bs, &x, &y, &z); err != nil {
		return
	}
	if markerType, err = bs.ReadUint8(); err != nil {
		return
	}
	if colour, err = bs.ReadUint32(); err != nil {
		return
	}
	style, err = bs.ReadUint8()
	return
}

// BuildRemoveMapIcon encodes RemoveMapIcon (RPC 144).
func BuildRemoveMapIcon(iconID uint8) []byte {
	return []byte{iconID}
}

// ParseRemoveMapIcon decodes RemoveMapIcon (RPC 144).
func ParseRemoveMapIcon(payload []byte) (uint8, error) {
	return raknet.FromBytes(payload).ReadUint8()
}
