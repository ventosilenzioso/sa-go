package protocol

import "encoding/binary"

// SA-MP server query protocol, ported from open.mp
// Server/Components/LegacyNetwork/Query/query.cpp.
//
// Request: "SAMP" + IPv4 (4 bytes) + port (uint16 LE) + type byte.
// The 10-byte header (without the type byte) is echoed into every response.

// QueryHeaderSize is the request prefix length before the type byte.
const QueryHeaderSize = 10

// QuerySize is the size of a headerless-size query (header + type byte).
const QuerySize = 11

// Query type bytes.
const (
	QueryPing    = 'p'
	QueryInfo    = 'i'
	QueryPlayers = 'c'
	QueryRules   = 'r'
	QueryExtra   = 'o' // open.mp extension
	QueryRCON    = 'x'
)

const (
	maxQueryHostname  = 63
	maxQueryGameMode  = 39
	maxQueryLanguage  = 39
	maxQueryDiscord   = 50
	maxQueryImageURL  = 160
	maxQueryPlayers   = 100
	maxPlayerNameSize = 24
)

// QueryPlayer is one entry of the player list response.
type QueryPlayer struct {
	Name  string
	Score int32
}

// QueryState is the server-side information exposed over the query protocol.
// All string fields are truncated to the limits SA-MP/open.mp enforce.
type QueryState struct {
	Hostname   string
	GameMode   string
	Language   string
	Passworded bool
	MaxPlayers uint16
	Players    []QueryPlayer
	Rules      [][2]string

	// open.mp 'o' extension; vanilla 0.3.7 clients never request this.
	DiscordLink    string
	LightBannerURL string
	DarkBannerURL  string
	LogoURL        string
}

// IsQueryRequest reports whether data looks like a query (never obfuscated).
func IsQueryRequest(data []byte) bool {
	return len(data) > QueryHeaderSize+1 && string(data[:4]) == "SAMP"
}

// BuildQueryRequest builds a client-style query request (used by tests/tools).
func BuildQueryRequest(ip [4]byte, port uint16, kind byte) []byte {
	req := make([]byte, 11)
	copy(req, "SAMP")
	copy(req[4:8], ip[:])
	binary.LittleEndian.PutUint16(req[8:10], port)
	req[10] = kind
	return req
}

// HandleQuery produces the raw datagram to send back for a query request.
// It returns nil when the request is malformed, unsupported or filtered
// (mirroring open.mp returning an empty span). RCON queries are handled by
// HandleRCON instead and return nil here.
func (s *QueryState) HandleQuery(req []byte) []byte {
	if len(req) < QuerySize {
		return nil
	}
	kind := req[QueryHeaderSize]
	switch kind {
	case QueryPing:
		if len(req) != QuerySize+4 {
			return nil
		}
		return append([]byte(nil), req...)
	case QueryInfo:
		if len(req) != QuerySize {
			return nil
		}
		return s.buildInfoBuffer(req)
	case QueryPlayers:
		if len(req) != QuerySize {
			return nil
		}
		return s.buildPlayersBuffer(req)
	case QueryRules:
		if len(req) != QuerySize {
			return nil
		}
		return s.buildRulesBuffer(req)
	case QueryExtra:
		if len(req) != QuerySize {
			return nil
		}
		return s.buildExtraBuffer(req)
	default:
		return nil
	}
}

// echoHeader copies the 10-byte request header into a fresh response.
func echoHeader(req []byte) []byte {
	out := make([]byte, QueryHeaderSize, 64)
	copy(out, req[:QueryHeaderSize])
	return out
}

func (s *QueryState) buildInfoBuffer(req []byte) []byte {
	name := UTF8ToCP1252(s.Hostname)
	if len(name) > maxQueryHostname {
		name = name[:maxQueryHostname]
	}
	gm := UTF8ToCP1252(s.GameMode)
	if len(gm) > maxQueryGameMode {
		gm = gm[:maxQueryGameMode]
	}
	lang := UTF8ToCP1252(s.Language)
	if len(lang) > maxQueryLanguage {
		lang = lang[:maxQueryLanguage]
	}

	out := echoHeader(req)
	out = append(out, QueryInfo)
	if s.Passworded {
		out = append(out, 1)
	} else {
		out = append(out, 0)
	}
	out = appendU16(out, uint16(len(s.Players)))
	out = appendU16(out, s.MaxPlayers)
	out = appendU32(out, uint32(len(name)))
	out = append(out, name...)
	out = appendU32(out, uint32(len(gm)))
	out = append(out, gm...)
	out = appendU32(out, uint32(len(lang)))
	out = append(out, lang...)
	return out
}

func (s *QueryState) buildPlayersBuffer(req []byte) []byte {
	players := s.Players
	if len(players) > maxQueryPlayers {
		return nil // open.mp clears the buffer past 100 players
	}
	out := echoHeader(req)
	out = append(out, QueryPlayers)
	out = appendU16(out, uint16(len(players)))
	for _, p := range players {
		name := UTF8ToCP1252(p.Name)
		if len(name) > maxPlayerNameSize {
			name = name[:maxPlayerNameSize]
		}
		out = append(out, byte(len(name)))
		out = append(out, name...)
		out = appendI32(out, p.Score)
	}
	return out
}

func (s *QueryState) buildRulesBuffer(req []byte) []byte {
	out := echoHeader(req)
	out = append(out, QueryRules)
	out = appendU16(out, uint16(len(s.Rules)))
	for _, r := range s.Rules {
		name := UTF8ToCP1252(r[0])
		value := UTF8ToCP1252(r[1])
		out = append(out, byte(len(name)))
		out = append(out, name...)
		out = append(out, byte(len(value)))
		out = append(out, value...)
	}
	return out
}

func (s *QueryState) buildExtraBuffer(req []byte) []byte {
	discord := []byte(s.DiscordLink)
	if len(discord) > maxQueryDiscord {
		discord = nil // open.mp sends length 0 for oversized links
	}
	light := trunc(s.LightBannerURL, maxQueryImageURL)
	dark := trunc(s.DarkBannerURL, maxQueryImageURL)
	logo := trunc(s.LogoURL, maxQueryImageURL)

	out := echoHeader(req)
	out = append(out, QueryExtra)
	out = appendU32(out, uint32(len(discord)))
	out = append(out, discord...)
	out = appendU32(out, uint32(len(light)))
	out = append(out, light...)
	out = appendU32(out, uint32(len(dark)))
	out = append(out, dark...)
	out = appendU32(out, uint32(len(logo)))
	out = append(out, logo...)
	return out
}

// ParseRCON extracts password and command from an 'x' query. ok is false for
// malformed requests.
func ParseRCON(req []byte) (password, command string, ok bool) {
	if len(req) < QuerySize+2 {
		return "", "", false
	}
	rest := req[QuerySize:]
	if len(rest) < 2 {
		return "", "", false
	}
	passLen := int(binary.LittleEndian.Uint16(rest))
	rest = rest[2:]
	if len(rest) < passLen+2 {
		return "", "", false
	}
	password = string(rest[:passLen])
	rest = rest[passLen:]
	cmdLen := int(binary.LittleEndian.Uint16(rest))
	rest = rest[2:]
	if len(rest) != cmdLen {
		return "", "", false
	}
	return password, string(rest), true
}

// BuildRCONRequest builds an 'x' query carrying password and command.
func BuildRCONRequest(ip [4]byte, port uint16, password, command string) []byte {
	req := make([]byte, 0, 11+2+len(password)+2+len(command))
	req = append(req, BuildQueryRequest(ip, port, QueryRCON)...)
	req = appendU16(req, uint16(len(password)))
	req = append(req, password...)
	req = appendU16(req, uint16(len(command)))
	req = append(req, command...)
	return req
}

// BuildRCONResponse wraps a console message for an RCON reply: the request's
// 11-byte header followed by uint16 length and the message text.
func BuildRCONResponse(req []byte, message string) []byte {
	out := make([]byte, 0, 11+2+len(message))
	out = append(out, req[:QuerySize]...)
	out = appendU16(out, uint16(len(message)))
	out = append(out, message...)
	return out
}

func trunc(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

func appendU16(b []byte, v uint16) []byte {
	return append(b, byte(v), byte(v>>8))
}

func appendU32(b []byte, v uint32) []byte {
	return append(b, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

func appendI32(b []byte, v int32) []byte {
	return appendU32(b, uint32(v))
}
