package util

import (
	"math/big"
	"testing"
)

func TestDoubleSHA256(t *testing.T) {
	// Known Bitcoin double-SHA256 of "hello"
	data := []byte("hello")
	hash := DoubleSHA256(data)
	hex := BytesToHex(hash[:])
	expected := "9595c9df90075148eb06860365df33584b75bff782a510c6cd4883a419833d50"
	if hex != expected {
		t.Errorf("DoubleSHA256(\"hello\") = %s, want %s", hex, expected)
	}
}

func TestReverseBytes(t *testing.T) {
	input := []byte{0x01, 0x02, 0x03, 0x04}
	result := ReverseBytes(input)
	expected := []byte{0x04, 0x03, 0x02, 0x01}
	for i := range result {
		if result[i] != expected[i] {
			t.Errorf("ReverseBytes byte %d = %x, want %x", i, result[i], expected[i])
		}
	}
	// Original should not be modified
	if input[0] != 0x01 {
		t.Error("ReverseBytes modified original slice")
	}
}

func TestCompactToTarget(t *testing.T) {
	tests := []struct {
		name    string
		compact uint32
		want    string // hex of target
	}{
		{
			name:    "testnet genesis",
			compact: 0x1d00ffff,
			want:    "ffff0000000000000000000000000000000000000000000000000000",
		},
		{
			name:    "zero",
			compact: 0x00000000,
			want:    "0",
		},
		{
			name:    "small exponent",
			compact: 0x03123456,
			want:    "123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := CompactToTarget(tt.compact)
			got := target.Text(16)
			if got != tt.want {
				t.Errorf("CompactToTarget(0x%08x) = %s, want %s", tt.compact, got, tt.want)
			}
		})
	}
}

func TestCompactRoundTrip(t *testing.T) {
	tests := []uint32{
		0x1d00ffff, // testnet
		0x03123456,
		0x04123456,
		0x1b0404cb, // some mainnet difficulty
	}

	for _, compact := range tests {
		target := CompactToTarget(compact)
		got := TargetToCompact(target)
		if got != compact {
			t.Errorf("Round-trip failed: compact 0x%08x -> target -> 0x%08x", compact, got)
		}
	}
}

func TestTargetToDifficulty(t *testing.T) {
	maxTarget := CompactToTarget(0x1d00ffff)
	diff := TargetToDifficulty(maxTarget, maxTarget)
	if diff != 1.0 {
		t.Errorf("Difficulty of max target should be 1.0, got %f", diff)
	}

	// Half the target should give difficulty 2
	halfTarget := new(big.Int).Div(maxTarget, big.NewInt(2))
	diff2 := TargetToDifficulty(halfTarget, maxTarget)
	if diff2 < 1.99 || diff2 > 2.01 {
		t.Errorf("Difficulty of half target should be ~2.0, got %f", diff2)
	}
}

func TestDifficultyToTarget(t *testing.T) {
	maxTarget := CompactToTarget(0x1d00ffff)
	target := DifficultyToTarget(1.0, maxTarget)
	if target.Cmp(maxTarget) != 0 {
		t.Errorf("DifficultyToTarget(1.0) should equal maxTarget")
	}
}

func TestHashMeetsTarget(t *testing.T) {
	target := CompactToTarget(0x1d00ffff)

	// A hash of all zeros should meet any target
	var zeroHash [32]byte
	if !HashMeetsTarget(zeroHash, target) {
		t.Error("Zero hash should meet any positive target")
	}

	// A hash of all 0xFF should not meet a reasonable target
	var maxHash [32]byte
	for i := range maxHash {
		maxHash[i] = 0xFF
	}
	if HashMeetsTarget(maxHash, target) {
		t.Error("Max hash should not meet target")
	}
}
