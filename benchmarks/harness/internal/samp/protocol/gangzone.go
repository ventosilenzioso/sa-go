package protocol

import (
	"math"

	"gosamp/internal/raknet"
)

// Gang zone RPCs. A gang zone is a rectangular map area shown with a coloured
// overlay. The colour is part of the SHOW packet (RPC 108), not a separate
// create step, so the engine stores the zone rectangle at create time and the
// colour is supplied when showing. MAX_GANG_ZONES = 1024 in SA-MP.
//
// Wire layouts (open.mp Shared/NetCode/gangzone.hpp):
//   ShowGangZone (108):  int16 zoneID, VEC2 min, VEC2 max, uint32 colour(ABGR)
//   HideGangZone (120):  int16 zoneID
//   FlashGangZone (121): int16 zoneID, uint32 colour(ABGR)
//   StopFlashGangZone (85): int16 zoneID

// MaxGangZones is the SA-MP gang zone pool size.
const MaxGangZones = 1024

func writeVEC2(bs *raknet.BitStream, x, y float32) {
	bs.WriteUint32(math.Float32bits(x))
	bs.WriteUint32(math.Float32bits(y))
}

func readVEC2(bs *raknet.BitStream, x, y *float32) error {
	v, err := bs.ReadUint32()
	if err != nil {
		return err
	}
	*x = math.Float32frombits(v)
	v, err = bs.ReadUint32()
	if err != nil {
		return err
	}
	*y = math.Float32frombits(v)
	return nil
}

// BuildShowGangZone encodes ShowGangZone (RPC 108). colour is RGBA uint32 in
// Lua convention and is converted to ABGR on the wire.
func BuildShowGangZone(zoneID int16, minX, minY, maxX, maxY float32, colour uint32) []byte {
	bs := raknet.New()
	bs.WriteUint16(uint16(zoneID))
	writeVEC2(bs, minX, minY)
	writeVEC2(bs, maxX, maxY)
	bs.WriteUint32(abgr(colour))
	return bs.Bytes()[:bs.Len()]
}

// ParseShowGangZone decodes ShowGangZone (RPC 108). colour is returned as RGBA.
func ParseShowGangZone(payload []byte) (zoneID int16, minX, minY, maxX, maxY float32, colour uint32, err error) {
	bs := raknet.FromBytes(payload)
	id, err := bs.ReadUint16()
	if err != nil {
		return
	}
	zoneID = int16(id)
	if err = readVEC2(bs, &minX, &minY); err != nil {
		return
	}
	if err = readVEC2(bs, &maxX, &maxY); err != nil {
		return
	}
	c, err := bs.ReadUint32()
	if err != nil {
		return
	}
	colour = abgr(c) // ABGR -> RGBA (abgr is its own inverse)
	return
}

// BuildHideGangZone encodes HideGangZone (RPC 120).
func BuildHideGangZone(zoneID int16) []byte {
	bs := raknet.New()
	bs.WriteUint16(uint16(zoneID))
	return bs.Bytes()[:bs.Len()]
}

// BuildFlashGangZone encodes FlashGangZone (RPC 121). colour is RGBA -> ABGR.
func BuildFlashGangZone(zoneID int16, colour uint32) []byte {
	bs := raknet.New()
	bs.WriteUint16(uint16(zoneID))
	bs.WriteUint32(abgr(colour))
	return bs.Bytes()[:bs.Len()]
}

// ParseFlashGangZone decodes FlashGangZone (RPC 121). colour returned as RGBA.
func ParseFlashGangZone(payload []byte) (zoneID int16, colour uint32, err error) {
	bs := raknet.FromBytes(payload)
	id, err := bs.ReadUint16()
	if err != nil {
		return
	}
	zoneID = int16(id)
	c, err := bs.ReadUint32()
	if err != nil {
		return
	}
	colour = abgr(c)
	return
}

// BuildStopFlashGangZone encodes StopFlashGangZone (RPC 85).
func BuildStopFlashGangZone(zoneID int16) []byte {
	bs := raknet.New()
	bs.WriteUint16(uint16(zoneID))
	return bs.Bytes()[:bs.Len()]
}