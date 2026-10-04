package protocol

import (
	"math"

	"gosamp/internal/raknet"
)

// TextDraw pool layout (SA-MP): global ids 0..2047, per-player ids 0..255 that
// map to wire ids 2048..2303. INVALID_TEXTDRAW (0xFFFF) means "clicked empty
// area / cancelled".
const (
	GlobalTextDrawPoolSize = 2048
	MaxGlobalTextDrawID    = 2047
	MaxPlayerTextDrawID    = 255
	InvalidTextDrawID      = 0xFFFF
)

// TextDraw flags bitfield for ShowTextDraw (RPC 134).
const (
	tdFlagUseBox       = 1 << 0
	tdFlagAlignmentPos = 1 // alignment shifted left by 1
	tdFlagProportional = 1 << 4
)

// TextDrawStyle is the SA-MP textdraw style.
type TextDrawStyle = uint8

// ShowTextDraw is the full state of one textdraw, encoded as RPC 134.
//
// Wire layout (open.mp PlayerShowTextDraw::write):
//
//	uint16 id, uint8 flags(UseBox|Alignment<<1|Proportional<<4),
//	VEC2 letterSize, uint32 letterColour(ABGR), VEC2 textSize,
//	uint32 boxColour(ABGR), uint8 shadow, uint8 outline,
//	uint32 backgroundColour(ABGR), uint8 style, uint8 selectable,
//	VEC2 position, uint16 model, VEC3 rotation, float zoom,
//	int16 color1, int16 color2, string16 text
//
// Note colour1/colour2 are two separate INT16 values (not one INT32); writing
// them as int32 shifts every following field and corrupts the text offset.
type ShowTextDraw struct {
	PlayerTextDraw   bool
	ID               int
	UseBox           bool
	Alignment        uint8
	Proportional     bool
	LetterSizeX      float32
	LetterSizeY      float32
	LetterColour     uint32  // RGBA (0xRRGGBBAA)
	TextSizeX        float32 // box width
	TextSizeY        float32 // box height
	BoxColour        uint32  // RGBA
	Shadow           uint8
	Outline          uint8
	BackgroundColour uint32 // RGBA
	Style            uint8
	Selectable       uint8
	PositionX        float32
	PositionY        float32
	Model            uint16
	RotationX        float32
	RotationY        float32
	RotationZ        float32
	Zoom             float32
	Color1           int16
	Color2           int16
	Text             string
}

// wireID returns the id as it appears on the wire (global id, or 2048+id for a
// per-player textdraw).
func (d ShowTextDraw) wireID() uint16 {
	if d.PlayerTextDraw {
		return uint16(GlobalTextDrawPoolSize + d.ID)
	}
	return uint16(d.ID)
}

// abgr converts an RGBA uint32 (0xRRGGBBAA) to SA-MP/network ABGR.
func abgr(rgba uint32) uint32 {
	r := (rgba >> 24) & 0xFF
	g := (rgba >> 16) & 0xFF
	b := (rgba >> 8) & 0xFF
	a := rgba & 0xFF
	return a<<24 | b<<16 | g<<8 | r
}

// BuildShowTextDraw encodes RPC 134 ShowTextDraw.
func BuildShowTextDraw(d ShowTextDraw) []byte {
	bs := raknet.New()
	flags := uint8(0)
	if d.UseBox {
		flags |= tdFlagUseBox
	}
	flags |= (d.Alignment & 0x03) << 1
	if d.Proportional {
		flags |= tdFlagProportional
	}
	bs.WriteUint16(d.wireID())
	bs.WriteUint8(flags)
	bs.WriteUint32(math.Float32bits(d.LetterSizeX))
	bs.WriteUint32(math.Float32bits(d.LetterSizeY))
	bs.WriteUint32(abgr(d.LetterColour))
	bs.WriteUint32(math.Float32bits(d.TextSizeX))
	bs.WriteUint32(math.Float32bits(d.TextSizeY))
	bs.WriteUint32(abgr(d.BoxColour))
	bs.WriteUint8(d.Shadow)
	bs.WriteUint8(d.Outline)
	bs.WriteUint32(abgr(d.BackgroundColour))
	bs.WriteUint8(d.Style)
	bs.WriteUint8(d.Selectable)
	bs.WriteUint32(math.Float32bits(d.PositionX))
	bs.WriteUint32(math.Float32bits(d.PositionY))
	bs.WriteUint16(d.Model)
	bs.WriteUint32(math.Float32bits(d.RotationX))
	bs.WriteUint32(math.Float32bits(d.RotationY))
	bs.WriteUint32(math.Float32bits(d.RotationZ))
	bs.WriteUint32(math.Float32bits(d.Zoom))
	bs.WriteUint16(uint16(d.Color1))
	bs.WriteUint16(uint16(d.Color2))
	writeDynStr16(bs, d.Text)
	return bs.Bytes()[:bs.Len()]
}

// ParseShowTextDraw decodes RPC 134 (used by tests / client tooling). A wire id
// >= 2048 is a per-player textdraw and is returned with PlayerTextDraw=true and
// ID reset to the per-player index.
func ParseShowTextDraw(payload []byte) (ShowTextDraw, error) {
	var d ShowTextDraw
	bs := raknet.FromBytes(payload)
	var err error
	wid, err := bs.ReadUint16()
	if err != nil {
		return d, err
	}
	if wid >= GlobalTextDrawPoolSize {
		d.PlayerTextDraw = true
		d.ID = int(wid - GlobalTextDrawPoolSize)
	} else {
		d.ID = int(wid)
	}
	flags, err := bs.ReadUint8()
	if err != nil {
		return d, err
	}
	d.UseBox = flags&tdFlagUseBox != 0
	d.Alignment = (flags >> 1) & 0x03
	d.Proportional = flags&tdFlagProportional != 0
	if d.LetterSizeX, err = readF32v(bs); err != nil {
		return d, err
	}
	if d.LetterSizeY, err = readF32v(bs); err != nil {
		return d, err
	}
	v, err := bs.ReadUint32()
	if err != nil {
		return d, err
	}
	d.LetterColour = fromABGR(v)
	if d.TextSizeX, err = readF32v(bs); err != nil {
		return d, err
	}
	if d.TextSizeY, err = readF32v(bs); err != nil {
		return d, err
	}
	if v, err = bs.ReadUint32(); err != nil {
		return d, err
	}
	d.BoxColour = fromABGR(v)
	if d.Shadow, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if d.Outline, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if v, err = bs.ReadUint32(); err != nil {
		return d, err
	}
	d.BackgroundColour = fromABGR(v)
	if d.Style, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if d.Selectable, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if d.PositionX, err = readF32v(bs); err != nil {
		return d, err
	}
	if d.PositionY, err = readF32v(bs); err != nil {
		return d, err
	}
	if d.Model, err = bs.ReadUint16(); err != nil {
		return d, err
	}
	if d.RotationX, err = readF32v(bs); err != nil {
		return d, err
	}
	if d.RotationY, err = readF32v(bs); err != nil {
		return d, err
	}
	if d.RotationZ, err = readF32v(bs); err != nil {
		return d, err
	}
	if d.Zoom, err = readF32v(bs); err != nil {
		return d, err
	}
	c1, err := bs.ReadUint16()
	if err != nil {
		return d, err
	}
	d.Color1 = int16(c1)
	c2, err := bs.ReadUint16()
	if err != nil {
		return d, err
	}
	d.Color2 = int16(c2)
	if d.Text, err = readDynStr16(bs); err != nil {
		return d, err
	}
	return d, nil
}

// BuildHideTextDraw encodes RPC 135 HideTextDraw: uint16 wire id.
func BuildHideTextDraw(playerTextDraw bool, id int) []byte {
	bs := raknet.New()
	if playerTextDraw {
		bs.WriteUint16(uint16(GlobalTextDrawPoolSize + id))
	} else {
		bs.WriteUint16(uint16(id))
	}
	return bs.Bytes()[:bs.Len()]
}

// ParseHideTextDraw decodes RPC 135.
func ParseHideTextDraw(payload []byte) (playerTextDraw bool, id int, err error) {
	bs := raknet.FromBytes(payload)
	wid, err := bs.ReadUint16()
	if err != nil {
		return false, 0, err
	}
	if wid >= GlobalTextDrawPoolSize {
		return true, int(wid - GlobalTextDrawPoolSize), nil
	}
	return false, int(wid), nil
}

// BuildTextDrawSetString encodes RPC 105 TextDrawSetString:
// uint16 wire id + string16 text.
func BuildTextDrawSetString(playerTextDraw bool, id int, text string) []byte {
	bs := raknet.New()
	if playerTextDraw {
		bs.WriteUint16(uint16(GlobalTextDrawPoolSize + id))
	} else {
		bs.WriteUint16(uint16(id))
	}
	writeDynStr16(bs, text)
	return bs.Bytes()[:bs.Len()]
}

// ParseTextDrawSetString decodes RPC 105.
func ParseTextDrawSetString(payload []byte) (playerTextDraw bool, id int, text string, err error) {
	bs := raknet.FromBytes(payload)
	wid, err := bs.ReadUint16()
	if err != nil {
		return false, 0, "", err
	}
	if wid >= GlobalTextDrawPoolSize {
		playerTextDraw = true
		id = int(wid - GlobalTextDrawPoolSize)
	} else {
		id = int(wid)
	}
	text, err = readDynStr16(bs)
	return playerTextDraw, id, text, err
}

// BuildToggleSelectTextDraw encodes the server -> client SelectTextDraw (RPC
// 83): one bit enable + uint32 highlight colour (RGBA).
func BuildToggleSelectTextDraw(enable bool, colour uint32) []byte {
	bs := raknet.New()
	bs.WriteBool(enable)
	bs.WriteUint32(colour)
	return bs.Bytes()[:bs.Len()]
}

// ParseToggleSelectTextDraw decodes the server -> client SelectTextDraw (RPC 83).
func ParseToggleSelectTextDraw(payload []byte) (enable bool, colour uint32, err error) {
	bs := raknet.FromBytes(payload)
	if enable, err = bs.ReadBool(); err != nil {
		return false, 0, err
	}
	colour, err = bs.ReadUint32()
	return enable, colour, err
}

// SelectTextDrawEvent is the client -> server SelectTextDraw (RPC 83) when a
// player clicks a selectable textdraw (or empty space => Invalid, id 0xFFFF).
type SelectTextDrawEvent struct {
	PlayerTextDraw bool
	Invalid        bool
	ID             int
}

// ParseSelectTextDraw decodes the client -> server SelectTextDraw (RPC 83).
func ParseSelectTextDraw(payload []byte) (SelectTextDrawEvent, error) {
	var e SelectTextDrawEvent
	bs := raknet.FromBytes(payload)
	wid, err := bs.ReadUint16()
	if err != nil {
		return e, err
	}
	if wid == InvalidTextDrawID {
		e.Invalid = true
		e.ID = int(InvalidTextDrawID)
		return e, nil
	}
	if wid >= GlobalTextDrawPoolSize {
		e.PlayerTextDraw = true
		e.ID = int(wid - GlobalTextDrawPoolSize)
	} else {
		e.ID = int(wid)
	}
	return e, nil
}

// --- helpers ---

func readF32v(bs *raknet.BitStream) (float32, error) {
	v, err := bs.ReadUint32()
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(v), nil
}

// fromABGR converts a network ABGR value back to an RGBA uint32 (0xRRGGBBAA).
func fromABGR(v uint32) uint32 {
	a := (v >> 24) & 0xFF
	b := (v >> 16) & 0xFF
	g := (v >> 8) & 0xFF
	r := v & 0xFF
	return r<<24 | g<<16 | b<<8 | a
}

// writeDynStr16 writes a 16-bit little-endian length prefix followed by the raw
// bytes, at the current bit position (RakNet string16 / BS_WriteString16).
func writeDynStr16(bs *raknet.BitStream, s string) {
	cp := UTF8ToCP1252(s)
	bs.WriteUint16(uint16(len(cp)))
	if len(cp) > 0 {
		bs.WriteBits(cp, len(cp)*8, true)
	}
}

// readDynStr16 reads a 16-bit little-endian length prefix then that many bytes.
func readDynStr16(bs *raknet.BitStream) (string, error) {
	n, err := bs.ReadUint16()
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", nil
	}
	b := make([]byte, int(n))
	if err := bs.ReadBits(b, int(n)*8, true); err != nil {
		return "", err
	}
	return CP1252ToUTF8(b), nil
}
