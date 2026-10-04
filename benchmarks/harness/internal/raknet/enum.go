package raknet

// Legacy RakNet packet identifiers (RAKNET_LEGACY build used by SA-MP and
// open.mp). Source: Include/raknet/PacketEnumerations.h "#if RAKNET_LEGACY".
const (
	IDUnknown                   byte = 0
	IDInternalPing              byte = 6
	IDPing                      byte = 7
	IDPingOpenConnections       byte = 8
	IDConnectedPong             byte = 9
	IDRequestStaticData         byte = 10
	IDConnectionRequest         byte = 11
	IDAuthKey                   byte = 12
	IDBroadcastPings            byte = 15
	IDSecuredConnectionResp     byte = 16
	IDSecuredConnectionConf     byte = 17
	IDRPCMapping                byte = 18
	IDSetRandomNumberSeed       byte = 19
	IDRPC                       byte = 20
	IDRPCReply                  byte = 21
	IDDetectLostConnections     byte = 23
	IDOpenConnectionRequest     byte = 24
	IDOpenConnectionReply       byte = 25
	IDOpenConnectionCookie      byte = 26
	IDRSAPublicKeyMismatch      byte = 28
	IDConnectionAttemptFailed   byte = 29
	IDNewIncomingConnection     byte = 30
	IDNoFreeIncomingConns       byte = 31
	IDDisconnectionNotification byte = 32
	IDConnectionLost            byte = 33
	IDConnectionRequestAccepted byte = 34
	IDInitializeEncryption      byte = 35
	IDConnectionBanned          byte = 36
	IDInvalidPassword           byte = 37
	IDModifiedPacket            byte = 38
	IDPong                      byte = 39
	IDTimestamp                 byte = 40
	IDReceivedStaticData        byte = 41
	IDRemoteDisconnect          byte = 42
	IDRemoteConnectionLost      byte = 43
	IDRemoteNewIncomingConn     byte = 44
	IDRemoteExistingConn        byte = 45
	IDRemoteStaticData          byte = 46
	IDAdvertiseSystem           byte = 55
)

// PacketReliability values as transmitted on the wire in the legacy build
// (4-bit field; SA-MP expects the +6 offset). Source: PacketPriority.h.
const (
	Unreliable          byte = 6
	UnreliableSequenced byte = 7
	Reliable            byte = 8
	ReliableOrdered     byte = 9
	ReliableSequenced   byte = 10
)

// HasOrderingField reports whether the reliability carries orderingChannel and
// orderingIndex fields.
func HasOrderingField(rel byte) bool {
	return rel == UnreliableSequenced || rel == ReliableSequenced || rel == ReliableOrdered
}

// IsValidReliability reports whether rel is one of the five legacy values.
func IsValidReliability(rel byte) bool {
	return rel >= Unreliable && rel <= ReliableSequenced
}

// NumberOfOrderedStreams is the number of ordering channels (5-bit field).
const NumberOfOrderedStreams = 32

// UDPHeaderSize is subtracted from the MTU to obtain the maximum payload for a
// datagram (ReliabilityLayer.cpp: maxDataBitSize = MTUSize - UDP_HEADER_SIZE).
const UDPHeaderSize = 28

// DefaultMTUSize mirrors open.mp's default MTU for SA-MP compatibility.
const DefaultMTUSize = 1500
