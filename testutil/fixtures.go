package testutil

import (
	"math/big"

	"github.com/djkazic/p2pool-go/internal/bitcoin"
	"github.com/djkazic/p2pool-go/internal/types"
	"github.com/djkazic/p2pool-go/pkg/util"
)

// SampleBlockTemplate returns a minimal block template for testing.
func SampleBlockTemplate() *bitcoin.BlockTemplate {
	return &bitcoin.BlockTemplate{
		Version:           536870912,
		PreviousBlockHash: "0000000000000003fa0d845513ea5014a7859d411f5f4a91eaab24eb47a18f39",
		Transactions:      []bitcoin.TemplateTransaction{},
		CoinbaseValue:     5000000000,
		Target:            "00000000ffff0000000000000000000000000000000000000000000000000000",
		CurTime:           1700000000,
		Bits:              "1d00ffff",
		Height:            800000,
	}
}

// SampleShare creates a sample share for testing.
func SampleShare(nonce uint32, prevShareHash [32]byte) *types.Share {
	return &types.Share{
		Header: types.ShareHeader{
			Version:   536870912,
			Timestamp: 1700000000,
			Bits:      0x1d00ffff,
			Nonce:     nonce,
		},
		ShareVersion:  1,
		PrevShareHash: prevShareHash,
		ShareTarget:   util.CompactToTarget(0x1d00ffff),
		MinerAddress:  "tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx",
	}
}

// SampleShareChain creates a linear chain of test shares.
func SampleShareChain(count int) []*types.Share {
	shares := make([]*types.Share, count)
	var prevHash [32]byte // Genesis has zero prev

	for i := 0; i < count; i++ {
		s := SampleShare(uint32(i), prevHash)
		shares[i] = s
		prevHash = s.Hash()
	}

	return shares
}

// EasyTarget returns a very easy target for testing (any hash will pass).
func EasyTarget() *big.Int {
	return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
}
