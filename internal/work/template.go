package work

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/djkazic/p2pool-go/internal/bitcoin"
	"github.com/djkazic/p2pool-go/internal/types"
	"github.com/djkazic/p2pool-go/pkg/util"
)

// SplitCoinbase splits a coinbase transaction at the extranonce position.
// Returns coinbase1 (hex before extranonce) and coinbase2 (hex after extranonce).
func SplitCoinbase(coinbaseTx []byte, extranonceOffset int, extranonceSize int) (string, string) {
	coinbase1 := hex.EncodeToString(coinbaseTx[:extranonceOffset])
	coinbase2 := hex.EncodeToString(coinbaseTx[extranonceOffset+extranonceSize:])
	return coinbase1, coinbase2
}

// ComputeMerkleBranches computes the Merkle branches for the Stratum protocol.
// txHashes are the hashes of all transactions (excluding coinbase) as hex strings.
func ComputeMerkleBranches(txHashes []string) ([]string, error) {
	if len(txHashes) == 0 {
		return []string{}, nil
	}

	// Convert hex strings to byte slices
	hashes := make([][]byte, len(txHashes))
	for i, h := range txHashes {
		b, err := hex.DecodeString(h)
		if err != nil {
			return nil, fmt.Errorf("invalid tx hash at index %d: %w", i, err)
		}
		hashes[i] = b
	}

	// Build Merkle branches — the sibling path from the coinbase leaf to the root.
	// At each level, hashes[0] is the sibling of the coinbase-path node.
	// The remaining hashes (index 1+) are paired for the next level.
	var branches []string
	for len(hashes) > 0 {
		branches = append(branches, hex.EncodeToString(hashes[0]))
		if len(hashes) == 1 {
			break
		}

		// Pair up remaining hashes (excluding the sibling we just took)
		remaining := hashes[1:]
		var newHashes [][]byte
		for i := 0; i < len(remaining); i += 2 {
			left := remaining[i]
			var right []byte
			if i+1 < len(remaining) {
				right = remaining[i+1]
			} else {
				right = left // duplicate last element for odd count
			}
			combined := append(left, right...)
			hash := util.DoubleSHA256(combined)
			newHashes = append(newHashes, hash[:])
		}
		hashes = newHashes
	}

	return branches, nil
}

// ComputeMerkleRoot computes the Merkle root given the coinbase hash and branches.
// This is what miners do to reconstruct the full Merkle root.
func ComputeMerkleRoot(coinbaseHash []byte, branches []string) ([]byte, error) {
	current := make([]byte, len(coinbaseHash))
	copy(current, coinbaseHash)

	for _, branch := range branches {
		branchBytes, err := hex.DecodeString(branch)
		if err != nil {
			return nil, fmt.Errorf("invalid branch hash: %w", err)
		}
		combined := append(current, branchBytes...)
		hash := util.DoubleSHA256(combined)
		current = hash[:]
	}

	return current, nil
}

// BuildJobFromTemplate creates a Stratum job from a block template and payouts.
func BuildJobFromTemplate(
	jobID string,
	tmpl *types.BlockTemplateData,
	payouts []types.PayoutEntry,
	prevShareHash [32]byte,
	extranonceSize int,
) (*JobData, error) {
	// Build coinbase
	builder := types.NewCoinbaseBuilder(tmpl.Network)
	commitment := types.BuildShareCommitment(prevShareHash)

	coinbaseTx, extranonceOffset, err := builder.BuildCoinbase(
		tmpl.Height,
		commitment,
		payouts,
		tmpl.WitnessCommitment,
		extranonceSize,
	)
	if err != nil {
		return nil, fmt.Errorf("build coinbase: %w", err)
	}

	// Split coinbase at extranonce position
	coinbase1, coinbase2 := SplitCoinbase(coinbaseTx, extranonceOffset, extranonceSize)

	// Compute Merkle branches from template transactions
	branches, err := ComputeMerkleBranches(tmpl.TxHashes)
	if err != nil {
		return nil, fmt.Errorf("compute merkle branches: %w", err)
	}

	// Convert prevhash from display order to Stratum v1 format
	prevHashStratum, err := displayToStratumPrevHash(tmpl.PrevBlockHash)
	if err != nil {
		return nil, fmt.Errorf("convert prevhash to stratum format: %w", err)
	}

	return &JobData{
		ID:               jobID,
		PrevBlockHash:    prevHashStratum,
		Coinbase1:        coinbase1,
		Coinbase2:        coinbase2,
		CoinbaseTx:       coinbaseTx,
		ExtranonceOffset: extranonceOffset,
		MerkleBranches:   branches,
		Version:          tmpl.Version,
		NBits:            tmpl.Bits,
		NTime:            tmpl.CurTime,
		Height:           tmpl.Height,
	}, nil
}

// JobData contains the full job data including internal fields not sent to miners.
type JobData struct {
	ID               string
	Seq              uint64
	PrevBlockHash    string
	Coinbase1        string
	Coinbase2        string
	CoinbaseTx       []byte
	ExtranonceOffset int
	MerkleBranches   []string
	Version          string
	NBits            string
	NTime            string
	Height           int64
	CleanJobs        bool                   // true for new block, false for refresh
	Template         *bitcoin.BlockTemplate // template used to build this job
}

// ReconstructHeader rebuilds the 80-byte block header and coinbase from a job
// and the miner's submission parameters. Returns (header, coinbaseBytes, error).
//
// The version parameter is the actual version to use (after applying any BIP 310
// version rolling bits). The 4-byte fields (version, nbits, ntime, nonce) are
// big-endian hex, reversed to little-endian for the header. The prevhash is in
// Stratum v1 format (4-byte-word-swapped internal order) and decoded accordingly.
func ReconstructHeader(job *JobData, version, extranonce1, extranonce2, ntime, nonce string) ([]byte, []byte, error) {
	// 1. Reconstruct full coinbase transaction
	coinbaseHex := job.Coinbase1 + extranonce1 + extranonce2 + job.Coinbase2
	coinbaseBytes, err := hex.DecodeString(coinbaseHex)
	if err != nil {
		return nil, nil, fmt.Errorf("decode coinbase hex: %w", err)
	}

	// 2. Hash coinbase (double-SHA256)
	coinbaseHash := util.DoubleSHA256(coinbaseBytes)

	// 3. Compute Merkle root from coinbase hash + branches
	merkleRoot, err := ComputeMerkleRoot(coinbaseHash[:], job.MerkleBranches)
	if err != nil {
		return nil, nil, fmt.Errorf("compute merkle root: %w", err)
	}

	// 4. Decode header fields
	versionBytes, err := hexBEToLE(version, 4)
	if err != nil {
		return nil, nil, fmt.Errorf("decode version: %w", err)
	}

	prevHashBytes, err := stratumPrevHashToInternal(job.PrevBlockHash)
	if err != nil {
		return nil, nil, fmt.Errorf("decode prevhash: %w", err)
	}

	ntimeBytes, err := hexBEToLE(ntime, 4)
	if err != nil {
		return nil, nil, fmt.Errorf("decode ntime: %w", err)
	}

	nbitsBytes, err := hexBEToLE(job.NBits, 4)
	if err != nil {
		return nil, nil, fmt.Errorf("decode nbits: %w", err)
	}

	nonceBytes, err := hexBEToLE(nonce, 4)
	if err != nil {
		return nil, nil, fmt.Errorf("decode nonce: %w", err)
	}

	// 5. Build 80-byte block header
	header := make([]byte, 80)
	copy(header[0:4], versionBytes)
	copy(header[4:36], prevHashBytes)
	copy(header[36:68], merkleRoot)
	copy(header[68:72], ntimeBytes)
	copy(header[72:76], nbitsBytes)
	copy(header[76:80], nonceBytes)

	return header, coinbaseBytes, nil
}

// ReconstructBlock builds the full serialized block for submission to bitcoind.
// It combines the header, coinbase transaction, and all transactions from the
// block template. The coinbase is wrapped with segwit witness data for submission.
func ReconstructBlock(header []byte, coinbase []byte, tmpl *bitcoin.BlockTemplate) (string, error) {
	var buf bytes.Buffer

	// Block header (80 bytes)
	buf.Write(header)

	// Transaction count (coinbase + template transactions)
	txCount := 1 + len(tmpl.Transactions)
	buf.Write(util.WriteVarInt(uint64(txCount)))

	// Coinbase transaction (add witness data for block submission)
	witnessCoinbase := types.AddCoinbaseWitness(coinbase)
	buf.Write(witnessCoinbase)

	// Remaining transactions from the template
	for _, tx := range tmpl.Transactions {
		txBytes, err := hex.DecodeString(tx.Data)
		if err != nil {
			return "", fmt.Errorf("decode template tx %s: %w", tx.TxID, err)
		}
		buf.Write(txBytes)
	}

	return hex.EncodeToString(buf.Bytes()), nil
}

// ComputeFullMerkleRoot builds the merkle root from a list of txid hashes (internal
// byte order). This is used for pre-submission verification — it independently
// computes the merkle root using the standard Bitcoin algorithm (not branches).
func ComputeFullMerkleRoot(txids [][]byte) []byte {
	if len(txids) == 0 {
		return nil
	}

	// Copy so we don't mutate the caller's slice
	hashes := make([][]byte, len(txids))
	for i, h := range txids {
		c := make([]byte, len(h))
		copy(c, h)
		hashes[i] = c
	}

	for len(hashes) > 1 {
		if len(hashes)%2 != 0 {
			dup := make([]byte, len(hashes[len(hashes)-1]))
			copy(dup, hashes[len(hashes)-1])
			hashes = append(hashes, dup)
		}
		var newLevel [][]byte
		for i := 0; i < len(hashes); i += 2 {
			combined := append(hashes[i], hashes[i+1]...)
			h := util.DoubleSHA256(combined)
			newLevel = append(newLevel, h[:])
		}
		hashes = newLevel
	}

	return hashes[0]
}

// VerifyMerkleRoot independently computes the expected merkle root from the
// non-witness coinbase and the template transactions, and compares it with the
// merkle root stored in the 80-byte block header. Returns nil if they match,
// or a detailed error describing the mismatch.
func VerifyMerkleRoot(header []byte, coinbase []byte, tmpl *bitcoin.BlockTemplate) error {
	if len(header) < 68 {
		return fmt.Errorf("header too short: %d bytes", len(header))
	}

	// Extract merkle root from header (bytes 36..68, internal byte order)
	headerMerkleRoot := header[36:68]

	// Compute coinbase txid (DoubleSHA256 of non-witness serialization)
	cbHash := util.DoubleSHA256(coinbase)

	// Collect all txids: coinbase first, then template transactions
	txids := make([][]byte, 1+len(tmpl.Transactions))
	txids[0] = cbHash[:]

	for i, tx := range tmpl.Transactions {
		b, err := hex.DecodeString(tx.TxID)
		if err != nil {
			return fmt.Errorf("invalid txid at index %d: %w", i, err)
		}
		// TxID from getblocktemplate is display order (reversed) — convert to internal
		txids[i+1] = util.ReverseBytes(b)
	}

	// Compute expected merkle root
	expectedRoot := ComputeFullMerkleRoot(txids)

	if !bytes.Equal(headerMerkleRoot, expectedRoot) {
		return fmt.Errorf(
			"merkle root mismatch: header=%s expected=%s coinbase_txid=%s tx_count=%d",
			hex.EncodeToString(headerMerkleRoot),
			hex.EncodeToString(expectedRoot),
			hex.EncodeToString(cbHash[:]),
			len(tmpl.Transactions),
		)
	}

	return nil
}

// hexBEToLE decodes a big-endian hex string and reverses it to little-endian byte order.
func hexBEToLE(hexStr string, expectedLen int) ([]byte, error) {
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex %q: %w", hexStr, err)
	}
	if len(b) != expectedLen {
		return nil, fmt.Errorf("expected %d bytes, got %d", expectedLen, len(b))
	}
	return util.ReverseBytes(b), nil
}

// displayToStratumPrevHash converts a block hash from display order (big-endian,
// as returned by getblocktemplate) to Stratum v1 prevhash format.
// Stratum prevhash = internal byte order with each 4-byte word byte-swapped.
// The miner byte-swaps each word back to recover the internal order for the header.
func displayToStratumPrevHash(displayHex string) (string, error) {
	b, err := hex.DecodeString(displayHex)
	if err != nil {
		return "", fmt.Errorf("invalid hex: %w", err)
	}
	if len(b) != 32 {
		return "", fmt.Errorf("expected 32 bytes, got %d", len(b))
	}
	// Display → internal (full byte reverse)
	internal := util.ReverseBytes(b)
	// Internal → stratum (swap each 4-byte word)
	swapWords4(internal)
	return hex.EncodeToString(internal), nil
}

// stratumPrevHashToInternal converts a Stratum v1 prevhash hex string to the
// 32-byte internal byte order used in the Bitcoin block header.
func stratumPrevHashToInternal(stratumHex string) ([]byte, error) {
	b, err := hex.DecodeString(stratumHex)
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}
	if len(b) != 32 {
		return nil, fmt.Errorf("expected 32 bytes, got %d", len(b))
	}
	// Stratum → internal (swap each 4-byte word)
	swapWords4(b)
	return b, nil
}

// swapWords4 byte-swaps each 4-byte word in a byte slice in place.
func swapWords4(b []byte) {
	for i := 0; i < len(b)-3; i += 4 {
		b[i], b[i+3] = b[i+3], b[i]
		b[i+1], b[i+2] = b[i+2], b[i+1]
	}
}
