package protocol

import "bytes"

// Windows-1252 (CP1252) to Unicode code point table for bytes 0x80-0x9F.
// SA-MP uses CP1252 for all string fields in the wire protocol, not UTF-8.
// Source: Unicode.org CP1252 table; verified against RakSAMP server behavior
// and open.mp network code. Bytes 0x00-0x7F and 0xA0-0xFF are identical to
// their byte values as Unicode code points.
//
// ASSUMPTION (documented for future verification via golden capture):
// This table reflects CP1252 as implemented by the SA-MP 0.3.7 client. If
// golden capture reveals the client uses a different charset (e.g. ISO-8859-1
// or Windows-1250 for some locales), this table must be updated. See
// docs/protocol.md §StringEncoding for the open issue.
var cp1252ToUnicode = [32]rune{
	0x20AC, 0x0081, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021, // 0x80-0x87
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008D, 0x017D, 0x008F, // 0x88-0x8F
	0x0090, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014, // 0x90-0x97
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x009D, 0x017E, 0x0178, // 0x98-0x9F
}

var unicodeToCP1252 = map[rune]byte{
	0x20AC: 0x80, 0x201A: 0x82, 0x0192: 0x83, 0x201E: 0x84,
	0x2026: 0x85, 0x2020: 0x86, 0x2021: 0x87, 0x02C6: 0x88,
	0x2030: 0x89, 0x0160: 0x8A, 0x2039: 0x8B, 0x0152: 0x8C,
	0x017D: 0x8E, 0x2018: 0x91, 0x2019: 0x92, 0x201C: 0x93,
	0x201D: 0x94, 0x2022: 0x95, 0x2013: 0x96, 0x2014: 0x97,
	0x02DC: 0x98, 0x2122: 0x99, 0x0161: 0x9A, 0x203A: 0x9B,
	0x0153: 0x9C, 0x017E: 0x9E, 0x0178: 0x9F,
}

// UTF8ToCP1252 converts a UTF-8 Go string to Windows-1252 bytes.
// Characters not representable in CP1252 (control chars 0x80-0x9F, or Unicode
// chars outside U+0000-U+00FF except those explicitly mapped above) are
// replaced with '?'. Length is predictable: len(result) ≤ len(s).
func UTF8ToCP1252(s string) []byte {
	// Fast path: if s is pure ASCII, result is identical.
	asciiOnly := true
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			asciiOnly = false
			break
		}
	}
	if asciiOnly {
		return []byte(s)
	}

	// General case: convert rune by rune.
	var buf bytes.Buffer
	for _, r := range s {
		if r <= 0xFFFD {
			if b, ok := unicodeToCP1252[r]; ok {
				buf.WriteByte(b)
				continue
			}
			if r < 0x80 {
				buf.WriteByte(byte(r))
				continue
			}
			if r >= 0xA0 && r <= 0xFF {
				buf.WriteByte(byte(r))
				continue
			}
		}
		buf.WriteByte('?')
	}
	return buf.Bytes()
}

// CP1252ToUTF8 converts Windows-1252 bytes to a UTF-8 Go string.
// All CP1252 bytes are valid; 0x80-0x9F are translated via the table above,
// 0x00-0x7F stay as single-byte ASCII, 0xA0-0xFF are > U+00A0 in Unicode.
func CP1252ToUTF8(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	asciiOnly := true
	for i := 0; i < len(b); i++ {
		if b[i] >= 0x80 {
			asciiOnly = false
			break
		}
	}
	if asciiOnly {
		return string(b)
	}

	buf := make([]byte, 0, len(b)*2) // *2 conservatively; most chars stay 1 byte
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c < 0x80 {
			buf = append(buf, c)
		} else if c < 0xA0 {
			buf = append(buf, string(cp1252ToUnicode[c-0x80])...)
		} else {
			buf = append(buf, 0xC0|byte(c>>6), 0x80|byte(c&0x3F))
		}
	}
	return string(buf)
}

// TruncateCP1252 truncates s to at most maxByteLen bytes when encoded as
// CP1252, without splitting multi-byte characters. Used for hostname/gamemode
// fields that have hard byte limits in the wire format.
func TruncateCP1252(s string, maxByteLen int) string {
	if maxByteLen <= 0 {
		return ""
	}
	if len(s) <= maxByteLen {
		if isASCII(s) {
			return s
		}
	}
	encoded := UTF8ToCP1252(s)
	if len(encoded) <= maxByteLen {
		return s
	}
	// Binary search for the largest prefix whose CP1252 encoding fits.
	lo, hi := 0, len(s)
	for lo+1 < hi {
		mid := (lo + hi) / 2
		if len(UTF8ToCP1252(s[:mid])) <= maxByteLen {
			lo = mid
		} else {
			hi = mid
		}
	}
	return s[:lo]
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// ValidateHostname checks that a hostname string does not exceed the wire
// protocol limit (63 bytes in CP1252) and returns the valid wire-ready form.
// Empty hostnames are replaced with the default.
func ValidateHostname(s string) []byte {
	if s == "" {
		s = "SA-MP Server"
	}
	cp := UTF8ToCP1252(s)
	if len(cp) > 63 {
		cp = cp[:63]
	}
	return cp
}

// EncodeStr8 encodes a Go string for the SA-MP dynStr8 wire format
// (length-prefixed raw bytes, no null terminator). The string is converted to
// CP1252 since that is SA-MP's wire charset; non-CP1252 characters become '?'.
func EncodeStr8(s string) []byte {
	return UTF8ToCP1252(s)
}

// DecodeStr8 decodes a dynStr8 wire payload to a Go UTF-8 string.
// The input should be raw CP1252 bytes; they are decoded to UTF-8.
func DecodeStr8(b []byte) string {
	return CP1252ToUTF8(b)
}
