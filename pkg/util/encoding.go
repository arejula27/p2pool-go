package util

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// HexToBytes decodes a hex string to bytes, returning an error if invalid.
func HexToBytes(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// BytesToHex encodes bytes to a hex string.
func BytesToHex(b []byte) string {
	return hex.EncodeToString(b)
}

// WriteVarInt writes a Bitcoin-style variable-length integer to a byte slice.
// Returns the bytes written.
func WriteVarInt(val uint64) []byte {
	switch {
	case val < 0xfd:
		return []byte{byte(val)}
	case val <= 0xffff:
		b := make([]byte, 3)
		b[0] = 0xfd
		binary.LittleEndian.PutUint16(b[1:], uint16(val))
		return b
	case val <= 0xffffffff:
		b := make([]byte, 5)
		b[0] = 0xfe
		binary.LittleEndian.PutUint32(b[1:], uint32(val))
		return b
	default:
		b := make([]byte, 9)
		b[0] = 0xff
		binary.LittleEndian.PutUint64(b[1:], val)
		return b
	}
}

// ReadVarInt reads a Bitcoin-style variable-length integer from a byte slice.
// Returns the value and the number of bytes consumed.
func ReadVarInt(data []byte) (uint64, int, error) {
	if len(data) == 0 {
		return 0, 0, fmt.Errorf("empty data")
	}

	switch {
	case data[0] < 0xfd:
		return uint64(data[0]), 1, nil
	case data[0] == 0xfd:
		if len(data) < 3 {
			return 0, 0, fmt.Errorf("insufficient data for uint16 varint")
		}
		return uint64(binary.LittleEndian.Uint16(data[1:3])), 3, nil
	case data[0] == 0xfe:
		if len(data) < 5 {
			return 0, 0, fmt.Errorf("insufficient data for uint32 varint")
		}
		return uint64(binary.LittleEndian.Uint32(data[1:5])), 5, nil
	default:
		if len(data) < 9 {
			return 0, 0, fmt.Errorf("insufficient data for uint64 varint")
		}
		return binary.LittleEndian.Uint64(data[1:9]), 9, nil
	}
}

// WriteScriptLen writes a Bitcoin script length prefix.
func WriteScriptLen(length int) []byte {
	switch {
	case length < 0x4c:
		return []byte{byte(length)}
	case length <= 0xff:
		return []byte{0x4c, byte(length)}
	case length <= 0xffff:
		b := make([]byte, 3)
		b[0] = 0x4d
		binary.LittleEndian.PutUint16(b[1:], uint16(length))
		return b
	default:
		b := make([]byte, 5)
		b[0] = 0x4e
		binary.LittleEndian.PutUint32(b[1:], uint32(length))
		return b
	}
}
