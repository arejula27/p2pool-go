package pplns

import (
	"math/big"

	"github.com/djkazic/p2pool-go/internal/types"
	"github.com/djkazic/p2pool-go/pkg/util"
)

// Window represents the PPLNS sliding window of shares.
type Window struct {
	shares    []*types.Share
	maxTarget *big.Int
}

// NewWindow creates a new PPLNS window from a list of shares (newest first).
func NewWindow(shares []*types.Share, maxTarget *big.Int) *Window {
	return &Window{
		shares:    shares,
		maxTarget: maxTarget,
	}
}

// ShareWeight returns the weight (difficulty) of a single share.
// Weight = maxTarget / shareTarget (i.e., the share's difficulty).
func (w *Window) ShareWeight(share *types.Share) *big.Int {
	if share.ShareTarget == nil || share.ShareTarget.Sign() == 0 {
		return big.NewInt(1)
	}
	return new(big.Int).Div(w.maxTarget, share.ShareTarget)
}

// MinerWeights returns a map of miner address -> total weight in the window.
func (w *Window) MinerWeights() map[string]*big.Int {
	weights := make(map[string]*big.Int)

	for _, share := range w.shares {
		weight := w.ShareWeight(share)
		addr := share.MinerAddress
		if existing, ok := weights[addr]; ok {
			existing.Add(existing, weight)
		} else {
			weights[addr] = new(big.Int).Set(weight)
		}
	}

	return weights
}

// TotalWeight returns the total weight of all shares in the window.
func (w *Window) TotalWeight() *big.Int {
	total := new(big.Int)
	for _, share := range w.shares {
		total.Add(total, w.ShareWeight(share))
	}
	return total
}

// ShareCount returns the number of shares in the window.
func (w *Window) ShareCount() int {
	return len(w.shares)
}

// MaxTarget returns the max target used for weight calculations.
func (w *Window) MaxTarget() *big.Int {
	return w.maxTarget
}

// DefaultMaxTarget returns the sharechain max target.
func DefaultMaxTarget() *big.Int {
	return util.CompactToTarget(0x207fffff)
}
