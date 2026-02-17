package util

import (
	"testing"
)

func TestVarIntRoundTrip(t *testing.T) {
	tests := []uint64{
		0, 1, 0xfc,
		0xfd, 0xfffe, 0xffff,
		0x10000, 0xfffffffe, 0xffffffff,
		0x100000000, 0xffffffffffffffff,
	}

	for _, val := range tests {
		encoded := WriteVarInt(val)
		decoded, n, err := ReadVarInt(encoded)
		if err != nil {
			t.Errorf("ReadVarInt error for %d: %v", val, err)
			continue
		}
		if n != len(encoded) {
			t.Errorf("ReadVarInt bytes consumed = %d, want %d for value %d", n, len(encoded), val)
		}
		if decoded != val {
			t.Errorf("VarInt round-trip failed: %d -> %d", val, decoded)
		}
	}
}

func TestVarIntSizes(t *testing.T) {
	// < 0xfd: 1 byte
	if len(WriteVarInt(0)) != 1 {
		t.Error("VarInt(0) should be 1 byte")
	}
	if len(WriteVarInt(0xfc)) != 1 {
		t.Error("VarInt(0xfc) should be 1 byte")
	}
	// 0xfd-0xffff: 3 bytes
	if len(WriteVarInt(0xfd)) != 3 {
		t.Error("VarInt(0xfd) should be 3 bytes")
	}
	// 0x10000-0xffffffff: 5 bytes
	if len(WriteVarInt(0x10000)) != 5 {
		t.Error("VarInt(0x10000) should be 5 bytes")
	}
	// > 0xffffffff: 9 bytes
	if len(WriteVarInt(0x100000000)) != 9 {
		t.Error("VarInt(0x100000000) should be 9 bytes")
	}
}

func TestHexConversion(t *testing.T) {
	original := []byte{0xde, 0xad, 0xbe, 0xef}
	hexStr := BytesToHex(original)
	if hexStr != "deadbeef" {
		t.Errorf("BytesToHex = %s, want deadbeef", hexStr)
	}

	decoded, err := HexToBytes(hexStr)
	if err != nil {
		t.Errorf("HexToBytes error: %v", err)
	}
	for i := range original {
		if decoded[i] != original[i] {
			t.Errorf("HexToBytes byte %d = %x, want %x", i, decoded[i], original[i])
		}
	}

	// Invalid hex
	_, err = HexToBytes("zzzz")
	if err == nil {
		t.Error("HexToBytes should fail on invalid hex")
	}
}

func TestReadVarIntErrors(t *testing.T) {
	// Empty data
	_, _, err := ReadVarInt([]byte{})
	if err == nil {
		t.Error("ReadVarInt should fail on empty data")
	}

	// Truncated 3-byte varint
	_, _, err = ReadVarInt([]byte{0xfd, 0x01})
	if err == nil {
		t.Error("ReadVarInt should fail on truncated uint16")
	}

	// Truncated 5-byte varint
	_, _, err = ReadVarInt([]byte{0xfe, 0x01, 0x02, 0x03})
	if err == nil {
		t.Error("ReadVarInt should fail on truncated uint32")
	}

	// Truncated 9-byte varint
	_, _, err = ReadVarInt([]byte{0xff, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07})
	if err == nil {
		t.Error("ReadVarInt should fail on truncated uint64")
	}
}
