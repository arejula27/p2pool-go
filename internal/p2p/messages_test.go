package p2p

import (
	"testing"
)

func TestShareMsg_RoundTrip(t *testing.T) {
	original := &ShareMsg{
		Type:            MsgTypeShare,
		Version:         536870912,
		Timestamp:       1700000000,
		Bits:            0x1d00ffff,
		Nonce:           12345,
		ShareVersion:    1,
		MinerAddress:    "tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx",
		CoinbaseTx:      []byte{0x01, 0x02, 0x03},
		ShareTargetBits: 0x207fffff,
	}
	original.PrevShareHash[0] = 0xab

	data, err := Encode(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded, err := DecodeShareMsg(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if decoded.Version != original.Version {
		t.Errorf("version mismatch: %d != %d", decoded.Version, original.Version)
	}
	if decoded.MinerAddress != original.MinerAddress {
		t.Errorf("miner address mismatch")
	}
	if decoded.PrevShareHash[0] != 0xab {
		t.Errorf("prev share hash mismatch")
	}
	if decoded.ShareTargetBits != original.ShareTargetBits {
		t.Errorf("share target bits mismatch")
	}
}

func TestTipAnnounce_RoundTrip(t *testing.T) {
	original := &TipAnnounce{
		Type:      MsgTypeTipAnnounce,
		Height:    800000,
		TotalWork: []byte{0x01, 0x23, 0x45},
	}
	original.TipHash[0] = 0xcd

	data, err := Encode(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded, err := DecodeTipAnnounce(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if decoded.Height != 800000 {
		t.Errorf("height = %d, want 800000", decoded.Height)
	}
	if decoded.TipHash[0] != 0xcd {
		t.Errorf("tip hash mismatch")
	}
}

func TestShareRequest_RoundTrip(t *testing.T) {
	original := &ShareRequest{
		Type:  MsgTypeShareReq,
		Count: 50,
	}
	original.StartHash[0] = 0xef

	data, err := Encode(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded, err := DecodeShareRequest(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if decoded.Count != 50 {
		t.Errorf("count = %d, want 50", decoded.Count)
	}
	if decoded.StartHash[0] != 0xef {
		t.Errorf("start hash mismatch")
	}
}

func TestBigIntConversion(t *testing.T) {
	// Test with nil
	b := BigIntToBytes(nil)
	if b != nil {
		t.Error("nil input should give nil output")
	}

	result := BytesToBigInt(nil)
	if result.Sign() != 0 {
		t.Error("nil input should give zero")
	}

	// Test round trip
	original := BytesToBigInt([]byte{0x01, 0x00, 0x00})
	b = BigIntToBytes(original)
	result = BytesToBigInt(b)
	if result.Cmp(original) != 0 {
		t.Errorf("round trip failed: %s != %s", result, original)
	}
}
