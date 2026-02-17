package types

import (
	"math/big"
	"testing"

	"github.com/djkazic/p2pool-go/pkg/util"
)

func TestShareHeader_Serialize(t *testing.T) {
	h := ShareHeader{
		Version:   1,
		Timestamp: 1700000000,
		Bits:      0x1d00ffff,
		Nonce:     12345,
	}

	data := h.Serialize()
	if len(data) != 80 {
		t.Fatalf("serialized header length = %d, want 80", len(data))
	}
}

func TestShareHeader_Hash(t *testing.T) {
	h := ShareHeader{
		Version:   1,
		Timestamp: 1700000000,
		Bits:      0x1d00ffff,
		Nonce:     0,
	}

	hash1 := h.Hash()
	hash2 := h.Hash()

	// Same header should produce same hash
	if hash1 != hash2 {
		t.Error("same header produced different hashes")
	}

	// Different nonce should produce different hash
	h.Nonce = 1
	hash3 := h.Hash()
	if hash1 == hash3 {
		t.Error("different nonce produced same hash")
	}
}

func TestShare_MeetsTarget(t *testing.T) {
	s := &Share{
		Header: ShareHeader{
			Version:   1,
			Timestamp: 1700000000,
			Bits:      0x1d00ffff,
			Nonce:     0,
		},
		ShareTarget: util.CompactToTarget(0x207fffff), // Very easy target
	}

	// A very easy target (max 256-bit value) should be met by any hash
	easyTarget := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	if !s.MeetsTarget(easyTarget) {
		t.Error("share should meet very easy target")
	}

	// An impossible target (0) should never be met
	impossibleTarget := big.NewInt(0)
	if s.MeetsTarget(impossibleTarget) {
		t.Error("share should not meet impossible target")
	}
}

func TestShare_HashHex(t *testing.T) {
	s := &Share{
		Header: ShareHeader{
			Version:   1,
			Timestamp: 1700000000,
			Bits:      0x1d00ffff,
			Nonce:     0,
		},
	}

	hex := s.HashHex()
	if len(hex) != 64 {
		t.Errorf("hash hex length = %d, want 64", len(hex))
	}
}

func TestShare_IsBlock(t *testing.T) {
	// With nBits = 0x207fffff (very easy), most hashes will meet target
	s := &Share{
		Header: ShareHeader{
			Version:   1,
			Timestamp: 1700000000,
			Bits:      0x207fffff, // regtest difficulty
			Nonce:     0,
		},
	}

	// With regtest difficulty, almost any hash is a valid block
	if !s.IsBlock() {
		t.Error("share should be a valid block with regtest difficulty")
	}
}

func TestShareDifficulty(t *testing.T) {
	maxTarget := util.CompactToTarget(0x1d00ffff)
	s := &Share{
		ShareTarget: maxTarget,
	}

	diff := ShareDifficulty(s, maxTarget)
	if diff != 1.0 {
		t.Errorf("difficulty = %f, want 1.0", diff)
	}
}
