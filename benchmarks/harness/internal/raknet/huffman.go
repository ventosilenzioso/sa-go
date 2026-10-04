package raknet

// RakNet Huffman string compression (StringCompressor), as used by SA-MP /
// open.mp for the CSTRING (szInfo) field of RPC ShowDialog (61). Ported from
// open.mp-network Encoding/{str_compress,huffman_tree}.cpp.

// englishCharacterFrequencies is the fixed frequency table RakNet uses to build
// the Huffman tree. The tree must match the client's byte-for-byte.
var englishCharacterFrequencies = [256]uint32{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 722, 0, 0, 2, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	11084, 58, 63, 1, 0, 31, 0, 317, 64, 64, 44, 0, 695, 62, 980, 266,
	69, 67, 56, 7, 73, 3, 14, 2, 69, 1, 167, 9, 1, 2, 25, 94,
	0, 195, 139, 34, 96, 48, 103, 56, 125, 653, 21, 5, 23, 64, 85, 44,
	34, 7, 92, 76, 147, 12, 14, 57, 15, 39, 15, 1, 1, 1, 2, 3,
	0, 3611, 845, 1077, 1884, 5870, 841, 1057, 2501, 3212, 164, 531, 2019, 1330, 3056, 4037,
	848, 47, 2586, 2919, 4771, 1707, 535, 1106, 152, 1243, 100, 0, 2, 0, 10, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
}

type huffNode struct {
	value  byte
	weight uint32
	left   *huffNode
	right  *huffNode
	parent *huffNode
}

// huffStringCompressor is a singleton holding the fixed encoding table.
type huffStringCompressor struct {
	encoding  [256][]byte
	bitLength [256]int
	root      *huffNode
}

var stringCompressor = newHuffStringCompressor()

func newHuffStringCompressor() *huffStringCompressor {
	sc := &huffStringCompressor{}
	sc.generate(englishCharacterFrequencies)
	return sc
}

// insertSorted mirrors RakNet InsertNodeIntoSortedList: insert before the first
// node whose weight >= the new node's weight, else append.
func insertSorted(list []*huffNode, node *huffNode) []*huffNode {
	for i := 0; i < len(list); i++ {
		if list[i].weight >= node.weight {
			list = append(list, nil)
			copy(list[i+1:], list[i:])
			list[i] = node
			return list
		}
	}
	return append(list, node)
}

func (sc *huffStringCompressor) generate(freq [256]uint32) {
	var leafList [256]*huffNode
	list := make([]*huffNode, 0, 256)
	for i := 0; i < 256; i++ {
		n := &huffNode{value: byte(i), weight: freq[i]}
		if n.weight == 0 {
			n.weight = 1
		}
		leafList[i] = n
		list = insertSorted(list, n)
	}

	for {
		lesser := list[0]
		greater := list[1]
		list = list[2:]
		n := &huffNode{left: lesser, right: greater, weight: lesser.weight + greater.weight}
		lesser.parent = n
		greater.parent = n
		if len(list) == 0 {
			sc.root = n
			n.parent = nil
			break
		}
		list = insertSorted(list, n)
	}

	// Generate the per-character encodings (root -> leaf bits, left aligned).
	for i := 0; i < 256; i++ {
		var path [256]bool
		plen := 0
		cur := leafList[i]
		for cur != sc.root {
			if cur.parent.left == cur {
				path[plen] = false
			} else {
				path[plen] = true
			}
			plen++
			cur = cur.parent
		}
		// Pack bits in root->leaf order, left aligned.
		nbytes := (plen + 7) / 8
		enc := make([]byte, nbytes)
		bitPos := 0
		for plen > 0 {
			plen--
			if path[plen] {
				enc[bitPos>>3] |= 0x80 >> uint(bitPos&7)
			}
			bitPos++
		}
		sc.encoding[i] = enc
		sc.bitLength[i] = bitPos
	}
}

// encodeArray writes the Huffman-encoded bytes of input to bs (left aligned),
// padding to a byte boundary.
func (sc *huffStringCompressor) encodeArray(input []byte, bs *BitStream) {
	for _, b := range input {
		bs.WriteBits(sc.encoding[b], sc.bitLength[b], false)
	}
	if bs.BitLen()%8 != 0 {
		remaining := 8 - (bs.BitLen() % 8)
		for i := 0; i < 256; i++ {
			if sc.bitLength[i] > remaining {
				bs.WriteBits(sc.encoding[i], remaining, false)
				break
			}
		}
	}
}

// encodeString writes a Huffman-compressed string: a compressed uint16 bit
// length followed by that many bits. Mirrors StringCompressor::EncodeString.
func (sc *huffStringCompressor) encodeString(input []byte, bs *BitStream) {
	var tmp BitStream
	sc.encodeArray(input, &tmp)
	bitLen := uint16(tmp.BitLen())
	bs.WriteCompressedUint16(bitLen)
	bs.WriteBits(tmp.Bytes(), int(bitLen), true)
}

// decodeString reads a Huffman-compressed string (compressed uint16 bit length
// then bits) and returns the decoded bytes.
func (sc *huffStringCompressor) decodeString(bs *BitStream) ([]byte, error) {
	bitLen, err := bs.ReadCompressedUint16()
	if err != nil {
		return nil, err
	}
	if bs.UnreadBits() < int(bitLen) {
		return nil, errShortRead
	}
	var out []byte
	cur := sc.root
	for i := 0; i < int(bitLen); i++ {
		bit, err := bs.ReadBool()
		if err != nil {
			return nil, err
		}
		if bit {
			cur = cur.right
		} else {
			cur = cur.left
		}
		if cur.left == nil && cur.right == nil {
			out = append(out, cur.value)
			cur = sc.root
		}
	}
	return out, nil
}

// WriteCompressedStr writes a RakNet Huffman-compressed string (the CSTRING
// field used by RPC ShowDialog / 61). The bytes are the CP1252 string content
// without a NUL terminator.
func (bs *BitStream) WriteCompressedStr(data []byte) {
	stringCompressor.encodeString(data, bs)
}

// ReadCompressedStr reads a RakNet Huffman-compressed string.
func (bs *BitStream) ReadCompressedStr() ([]byte, error) {
	return stringCompressor.decodeString(bs)
}
