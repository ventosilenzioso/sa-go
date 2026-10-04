package raknet

import "math/rand"

// SAMP_PETARDED / OMP_PETARDED are the 16-bit cookie XOR masks used by the
// SA-MP client (0x6969) and open.mp clients (0x6D70). Source: SAMPRakNet.hpp.
const (
	SAMPPetarded  uint16 = 0x6969
	OMPPetarded   uint16 = 0x6D70
	queryMagicLen        = 4
)

// CookieGen produces and validates the per-address connection cookies used by
// the offline handshake (SAMPRakNet::SeedCookie / GetCookie).
type CookieGen struct {
	tables [2][256]uint16
}

// NewCookieGen seeds both cookie tables from rng.
func NewCookieGen(rng *rand.Rand) *CookieGen {
	g := &CookieGen{}
	for i := 0; i < 256; i++ {
		g.tables[0][i] = uint16(rng.Intn(65536))
		g.tables[1][i] = uint16(rng.Intn(65536))
	}
	return g
}

// Cookie returns the cookie for an address given in network byte order
// (first octet at the lowest byte, as stored in a little-endian uint32).
func (g *CookieGen) Cookie(addr uint32) uint16 {
	a0 := byte(addr)
	a1 := byte(addr >> 8)
	a2 := byte(addr >> 16)
	a3 := byte(addr >> 24)
	return (g.tables[0][a0] | g.tables[1][a3]<<8) ^ (uint16(a1)<<8 | uint16(a2))
}

// IsQuery reports whether the raw datagram is a server browser query. Queries
// are the only datagrams that are not client-obfuscated.
func IsQuery(data []byte) bool {
	return len(data) > 10 && data[0] == 'S' && data[1] == 'A' && data[2] == 'M' && data[3] == 'P'
}

// OfflineKind classifies an unconnected datagram (already de-obfuscated).
type OfflineKind int

const (
	// OfflineOther is anything that does not belong to the offline handshake
	// (including first-packet datagrams of a connection already in progress).
	OfflineOther OfflineKind = iota
	// OfflineOpenRequest is ID_OPEN_CONNECTION_REQUEST (3 bytes).
	OfflineOpenRequest
	// OfflinePing is ID_PING with a 4-byte timestamp.
	OfflinePing
	// OfflinePong is ID_PONG.
	OfflinePong
	// OfflineCookieReply is ID_OPEN_CONNECTION_COOKIE (client side).
	OfflineCookieReply
	// OfflineOpenReply is ID_OPEN_CONNECTION_REPLY (client side).
	OfflineOpenReply
)

// OfflineDatagram is the result of classifying an unconnected datagram.
type OfflineDatagram struct {
	Kind OfflineKind
	// Cookie is the 16-bit cookie field of an open connection request or a
	// cookie reply (little-endian byte pair at offset 1).
	Cookie uint16
	// PingTime is the echoed timestamp of ID_PING / ID_PONG.
	PingTime uint32
}

// ParseOffline inspects a plain (de-obfuscated) datagram.
func ParseOffline(data []byte) OfflineDatagram {
	if len(data) == 0 {
		return OfflineDatagram{Kind: OfflineOther}
	}
	switch data[0] {
	case IDOpenConnectionRequest:
		if len(data) == 3 {
			return OfflineDatagram{
				Kind:   OfflineOpenRequest,
				Cookie: uint16(data[1]) | uint16(data[2])<<8,
			}
		}
	case IDPing:
		if len(data) == 5 {
			return OfflineDatagram{
				Kind:     OfflinePing,
				PingTime: le32(data[1:]),
			}
		}
	case IDPong:
		if len(data) >= 5 {
			return OfflineDatagram{Kind: OfflinePong, PingTime: le32(data[1:])}
		}
	case IDOpenConnectionCookie:
		if len(data) == 3 {
			return OfflineDatagram{
				Kind:   OfflineCookieReply,
				Cookie: uint16(data[1]) | uint16(data[2])<<8,
			}
		}
	case IDOpenConnectionReply:
		if len(data) <= 2 {
			return OfflineDatagram{Kind: OfflineOpenReply}
		}
	}
	return OfflineDatagram{Kind: OfflineOther}
}

// BuildOpenCookieReply builds the server's ID_OPEN_CONNECTION_COOKIE response.
func BuildOpenCookieReply(cookie uint16) []byte {
	return []byte{IDOpenConnectionCookie, byte(cookie), byte(cookie >> 8)}
}

// BuildOpenReply builds the server's ID_OPEN_CONNECTION_REPLY response.
func BuildOpenReply() []byte {
	// Second byte is padding; some routers block 1-byte packets.
	return []byte{IDOpenConnectionReply, 0}
}

// BuildOfflinePong builds an ID_PONG response echoing pingTime, followed by
// offlinePingResponse data (empty for SA-MP).
func BuildOfflinePong(pingTime uint32, offlineData []byte) []byte {
	out := make([]byte, 5+len(offlineData))
	out[0] = IDPong
	out[1] = byte(pingTime)
	out[2] = byte(pingTime >> 8)
	out[3] = byte(pingTime >> 16)
	out[4] = byte(pingTime >> 24)
	copy(out[5:], offlineData)
	return out
}

// BuildOpenRequest builds a client-style ID_OPEN_CONNECTION_REQUEST. cookie is
// written verbatim (the client xors the cookie with SAMPPetarded first).
func BuildOpenRequest(cookie uint16) []byte {
	return []byte{IDOpenConnectionRequest, byte(cookie), byte(cookie >> 8)}
}

// BuildOfflinePing builds an unconnected ID_PING.
func BuildOfflinePing(t uint32) []byte {
	return []byte{IDPing, byte(t), byte(t >> 8), byte(t >> 16), byte(t >> 24)}
}

func le32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
