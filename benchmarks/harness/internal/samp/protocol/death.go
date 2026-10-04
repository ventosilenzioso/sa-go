package protocol

import (
	"gosamp/internal/raknet"
)

// DeathNotification is the client -> server death report (RPC 53):
// uint8 reason, uint16 killerID.
type DeathNotification struct {
	Reason   uint8
	KillerID uint16
}

// ParseDeathNotification decodes RPC 53. SA-MP clients have sent both a 3-byte
// (uint8 reason + uint16 killer) and, in some builds, a 1-byte form; this
// accepts the 3-byte canonical layout and tolerates a short payload.
func ParseDeathNotification(payload []byte) (DeathNotification, error) {
	var d DeathNotification
	bs := raknet.FromBytes(payload)
	var err error
	if d.Reason, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	// Killer is optional in truncated payloads.
	if k, e := bs.ReadUint16(); e == nil {
		d.KillerID = k
	}
	return d, nil
}

// BuildWorldPlayerDeath builds the server -> client WorldPlayerDeath (RPC 166)
// payload: uint16 playerID.
func BuildWorldPlayerDeath(playerID uint16) []byte {
	bs := raknet.New()
	bs.WriteUint16(playerID)
	return bs.Bytes()[:bs.Len()]
}