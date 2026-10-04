package protocol

import (
	"math"

	"gosamp/internal/raknet"
)

// Vehicle occupancy RPCs. SA-MP vehicle entry/exit is client-authoritative: the
// client sends EnterVehicle (26)/ExitVehicle (154) and then switches sync
// streams. The server records occupancy so it can relay to other clients and
// inform the gamemode. A server-side forced exit uses RemovePlayerFromVehicle
// (71); the client applies the exit physics itself.
//
// Wire layouts:
//   PutPlayerInVehicle (70, server->client):    uint16 vehicleID, uint8 seatID
//   RemovePlayerFromVehicle (71, server->client): (no parameters)
//   SetPlayerVelocity (90, server->client):     float x, y, z
//   EnterVehicle (26, client->server):          uint16 vehicleID, uint8 passenger
//   ExitVehicle (154, client->server):          uint16 vehicleID

// BuildPutPlayerInVehicle encodes PutPlayerInVehicle (RPC 70).
func BuildPutPlayerInVehicle(vehicleID uint16, seatID uint8) []byte {
	bs := raknet.New()
	bs.WriteUint16(vehicleID)
	bs.WriteUint8(seatID)
	return bs.Bytes()[:bs.Len()]
}

// ParsePutPlayerInVehicle decodes PutPlayerInVehicle (RPC 70).
func ParsePutPlayerInVehicle(payload []byte) (vehicleID uint16, seatID uint8, err error) {
	bs := raknet.FromBytes(payload)
	if vehicleID, err = bs.ReadUint16(); err != nil {
		return
	}
	seatID, err = bs.ReadUint8()
	return
}

// BuildRemovePlayerFromVehicle encodes RemovePlayerFromVehicle (RPC 71): no
// parameters.
func BuildRemovePlayerFromVehicle() []byte { return []byte{} }

// BuildSetPlayerVelocity encodes SetPlayerVelocity (RPC 90): float x, y, z.
func BuildSetPlayerVelocity(x, y, z float32) []byte {
	bs := raknet.New()
	bs.WriteUint32(math.Float32bits(x))
	bs.WriteUint32(math.Float32bits(y))
	bs.WriteUint32(math.Float32bits(z))
	return bs.Bytes()[:bs.Len()]
}

// ParseSetPlayerVelocity decodes SetPlayerVelocity (RPC 90).
func ParseSetPlayerVelocity(payload []byte) (x, y, z float32, err error) {
	bs := raknet.FromBytes(payload)
	if err = readVEC3(bs, &x, &y, &z); err != nil {
		return
	}
	return
}

// EnterVehicle is the decoded client EnterVehicle (RPC 26) report.
type EnterVehicle struct {
	VehicleID uint16
	Passenger bool
}

// ParseEnterVehicle decodes EnterVehicle (RPC 26).
func ParseEnterVehicle(payload []byte) (EnterVehicle, error) {
	var e EnterVehicle
	bs := raknet.FromBytes(payload)
	var err error
	if e.VehicleID, err = bs.ReadUint16(); err != nil {
		return e, err
	}
	p, err := bs.ReadUint8()
	if err != nil {
		return e, err
	}
	e.Passenger = p != 0
	return e, nil
}

// ParseExitVehicle decodes ExitVehicle (RPC 154): uint16 vehicleID.
func ParseExitVehicle(payload []byte) (uint16, error) {
	return raknet.FromBytes(payload).ReadUint16()
}

// BuildSetPlayerPos encodes SetPlayerPos (RPC 12): float x, y, z.
func BuildSetPlayerPos(x, y, z float32) []byte {
	bs := raknet.New()
	bs.WriteUint32(math.Float32bits(x))
	bs.WriteUint32(math.Float32bits(y))
	bs.WriteUint32(math.Float32bits(z))
	return bs.Bytes()[:bs.Len()]
}

// ParseSetPlayerPos decodes SetPlayerPos (RPC 12).
func ParseSetPlayerPos(payload []byte) (x, y, z float32, err error) {
	bs := raknet.FromBytes(payload)
	err = readVEC3(bs, &x, &y, &z)
	return
}

// BuildSetPlayerSkin encodes SetPlayerSkin (RPC 153): uint32 playerID, uint32 skinID.
func BuildSetPlayerSkin(playerID, skinID int32) []byte {
	bs := raknet.New()
	bs.WriteUint32(uint32(playerID))
	bs.WriteUint32(uint32(skinID))
	return bs.Bytes()[:bs.Len()]
}

// ParseSetPlayerSkin decodes SetPlayerSkin (RPC 153).
func ParseSetPlayerSkin(payload []byte) (playerID, skinID int32, err error) {
	bs := raknet.FromBytes(payload)
	p, err := bs.ReadUint32()
	if err != nil {
		return 0, 0, err
	}
	s, err := bs.ReadUint32()
	return int32(p), int32(s), err
}

// BuildSetPlayerFacingAngle encodes SetPlayerFacingAngle (RPC 19): float angle.
func BuildSetPlayerFacingAngle(angle float32) []byte {
	bs := raknet.New()
	bs.WriteUint32(math.Float32bits(angle))
	return bs.Bytes()[:bs.Len()]
}

// ParseSetPlayerFacingAngle decodes SetPlayerFacingAngle (RPC 19).
func ParseSetPlayerFacingAngle(payload []byte) (angle float32, err error) {
	bs := raknet.FromBytes(payload)
	a, err := bs.ReadUint32()
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(a), nil
}