package protocol

import (
	"gosamp/internal/raknet"
)

// SA-MP weapon slot constants (0..12).
const (
	SlotUnarmed      = 0
	SlotMelee        = 1
	SlotHandguns     = 2
	SlotShotguns     = 3
	SlotSubMachine   = 4
	SlotAssault      = 5
	SlotRifles       = 6
	SlotHeavy        = 7
	SlotProjectiles  = 8
	SlotSpecial1     = 9  // Spray Paint, Fire Extinguisher, Camera
	SlotGifts        = 10 // Dildos, Vibrators, Flowers, Cane
	SlotSpecial2     = 11 // Nightvision, Thermal, Parachute
	SlotDetonator    = 12 // Bomb / Detonator
	TotalWeaponSlots = 13
)

// Weapon IDs 0..46 and virtual damage reasons 49..54.
const (
	WeaponFist             = 0
	WeaponBrassKnuckles    = 1
	WeaponGolfClub         = 2
	WeaponNiteStick        = 3
	WeaponKnife            = 4
	WeaponBaseballBat      = 5
	WeaponShovel           = 6
	WeaponPoolCue          = 7
	WeaponKatana           = 8
	WeaponChainsaw         = 9
	WeaponDildo            = 10
	WeaponDildo2           = 11
	WeaponVibrator         = 12
	WeaponVibrator2        = 13
	WeaponFlowers          = 14
	WeaponCane             = 15
	WeaponGrenade          = 16
	WeaponTeargas          = 17
	WeaponMolotov          = 18
	WeaponColt45           = 22
	WeaponSilenced         = 23
	WeaponDeagle           = 24
	WeaponShotgun          = 25
	WeaponSawnoff          = 26
	WeaponCombatShotgun    = 27
	WeaponMicroUzi         = 28
	WeaponMP5              = 29
	WeaponAK47             = 30
	WeaponM4               = 31
	WeaponTec9             = 32
	WeaponCountryRifle     = 33
	WeaponSniper           = 34
	WeaponRPG              = 35
	WeaponHeatseeker       = 36
	WeaponFlamethrower     = 37
	WeaponMinigun          = 38
	WeaponSatchel          = 39
	WeaponDetonator        = 40
	WeaponSpraycan         = 41
	WeaponFireExtinguisher = 42
	WeaponCamera           = 43
	WeaponNightvision      = 44
	WeaponThermal          = 45
	WeaponParachute        = 46

	// Non-physical virtual reasons for damage/death.
	WeaponVehicle          = 49
	WeaponHelicopterBlades = 50
	WeaponExplosion        = 51
	WeaponDrowned          = 53
	WeaponSplat            = 54
)

// WeaponInfo holds metadata for one weapon: ID, canonical name, slot, clip size.
type WeaponInfo struct {
	ID       int
	Name     string
	Slot     int // 0..12, or -1 for non-physical/virtual
	ClipSize int
}

// WeaponTable is the authoritative reference of all SA-MP weapons and damage reasons.
var WeaponTable = map[int]WeaponInfo{
	WeaponFist:             {ID: 0, Name: "Fist", Slot: SlotUnarmed, ClipSize: 0},
	WeaponBrassKnuckles:    {ID: 1, Name: "Brass Knuckles", Slot: SlotUnarmed, ClipSize: 0},
	WeaponGolfClub:         {ID: 2, Name: "Golf Club", Slot: SlotMelee, ClipSize: 0},
	WeaponNiteStick:        {ID: 3, Name: "Nite Stick", Slot: SlotMelee, ClipSize: 0},
	WeaponKnife:            {ID: 4, Name: "Knife", Slot: SlotMelee, ClipSize: 0},
	WeaponBaseballBat:      {ID: 5, Name: "Baseball Bat", Slot: SlotMelee, ClipSize: 0},
	WeaponShovel:           {ID: 6, Name: "Shovel", Slot: SlotMelee, ClipSize: 0},
	WeaponPoolCue:          {ID: 7, Name: "Pool Cue", Slot: SlotMelee, ClipSize: 0},
	WeaponKatana:           {ID: 8, Name: "Katana", Slot: SlotMelee, ClipSize: 0},
	WeaponChainsaw:         {ID: 9, Name: "Chainsaw", Slot: SlotMelee, ClipSize: 0},
	WeaponDildo:            {ID: 10, Name: "Dildo", Slot: SlotGifts, ClipSize: 0},
	WeaponDildo2:           {ID: 11, Name: "Dildo", Slot: SlotGifts, ClipSize: 0},
	WeaponVibrator:         {ID: 12, Name: "Vibrator", Slot: SlotGifts, ClipSize: 0},
	WeaponVibrator2:        {ID: 13, Name: "Vibrator", Slot: SlotGifts, ClipSize: 0},
	WeaponFlowers:          {ID: 14, Name: "Flowers", Slot: SlotGifts, ClipSize: 0},
	WeaponCane:             {ID: 15, Name: "Cane", Slot: SlotGifts, ClipSize: 0},
	WeaponGrenade:          {ID: 16, Name: "Grenade", Slot: SlotProjectiles, ClipSize: 1},
	WeaponTeargas:          {ID: 17, Name: "Teargas", Slot: SlotProjectiles, ClipSize: 1},
	WeaponMolotov:          {ID: 18, Name: "Molotov Cocktail", Slot: SlotProjectiles, ClipSize: 1},
	WeaponColt45:           {ID: 22, Name: "Colt 45", Slot: SlotHandguns, ClipSize: 17},
	WeaponSilenced:         {ID: 23, Name: "Silenced Pistol", Slot: SlotHandguns, ClipSize: 17},
	WeaponDeagle:           {ID: 24, Name: "Desert Eagle", Slot: SlotHandguns, ClipSize: 7},
	WeaponShotgun:          {ID: 25, Name: "Shotgun", Slot: SlotShotguns, ClipSize: 1},
	WeaponSawnoff:          {ID: 26, Name: "Sawn-Off Shotgun", Slot: SlotShotguns, ClipSize: 2},
	WeaponCombatShotgun:    {ID: 27, Name: "Combat Shotgun", Slot: SlotShotguns, ClipSize: 7},
	WeaponMicroUzi:         {ID: 28, Name: "Micro UZI", Slot: SlotSubMachine, ClipSize: 0},
	WeaponMP5:              {ID: 29, Name: "MP5", Slot: SlotSubMachine, ClipSize: 30},
	WeaponAK47:             {ID: 30, Name: "AK47", Slot: SlotAssault, ClipSize: 30},
	WeaponM4:               {ID: 31, Name: "M4", Slot: SlotAssault, ClipSize: 30},
	WeaponTec9:             {ID: 32, Name: "Tec9", Slot: SlotSubMachine, ClipSize: 0},
	WeaponCountryRifle:     {ID: 33, Name: "Country Rifle", Slot: SlotRifles, ClipSize: 1},
	WeaponSniper:           {ID: 34, Name: "Sniper Rifle", Slot: SlotRifles, ClipSize: 1},
	WeaponRPG:              {ID: 35, Name: "RPG", Slot: SlotHeavy, ClipSize: 1},
	WeaponHeatseeker:       {ID: 36, Name: "Missile Launcher", Slot: SlotHeavy, ClipSize: 1},
	WeaponFlamethrower:     {ID: 37, Name: "Flamethrower", Slot: SlotHeavy, ClipSize: 0},
	WeaponMinigun:          {ID: 38, Name: "Minigun", Slot: SlotHeavy, ClipSize: 0},
	WeaponSatchel:          {ID: 39, Name: "Satchel Charge", Slot: SlotProjectiles, ClipSize: 1},
	WeaponDetonator:        {ID: 40, Name: "Bomb/Detonator", Slot: SlotDetonator, ClipSize: 0},
	WeaponSpraycan:         {ID: 41, Name: "Spray Paint", Slot: SlotSpecial1, ClipSize: 0},
	WeaponFireExtinguisher: {ID: 42, Name: "Fire Extinguisher", Slot: SlotSpecial1, ClipSize: 0},
	WeaponCamera:           {ID: 43, Name: "Camera", Slot: SlotSpecial1, ClipSize: 0},
	WeaponNightvision:      {ID: 44, Name: "Nightvision Goggles", Slot: SlotSpecial2, ClipSize: 0},
	WeaponThermal:          {ID: 45, Name: "Thermal Goggles", Slot: SlotSpecial2, ClipSize: 0},
	WeaponParachute:        {ID: 46, Name: "Parachute", Slot: SlotSpecial2, ClipSize: 0},

	// Virtual damage reasons.
	WeaponVehicle:          {ID: 49, Name: "Vehicle", Slot: -1, ClipSize: 0},
	WeaponHelicopterBlades: {ID: 50, Name: "Helicopter Blades", Slot: -1, ClipSize: 0},
	WeaponExplosion:        {ID: 51, Name: "Explosion", Slot: -1, ClipSize: 0},
	WeaponDrowned:          {ID: 53, Name: "Drowned", Slot: -1, ClipSize: 0},
	WeaponSplat:            {ID: 54, Name: "Splat", Slot: -1, ClipSize: 0},
}

// GetWeaponInfo returns the WeaponInfo for a given weapon ID if known.
func GetWeaponInfo(id int) (WeaponInfo, bool) {
	info, ok := WeaponTable[id]
	return info, ok
}

// GetWeaponName returns the canonical name of a weapon ID, or "" if unknown.
func GetWeaponName(id int) string {
	if info, ok := WeaponTable[id]; ok {
		return info.Name
	}
	return ""
}

// GetWeaponSlot returns the slot (0..12) for a physical weapon, or (-1, false).
func GetWeaponSlot(id int) (int, bool) {
	if info, ok := WeaponTable[id]; ok && info.Slot >= 0 {
		return info.Slot, true
	}
	return -1, false
}

// IsPhysicalWeapon returns true if id is a physical weapon (0..46 with a valid slot).
func IsPhysicalWeapon(id int) bool {
	info, ok := WeaponTable[id]
	return ok && info.Slot >= 0
}

// WeaponSlotUpdate is one entry of WeaponsUpdate (RPC 204).
type WeaponSlotUpdate struct {
	Slot   uint8
	Weapon uint8
	Ammo   uint16
}

// BuildWeaponsUpdateRPC encodes WeaponsUpdate (RPC 204):
// count (uint8), followed by count * [slot (uint8), weapon (uint8), ammo (uint16)].
func BuildWeaponsUpdateRPC(slots []WeaponSlotUpdate) []byte {
	bs := raknet.New()
	bs.WriteUint8(uint8(len(slots)))
	for _, s := range slots {
		bs.WriteUint8(s.Slot)
		bs.WriteUint8(s.Weapon)
		bs.WriteUint16(s.Ammo)
	}
	return bs.Bytes()[:bs.Len()]
}

// ParseWeaponsUpdateRPC decodes WeaponsUpdate (RPC 204).
func ParseWeaponsUpdateRPC(payload []byte) ([]WeaponSlotUpdate, error) {
	bs := raknet.FromBytes(payload)
	count, err := bs.ReadUint8()
	if err != nil {
		return nil, err
	}
	slots := make([]WeaponSlotUpdate, 0, count)
	for i := 0; i < int(count); i++ {
		slot, err := bs.ReadUint8()
		if err != nil {
			return nil, err
		}
		weapon, err := bs.ReadUint8()
		if err != nil {
			return nil, err
		}
		ammo, err := bs.ReadUint16()
		if err != nil {
			return nil, err
		}
		slots = append(slots, WeaponSlotUpdate{Slot: slot, Weapon: weapon, Ammo: ammo})
	}
	return slots, nil
}

// BuildGivePlayerWeapon encodes standard SA-MP RPC 22: UINT32 dWeaponID, UINT32 dBullets.
func BuildGivePlayerWeapon(weaponID int, ammo int) []byte {
	bs := raknet.New()
	bs.WriteUint32(uint32(weaponID))
	bs.WriteUint32(uint32(ammo))
	return bs.Bytes()[:bs.Len()]
}

// ParseGivePlayerWeapon decodes standard SA-MP RPC 22.
func ParseGivePlayerWeapon(payload []byte) (weaponID int, ammo int, err error) {
	bs := raknet.FromBytes(payload)
	w, err := bs.ReadUint32()
	if err != nil {
		return 0, 0, err
	}
	a, err := bs.ReadUint32()
	return int(w), int(a), err
}

// BuildResetWeapons encodes standard SA-MP RPC 21 (no parameters).
func BuildResetWeapons() []byte {
	return []byte{}
}

// BuildSetWeaponAmmo encodes standard SA-MP RPC 145: UINT8 bWeaponID, UINT16 wAmmo.
func BuildSetWeaponAmmo(weaponID int, ammo int) []byte {
	bs := raknet.New()
	bs.WriteUint8(uint8(weaponID))
	bs.WriteUint16(uint16(ammo))
	return bs.Bytes()[:bs.Len()]
}

// ParseSetWeaponAmmo decodes standard SA-MP RPC 145.
func ParseSetWeaponAmmo(payload []byte) (weaponID int, ammo int, err error) {
	bs := raknet.FromBytes(payload)
	w, err := bs.ReadUint8()
	if err != nil {
		return 0, 0, err
	}
	a, err := bs.ReadUint16()
	return int(w), int(a), err
}
