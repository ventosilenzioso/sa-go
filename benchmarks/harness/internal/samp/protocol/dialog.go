package protocol

import (
	"gosamp/internal/raknet"
)

// SA-MP dialog styles (the `style` byte of RPC_ShowDialog / 61).
const (
	DialogStyleMsgBox   = 0
	DialogStyleInput    = 1
	DialogStyleList     = 2
	DialogStylePassword = 3
	// DialogStyleTabList* and other 0.3.7+ styles exist but are not modeled;
	// the wire byte is passed through unchanged.
)

// DialogIDHidden is the dialog id used to hide the currently shown dialog.
// SA-MP treats a ShowDialog with style 0 and empty strings plus this id as a
// hide; the convention used here (and by open.mp/RakSAMP scripts) is the
// sentinel -1 (0xFFFF as a signed int).
const DialogIDHidden int32 = -1

// ShowDialog is one ShowDialog (61) payload, server -> client.
type ShowDialog struct {
	ID      int32 // sent as uint16; -1 hides
	Style   uint8 // 0..3 (see DialogStyle*)
	Title   string
	Button1 string
	Button2 string // empty = second button hidden
	Body    string // for LIST, items separated by '\n'
}

// BuildShowDialog encodes a ShowDialog (61) payload:
//
//	INT16 dialog id, UINT8 style, UINT8 titleLen + title, UINT8 b1Len + b1,
//	UINT8 b2Len + b2, compressed-string body
//
// This matches open.mp's Shared/NetCode/dialog.hpp exactly: title and both
// buttons are writeDynStr8, while the body/info (the LIST item block) is
// WriteCompressedStr — a RakNet Huffman-compressed string, NOT a plain
// length-prefixed string. An empty button2 hides the second button (SA-MP
// convention), so keep button1 clearly labeled.
func BuildShowDialog(d ShowDialog) []byte {
	bs := raknet.New()
	bs.WriteUint16(uint16(int16(d.ID)))
	bs.WriteUint8(d.Style)
	writeDynStr8(bs, d.Title)
	writeDynStr8(bs, d.Button1)
	writeDynStr8(bs, d.Button2)
	bs.WriteCompressedStr(UTF8ToCP1252(d.Body))
	return bs.Bytes()[:bs.Len()]
}

// BuildHideDialog encodes the payload that dismisses a player's active dialog.
// SA-MP has no dedicated RPC for this; the convention is a ShowDialog with the
// hidden sentinel id and empty strings.
func BuildHideDialog() []byte {
	return BuildShowDialog(ShowDialog{ID: DialogIDHidden})
}

// ParseShowDialog decodes a ShowDialog (61) payload (server -> client).
func ParseShowDialog(payload []byte) (ShowDialog, error) {
	var d ShowDialog
	bs := raknet.FromBytes(payload)
	id, err := bs.ReadUint16()
	if err != nil {
		return d, err
	}
	d.ID = int32(int16(id))
	if d.Style, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	if d.Title, err = readDynStr8(bs); err != nil {
		return d, err
	}
	if d.Button1, err = readDynStr8(bs); err != nil {
		return d, err
	}
	if d.Button2, err = readDynStr8(bs); err != nil {
		return d, err
	}
	body, err := bs.ReadCompressedStr()
	if err != nil {
		return d, err
	}
	d.Body = CP1252ToUTF8(body)
	return d, nil
}

// DialogResponse is one DialogResponse (62) payload, client -> server.
type DialogResponse struct {
	ID       int32  // dialog id the client responded to
	Response uint8  // 1 = left/OK/select, 0 = right/cancel/dismissed
	ListItem int16  // selected list row (LIST), or -1 when not applicable
	Input    string // entered text (INPUT/PASSWORD), empty otherwise
}

// ParseDialogResponse decodes a DialogResponse (62) payload:
//
//	INT16 dialog id, UINT8 response, INT16 listitem, UINT8 textLen + text
//
// This matches SA-MP 0.3.7 (SAMPRPC.cpp RPC_DialogResponse) and SAMP.Lua's
// rpc_dialog_response reader.
func ParseDialogResponse(payload []byte) (DialogResponse, error) {
	var d DialogResponse
	bs := raknet.FromBytes(payload)
	id, err := bs.ReadUint16()
	if err != nil {
		return d, err
	}
	d.ID = int32(int16(id))
	if d.Response, err = bs.ReadUint8(); err != nil {
		return d, err
	}
	li, err := bs.ReadUint16()
	if err != nil {
		return d, err
	}
	d.ListItem = int16(li)
	if d.Input, err = readDynStr8(bs); err != nil {
		return d, err
	}
	return d, nil
}

// BuildDialogResponse builds a DialogResponse (62) payload. Used by tests and
// client tooling; the server never sends this RPC.
func BuildDialogResponse(d DialogResponse) []byte {
	bs := raknet.New()
	bs.WriteUint16(uint16(int16(d.ID)))
	bs.WriteUint8(d.Response)
	bs.WriteUint16(uint16(d.ListItem))
	writeDynStr8(bs, d.Input)
	return bs.Bytes()[:bs.Len()]
}
