package protocol

import (
	"math"

	"gosamp/internal/raknet"
)

// Vehicle RPCs (server -> client). WorldVehicleAdd (164) streams a vehicle to a
// client; WorldVehicleRemove (165) removes it.
//
// Wire layout (open.mp Shared/NetCode/vehicle.hpp StreamInVehicle, matches
// samp-packet-list WorldVehicleAdd):
//   uint16 vehicleID, uint32 modelID, VEC3 position, float angle,
//   uint8 colour1, uint8 colour2, float health, uint8 interior,
//   uint32 doorDamage, uint32 panelDamage, uint8 lightDamage, uint8 tyreDamage,
//   uint8 siren, uint8 mods[14], uint8 paintjob,
//   uint32 bodyColour1, uint32 bodyColour2

// MaxVehicleModSlots is the number of component slots carried by RPC 164.
const MaxVehicleModSlots = 14

// VehicleSpawn is the full description of a vehicle sent to clients (RPC 164).
type VehicleSpawn struct {
	ModelID                  int32
	X, Y, Z, Angle           float32
	Colour1, Colour2         uint8
	Health                   float32
	Interior                 uint8
	DoorDamage, PanelDamage  uint32
	LightDamage, TyreDamage  uint8
	Siren                    uint8
	Mods                     [MaxVehicleModSlots]uint8
	Paintjob                 uint8
	BodyColour1, BodyColour2 uint32
}

// BuildWorldVehicleAdd builds the WorldVehicleAdd (RPC 164) payload.
func BuildWorldVehicleAdd(vehicleID uint16, d VehicleSpawn) []byte {
	bs := raknet.New()
	bs.WriteUint16(vehicleID)
	bs.WriteUint32(uint32(d.ModelID))
	writeVEC3(bs, d.X, d.Y, d.Z)
	bs.WriteUint32(math.Float32bits(d.Angle))
	bs.WriteUint8(d.Colour1)
	bs.WriteUint8(d.Colour2)
	bs.WriteUint32(math.Float32bits(d.Health))
	bs.WriteUint8(d.Interior)
	bs.WriteUint32(d.DoorDamage)
	bs.WriteUint32(d.PanelDamage)
	bs.WriteUint8(d.LightDamage)
	bs.WriteUint8(d.TyreDamage)
	bs.WriteUint8(d.Siren)
	for _, m := range d.Mods {
		bs.WriteUint8(m)
	}
	bs.WriteUint8(d.Paintjob)
	bs.WriteUint32(d.BodyColour1)
	bs.WriteUint32(d.BodyColour2)
	return bs.Bytes()[:bs.Len()]
}

// ParseWorldVehicleAdd decodes a WorldVehicleAdd (RPC 164) payload.
func ParseWorldVehicleAdd(payload []byte) (uint16, VehicleSpawn, error) {
	var d VehicleSpawn
	bs := raknet.FromBytes(payload)
	vehicleID, err := bs.ReadUint16()
	if err != nil {
		return 0, d, err
	}
	model, err := bs.ReadUint32()
	if err != nil {
		return 0, d, err
	}
	d.ModelID = int32(model)
	if err = readVEC3(bs, &d.X, &d.Y, &d.Z); err != nil {
		return 0, d, err
	}
	angle, err := bs.ReadUint32()
	if err != nil {
		return 0, d, err
	}
	d.Angle = math.Float32frombits(angle)
	if d.Colour1, err = bs.ReadUint8(); err != nil {
		return 0, d, err
	}
	if d.Colour2, err = bs.ReadUint8(); err != nil {
		return 0, d, err
	}
	health, err := bs.ReadUint32()
	if err != nil {
		return 0, d, err
	}
	d.Health = math.Float32frombits(health)
	if d.Interior, err = bs.ReadUint8(); err != nil {
		return 0, d, err
	}
	if d.DoorDamage, err = bs.ReadUint32(); err != nil {
		return 0, d, err
	}
	if d.PanelDamage, err = bs.ReadUint32(); err != nil {
		return 0, d, err
	}
	if d.LightDamage, err = bs.ReadUint8(); err != nil {
		return 0, d, err
	}
	if d.TyreDamage, err = bs.ReadUint8(); err != nil {
		return 0, d, err
	}
	if d.Siren, err = bs.ReadUint8(); err != nil {
		return 0, d, err
	}
	for i := 0; i < MaxVehicleModSlots; i++ {
		if d.Mods[i], err = bs.ReadUint8(); err != nil {
			return 0, d, err
		}
	}
	if d.Paintjob, err = bs.ReadUint8(); err != nil {
		return 0, d, err
	}
	if d.BodyColour1, err = bs.ReadUint32(); err != nil {
		return 0, d, err
	}
	if d.BodyColour2, err = bs.ReadUint32(); err != nil {
		return 0, d, err
	}
	return vehicleID, d, nil
}

// BuildWorldVehicleRemove builds the WorldVehicleRemove (RPC 165) payload.
func BuildWorldVehicleRemove(vehicleID uint16) []byte {
	bs := raknet.New()
	bs.WriteUint16(vehicleID)
	return bs.Bytes()[:bs.Len()]
}

// --- Vehicle damage / destruction (server <-> client) ---

// VehicleDamageStatus is the damage bitmask state of a vehicle: panels and
// doors are 32-bit masks, lights and tyres are 8-bit masks (open.mp
// VEHICLE_PANEL_STATUS / DOOR / LIGHT / TYRE).
type VehicleDamageStatus struct {
	Panels uint32
	Doors  uint32
	Lights uint8
	Tyres  uint8
}

// BuildUpdateVehicleDamageStatus builds the server -> client UpdateVehicleDamage
// Status (RPC 106) payload: uint16 vehicleID, uint32 panels, uint32 doors,
// uint8 lights, uint8 tyres.
func BuildUpdateVehicleDamageStatus(vehicleID uint16, d VehicleDamageStatus) []byte {
	bs := raknet.New()
	bs.WriteUint16(vehicleID)
	bs.WriteUint32(d.Panels)
	bs.WriteUint32(d.Doors)
	bs.WriteUint8(d.Lights)
	bs.WriteUint8(d.Tyres)
	return bs.Bytes()[:bs.Len()]
}

// ParseUpdateVehicleDamageStatus decodes a client -> server UpdateVehicleDamage
// Status (RPC 106) payload. Same layout as the outgoing form.
func ParseUpdateVehicleDamageStatus(payload []byte) (vehicleID uint16, d VehicleDamageStatus, err error) {
	bs := raknet.FromBytes(payload)
	if vehicleID, err = bs.ReadUint16(); err != nil {
		return
	}
	if d.Panels, err = bs.ReadUint32(); err != nil {
		return
	}
	if d.Doors, err = bs.ReadUint32(); err != nil {
		return
	}
	if d.Lights, err = bs.ReadUint8(); err != nil {
		return
	}
	d.Tyres, err = bs.ReadUint8()
	return
}

// BuildSetVehicleTireStatus builds the server -> client SetVehicleTireStatus
// (RPC 98) payload: uint16 vehicleID, uint8 tyres bitmask.
func BuildSetVehicleTireStatus(vehicleID uint16, tyres uint8) []byte {
	bs := raknet.New()
	bs.WriteUint16(vehicleID)
	bs.WriteUint8(tyres)
	return bs.Bytes()[:bs.Len()]
}

// ParseSetVehicleTireStatus decodes SetVehicleTireStatus (RPC 98).
func ParseSetVehicleTireStatus(payload []byte) (vehicleID uint16, tyres uint8, err error) {
	bs := raknet.FromBytes(payload)
	if vehicleID, err = bs.ReadUint16(); err != nil {
		return
	}
	tyres, err = bs.ReadUint8()
	return
}

// BuildVehicleDestroyed builds the server -> client VehicleDestroyed (RPC 136)
// payload: uint16 vehicleID. SA-MP has no killer parameter on the wire; the
// gamemode supplies killerid from its own bookkeeping.
func BuildVehicleDestroyed(vehicleID uint16) []byte {
	bs := raknet.New()
	bs.WriteUint16(vehicleID)
	return bs.Bytes()[:bs.Len()]
}

// ParseVehicleDestroyed decodes VehicleDestroyed (RPC 136).
func ParseVehicleDestroyed(payload []byte) (vehicleID uint16, err error) {
	bs := raknet.FromBytes(payload)
	return bs.ReadUint16()
}

// BuildSetVehicleHealth builds the server -> client SetVehicleHealth (RPC 147)
// payload: uint16 vehicleID, float health.
func BuildSetVehicleHealth(vehicleID uint16, health float32) []byte {
	bs := raknet.New()
	bs.WriteUint16(vehicleID)
	bs.WriteUint32(math.Float32bits(health))
	return bs.Bytes()[:bs.Len()]
}

// ParseSetVehicleHealth decodes SetVehicleHealth (RPC 147).
func ParseSetVehicleHealth(payload []byte) (vehicleID uint16, health float32, err error) {
	bs := raknet.FromBytes(payload)
	if vehicleID, err = bs.ReadUint16(); err != nil {
		return
	}
	h, err := bs.ReadUint32()
	if err != nil {
		return 0, 0, err
	}
	health = math.Float32frombits(h)
	return
}
