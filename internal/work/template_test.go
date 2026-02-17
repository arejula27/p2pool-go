package work

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/djkazic/p2pool-go/pkg/util"
)

// TestMerkleRootConsistency verifies that ComputeFullMerkleRoot produces the
// same root as the branch-based approach (ComputeMerkleBranches + ComputeMerkleRoot).
func TestMerkleRootConsistency(t *testing.T) {
	// Generate some deterministic "transaction hashes" (internal byte order)
	makeTxHash := func(seed byte) []byte {
		data := []byte{seed, seed, seed, seed}
		h := util.DoubleSHA256(data)
		return h[:]
	}

	// Test with different transaction counts (0 to 7 non-coinbase txs)
	for txCount := 0; txCount <= 7; txCount++ {
		// Coinbase hash
		cbData := []byte("coinbase-data-for-test")
		cbHash := util.DoubleSHA256(cbData)

		// Non-coinbase tx hashes
		var txHashes []string
		var allTxIDs [][]byte // coinbase first
		allTxIDs = append(allTxIDs, cbHash[:])

		for i := 0; i < txCount; i++ {
			h := makeTxHash(byte(i + 1))
			txHashes = append(txHashes, hex.EncodeToString(h))
			allTxIDs = append(allTxIDs, h)
		}

		// Method 1: Branch-based (what the miner uses)
		branches, err := ComputeMerkleBranches(txHashes)
		if err != nil {
			t.Fatalf("txCount=%d: ComputeMerkleBranches: %v", txCount, err)
		}
		rootViaBranches, err := ComputeMerkleRoot(cbHash[:], branches)
		if err != nil {
			t.Fatalf("txCount=%d: ComputeMerkleRoot: %v", txCount, err)
		}

		// Method 2: Full merkle tree (what we use for verification)
		rootFull := ComputeFullMerkleRoot(allTxIDs)

		if !bytes.Equal(rootViaBranches, rootFull) {
			t.Errorf("txCount=%d: merkle root mismatch\n  branches: %s\n  full:     %s",
				txCount,
				hex.EncodeToString(rootViaBranches),
				hex.EncodeToString(rootFull),
			)
		}
	}
}

// TestMerkleBranchesEmpty verifies the edge case of no transactions.
func TestMerkleBranchesEmpty(t *testing.T) {
	branches, err := ComputeMerkleBranches(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(branches) != 0 {
		t.Errorf("expected 0 branches, got %d", len(branches))
	}
}

// TestPrevHashRoundTrip verifies display→stratum→internal is correct.
func TestPrevHashRoundTrip(t *testing.T) {
	// A known prevhash in display order (big-endian hex)
	displayHex := "00000000000000000002a7c4c1e48d76c5a37902165a270156b7a8d72f8e4b19"

	// Convert display → stratum
	stratumHex, err := displayToStratumPrevHash(displayHex)
	if err != nil {
		t.Fatalf("displayToStratumPrevHash: %v", err)
	}

	// Convert stratum → internal
	internal, err := stratumPrevHashToInternal(stratumHex)
	if err != nil {
		t.Fatalf("stratumPrevHashToInternal: %v", err)
	}

	// Display → internal (direct) should match
	displayBytes, _ := hex.DecodeString(displayHex)
	expectedInternal := util.ReverseBytes(displayBytes)

	if !bytes.Equal(internal, expectedInternal) {
		t.Errorf("prevhash round-trip failed\n  got:      %s\n  expected: %s",
			hex.EncodeToString(internal),
			hex.EncodeToString(expectedInternal),
		)
	}
}

// TestHexBEToLE verifies big-endian hex to little-endian conversion.
func TestHexBEToLE(t *testing.T) {
	result, err := hexBEToLE("20000000", 4)
	if err != nil {
		t.Fatalf("hexBEToLE: %v", err)
	}
	expected := []byte{0x00, 0x00, 0x00, 0x20}
	if !bytes.Equal(result, expected) {
		t.Errorf("got %x, expected %x", result, expected)
	}
}
