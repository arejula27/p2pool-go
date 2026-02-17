package sharechain

import (
	"math/big"
	"time"

	"github.com/djkazic/p2pool-go/internal/types"
	"github.com/djkazic/p2pool-go/pkg/util"
)

const (
	// DifficultyAdjustmentWindow is the number of shares to look back for difficulty adjustment.
	DifficultyAdjustmentWindow = 72 // ~36 minutes at 30s target

	// MinShareTarget prevents the difficulty from going too high (target too low).
	minShareTargetBits = 0x1d00ffff // Bitcoin difficulty 1

	// MaxShareTarget is the easiest possible share target (highest allowed value).
	// Uses regtest-style max target so CPU miners can produce shares.
	maxShareTargetBits = 0x207fffff
)

var (
	MinShareTarget = util.CompactToTarget(minShareTargetBits)
	MaxShareTarget = util.CompactToTarget(maxShareTargetBits)
)

// DifficultyCalculator adjusts sharechain difficulty.
type DifficultyCalculator struct {
	targetTime time.Duration
}

// NewDifficultyCalculator creates a new difficulty calculator.
func NewDifficultyCalculator(targetTime time.Duration) *DifficultyCalculator {
	return &DifficultyCalculator{
		targetTime: targetTime,
	}
}

// NextTarget calculates the next share target based on a window of recent shares.
// Uses: newTarget = currentTarget * (actualTime / expectedTime), clamped to 4x.
//
// The window is trimmed to only include shares whose target is within 4x of the
// newest share's target. During difficulty transitions (cold start, hashrate
// changes), the window may contain shares at wildly different difficulties.
// Including stale-difficulty shares distorts the timing data — e.g., 70 instant
// shares at MaxShareTarget would dominate the window average even after the
// algorithm has found the right difficulty, causing compounding overshoot or
// glacially slow convergence. Trimming ensures the algorithm uses only timing
// data from shares at a comparable difficulty level.
func (dc *DifficultyCalculator) NextTarget(shares []*types.Share) *big.Int {
	if len(shares) < 2 {
		return new(big.Int).Set(MaxShareTarget)
	}

	window := shares
	if len(window) > DifficultyAdjustmentWindow {
		window = window[:DifficultyAdjustmentWindow]
	}

	// window[0] is the most recent share, window[len-1] is the oldest
	newest := window[0]

	currentTarget := newest.ShareTarget
	if currentTarget == nil || currentTarget.Sign() == 0 {
		return new(big.Int).Set(MaxShareTarget)
	}

	// Trim window to shares with targets within 4x of the newest share.
	// This matches the 4x per-step clamp: shares more than 4x away are from
	// a different difficulty regime and their timing data is not comparable.
	upper := new(big.Int).Mul(currentTarget, big.NewInt(4))
	lower := new(big.Int).Div(currentTarget, big.NewInt(4))
	for i := 1; i < len(window); i++ {
		st := window[i].ShareTarget
		if st == nil || st.Sign() == 0 || st.Cmp(upper) > 0 || st.Cmp(lower) < 0 {
			window = window[:i]
			break
		}
	}

	if len(window) < 2 {
		// Not enough similar-difficulty shares for timing-based adjustment.
		// Return the newest share's target unchanged.
		return util.CompactToTarget(util.TargetToCompact(currentTarget))
	}

	oldest := window[len(window)-1]

	actualTime := int64(newest.Header.Timestamp) - int64(oldest.Header.Timestamp)
	if actualTime <= 0 {
		actualTime = 1
	}

	expectedTime := int64(dc.targetTime.Seconds()) * int64(len(window)-1)
	if expectedTime <= 0 {
		expectedTime = 1
	}

	// newTarget = currentTarget * actualTime / expectedTime
	newTarget := new(big.Int).Mul(currentTarget, big.NewInt(actualTime))
	newTarget.Div(newTarget, big.NewInt(expectedTime))

	// Clamp to 4x adjustment per calculation
	maxAdjust := new(big.Int).Mul(currentTarget, big.NewInt(4))
	minAdjust := new(big.Int).Div(currentTarget, big.NewInt(4))

	if newTarget.Cmp(maxAdjust) > 0 {
		newTarget.Set(maxAdjust)
	}
	if newTarget.Cmp(minAdjust) < 0 {
		newTarget.Set(minAdjust)
	}

	// Clamp to global limits
	if newTarget.Cmp(MaxShareTarget) > 0 {
		newTarget.Set(MaxShareTarget)
	}
	// Normalize through compact round-trip so all nodes produce identical
	// big.Int values regardless of whether a share was mined locally or
	// received via P2P (where targets are transmitted as compact uint32).
	return util.CompactToTarget(util.TargetToCompact(newTarget))
}
