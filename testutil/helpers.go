package testutil

import (
	"encoding/hex"
	"testing"
)

// MustDecodeHex decodes hex or fails the test.
func MustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("invalid hex %q: %v", s, err)
	}
	return b
}

// HashFromHex converts a hex string to a [32]byte, zero-padding if needed.
func HashFromHex(s string) [32]byte {
	b, _ := hex.DecodeString(s)
	var h [32]byte
	copy(h[:], b)
	return h
}
