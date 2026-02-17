package types

import (
	"math/big"

	"github.com/djkazic/p2pool-go/pkg/util"
)

var (
	// TestnetMaxTarget is Bitcoin testnet's maximum target (difficulty 1).
	TestnetMaxTarget = util.CompactToTarget(0x1d00ffff)

	// MainnetMaxTarget is Bitcoin mainnet's maximum target (difficulty 1).
	MainnetMaxTarget = util.CompactToTarget(0x1d00ffff)

	// DefaultShareTarget is the initial sharechain target (much easier than Bitcoin).
	// This corresponds to a difficulty of ~1, appropriate for testnet CPU mining.
	DefaultShareTarget = util.CompactToTarget(0x1d00ffff)
)

// ShareDifficulty returns the share's difficulty relative to the max target.
func ShareDifficulty(share *Share, maxTarget *big.Int) float64 {
	if share.ShareTarget == nil || share.ShareTarget.Sign() == 0 {
		return 0
	}
	return util.TargetToDifficulty(share.ShareTarget, maxTarget)
}

// BitcoinDifficulty returns the Bitcoin difficulty from the share's nBits.
func BitcoinDifficulty(share *Share, maxTarget *big.Int) float64 {
	target := util.CompactToTarget(share.Header.Bits)
	return util.TargetToDifficulty(target, maxTarget)
}
