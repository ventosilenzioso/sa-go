// Package raknet implements the RakNet (SA-MP legacy / RAKNET_LEGACY) wire
// protocol: bitstream codec, datagram framing, reliability layer and offline
// handshake. All encodings are byte-for-byte compatible with the reference
// implementations (open.mp RakNet, RakSAMP) as verified against their sources.
package raknet

import "errors"

var (
	errShortRead = errors.New("raknet: bitstream: not enough bits")
	errMalformed = errors.New("raknet: bitstream: malformed stream")
)

// BitStream is a MSB-first bit-oriented buffer compatible with RakNet's
// RakNet::BitStream (Source/BitStream.cpp). Bits are packed starting at the
// most significant bit of each byte; multi-byte scalars are stored little-endian.
type BitStream struct {
	buf     []byte
	bitLen  int // number of valid bits written
	readPos int // read cursor in bits
}

// New returns an empty BitStream for writing.
func New() *BitStream {
	return &BitStream{buf: make([]byte, 0, 64)}
}

// FromBytes wraps data for reading. The whole buffer is considered readable.
func FromBytes(data []byte) *BitStream {
	return &BitStream{buf: data, bitLen: len(data) * 8}
}

// Bytes returns the underlying buffer (only the first BitLen()/8 bytes are valid
// if BitLen is not byte aligned).
func (bs *BitStream) Bytes() []byte { return bs.buf }

// Len returns the number of bytes used, rounded up.
func (bs *BitStream) Len() int { return (bs.bitLen + 7) / 8 }

// BitLen returns the number of bits written to the stream.
func (bs *BitStream) BitLen() int { return bs.bitLen }

// UnreadBits returns the number of bits left to read.
func (bs *BitStream) UnreadBits() int { return bs.bitLen - bs.readPos }

// ReadOffset returns the current read position in bits.
func (bs *BitStream) ReadOffset() int { return bs.readPos }

// SetReadOffset rewinds or advances the read cursor (used when re-parsing).
func (bs *BitStream) SetReadOffset(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos > bs.bitLen {
		pos = bs.bitLen
	}
	bs.readPos = pos
}

func (bs *BitStream) ensureByte(i int) {
	for len(bs.buf) <= i {
		bs.buf = append(bs.buf, 0)
	}
}

func (bs *BitStream) readByte(i int) byte {
	if i >= len(bs.buf) {
		return 0
	}
	return bs.buf[i]
}

// WriteBits is a faithful port of BitStream::WriteBits.
// When rightAligned is true and fewer than 8 bits are written, the low bits of
// src are shifted into the most significant positions of the stream byte.
func (bs *BitStream) WriteBits(src []byte, nbits int, rightAligned bool) {
	if nbits <= 0 {
		return
	}
	offset := 0
	usedMod8 := bs.bitLen & 7
	for nbits > 0 {
		var dataByte byte
		if offset < len(src) {
			dataByte = src[offset]
		}
		if nbits < 8 && rightAligned {
			dataByte <<= 8 - nbits
		}

		if usedMod8 == 0 {
			bs.ensureByte(bs.bitLen >> 3)
			bs.buf[bs.bitLen>>3] = dataByte
		} else {
			bs.ensureByte(bs.bitLen>>3 + 1)
			bs.buf[bs.bitLen>>3] |= dataByte >> usedMod8
			if 8-usedMod8 < 8 && 8-usedMod8 < nbits {
				bs.buf[bs.bitLen>>3+1] = dataByte << (8 - usedMod8)
			}
		}

		if nbits >= 8 {
			bs.bitLen += 8
		} else {
			bs.bitLen += nbits
		}
		nbits -= 8
		offset++
	}
}

// ReadBits is a faithful port of BitStream::ReadBits (alignBitsToRight=true
// default). It returns errShortRead when the stream does not hold enough bits.
func (bs *BitStream) ReadBits(dst []byte, nbits int, alignToRight bool) error {
	if nbits <= 0 {
		return errMalformed
	}
	if bs.UnreadBits() < nbits {
		return errShortRead
	}
	for i := 0; i < (nbits+7)/8; i++ {
		dst[i] = 0
	}
	readMod8 := bs.readPos & 7
	offset := 0
	for nbits > 0 {
		dst[offset] |= bs.readByte(bs.readPos>>3) << readMod8
		if readMod8 > 0 && nbits > 8-readMod8 {
			dst[offset] |= bs.readByte(bs.readPos>>3+1) >> (8 - readMod8)
		}
		nbits -= 8
		if nbits < 0 {
			if alignToRight {
				dst[offset] >>= -nbits
			}
			bs.readPos += 8 + nbits
		} else {
			bs.readPos += 8
		}
		offset++
	}
	return nil
}

// WriteBool writes a single bit (RakNet Write(bool)).
func (bs *BitStream) WriteBool(v bool) {
	usedMod8 := bs.bitLen & 7
	if usedMod8 == 0 {
		bs.ensureByte(bs.bitLen >> 3)
		bs.buf[bs.bitLen>>3] = 0
	}
	if v {
		bs.buf[bs.bitLen>>3] |= 0x80 >> usedMod8
	}
	bs.bitLen++
}

// ReadBool reads a single bit.
func (bs *BitStream) ReadBool() (bool, error) {
	if bs.UnreadBits() < 1 {
		return false, errShortRead
	}
	v := (bs.buf[bs.readPos>>3] << (bs.readPos & 7)) & 0x80
	bs.readPos++
	return v != 0, nil
}

// WriteUint8 writes 8 bits (RakNet Write(unsigned char)).
func (bs *BitStream) WriteUint8(v byte) {
	bs.WriteBits([]byte{v}, 8, true)
}

// ReadUint8 reads 8 bits.
func (bs *BitStream) ReadUint8() (byte, error) {
	var b [1]byte
	if err := bs.ReadBits(b[:], 8, true); err != nil {
		return 0, err
	}
	return b[0], nil
}

// WriteUint16 writes a little-endian uint16.
func (bs *BitStream) WriteUint16(v uint16) {
	bs.WriteBits([]byte{byte(v), byte(v >> 8)}, 16, true)
}

// ReadUint16 reads a little-endian uint16.
func (bs *BitStream) ReadUint16() (uint16, error) {
	var b [2]byte
	if err := bs.ReadBits(b[:], 16, true); err != nil {
		return 0, err
	}
	return uint16(b[0]) | uint16(b[1])<<8, nil
}

// WriteUint32 writes a little-endian uint32.
func (bs *BitStream) WriteUint32(v uint32) {
	bs.WriteBits([]byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}, 32, true)
}

// ReadUint32 reads a little-endian uint32.
func (bs *BitStream) ReadUint32() (uint32, error) {
	var b [4]byte
	if err := bs.ReadBits(b[:], 32, true); err != nil {
		return 0, err
	}
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24, nil
}

// WriteUint64 writes a little-endian uint64.
func (bs *BitStream) WriteUint64(v uint64) {
	var b [8]byte
	for i := range b {
		b[i] = byte(v >> (8 * i))
	}
	bs.WriteBits(b[:], 64, true)
}

// ReadUint64 reads a little-endian uint64.
func (bs *BitStream) ReadUint64() (uint64, error) {
	var b [8]byte
	if err := bs.ReadBits(b[:], 64, true); err != nil {
		return 0, err
	}
	var v uint64
	for i := range b {
		v |= uint64(b[i]) << (8 * i)
	}
	return v, nil
}

// AlignWrite pads the stream to the next byte boundary (no-op if aligned).
func (bs *BitStream) AlignWrite() {
	if bs.bitLen > 0 {
		bs.bitLen += 8 - (((bs.bitLen - 1) & 7) + 1)
	}
}

// AlignRead advances the read cursor to the next byte boundary.
func (bs *BitStream) AlignRead() {
	if bs.readPos > 0 {
		bs.readPos += 8 - (((bs.readPos - 1) & 7) + 1)
		if bs.readPos > bs.bitLen {
			bs.readPos = bs.bitLen
		}
	}
}

// WriteAlignedBytes byte-aligns then writes whole bytes.
func (bs *BitStream) WriteAlignedBytes(src []byte) {
	bs.AlignWrite()
	bs.WriteBits(src, len(src)*8, true)
}

// ReadAlignedBytes byte-aligns then reads whole bytes.
func (bs *BitStream) ReadAlignedBytes(n int) ([]byte, error) {
	bs.AlignRead()
	dst := make([]byte, n)
	if err := bs.ReadBits(dst, n*8, true); err != nil {
		return nil, err
	}
	return dst, nil
}

// WriteCompressed is a port of BitStream::WriteCompressed.
// sizeBits is the bit width of the value (e.g. 16 for uint16); src holds the
// value in native (little-endian) byte order.
func (bs *BitStream) WriteCompressed(src []byte, sizeBits int, unsignedData bool) {
	currentByte := (sizeBits >> 3) - 1

	var byteMatch byte
	if unsignedData {
		byteMatch = 0
	} else {
		byteMatch = 0xFF
	}

	for currentByte > 0 {
		if src[currentByte] == byteMatch {
			bs.WriteBool(true)
		} else {
			bs.WriteBool(false)
			bs.WriteBits(src, (currentByte+1)<<3, true)
			return
		}
		currentByte--
	}

	if (unsignedData && src[0]&0xF0 == 0x00) || (!unsignedData && src[0]&0xF0 == 0xF0) {
		bs.WriteBool(true)
		bs.WriteBits(src[0:1], 4, true)
	} else {
		bs.WriteBool(false)
		bs.WriteBits(src[0:1], 8, true)
	}
}

// ReadCompressed is a port of BitStream::ReadCompressed.
func (bs *BitStream) ReadCompressed(sizeBits int, unsignedData bool) ([]byte, error) {
	currentByte := (sizeBits >> 3) - 1
	out := make([]byte, sizeBits/8)

	var byteMatch, halfByteMatch byte
	if unsignedData {
		byteMatch = 0
		halfByteMatch = 0
	} else {
		byteMatch = 0xFF
		halfByteMatch = 0xF0
	}

	for currentByte > 0 {
		b, err := bs.ReadBool()
		if err != nil {
			return nil, err
		}
		if b {
			out[currentByte] = byteMatch
			currentByte--
		} else {
			if err := bs.ReadBits(out, (currentByte+1)<<3, true); err != nil {
				return nil, err
			}
			return out, nil
		}
	}

	if bs.readPos+1 > bs.bitLen {
		return nil, errShortRead
	}
	b, err := bs.ReadBool()
	if err != nil {
		return nil, err
	}
	if b {
		if err := bs.ReadBits(out[currentByte:currentByte+1], 4, true); err != nil {
			return nil, err
		}
		out[currentByte] |= halfByteMatch
	} else {
		if err := bs.ReadBits(out[currentByte:currentByte+1], 8, true); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// WriteCompressedUint16 writes an unsigned compressed uint16 (RakNet
// WriteCompressed(uint16)).
func (bs *BitStream) WriteCompressedUint16(v uint16) {
	bs.WriteCompressed([]byte{byte(v), byte(v >> 8)}, 16, true)
}

// ReadCompressedUint16 reads an unsigned compressed uint16.
func (bs *BitStream) ReadCompressedUint16() (uint16, error) {
	b, err := bs.ReadCompressed(16, true)
	if err != nil {
		return 0, err
	}
	return uint16(b[0]) | uint16(b[1])<<8, nil
}

// WriteCompressedUint32 writes an unsigned compressed uint32.
func (bs *BitStream) WriteCompressedUint32(v uint32) {
	bs.WriteCompressed([]byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}, 32, true)
}

// ReadCompressedUint32 reads an unsigned compressed uint32.
func (bs *BitStream) ReadCompressedUint32() (uint32, error) {
	b, err := bs.ReadCompressed(32, true)
	if err != nil {
		return 0, err
	}
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24, nil
}
