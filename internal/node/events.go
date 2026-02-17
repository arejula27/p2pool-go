package node

import (
	"github.com/djkazic/p2pool-go/internal/p2p"
	"github.com/djkazic/p2pool-go/internal/sharechain"
	"github.com/djkazic/p2pool-go/internal/stratum"
	"github.com/djkazic/p2pool-go/internal/work"
)

// Event types for the orchestrator event loop.

// NewJobEvent signals that a new mining job is available.
type NewJobEvent struct {
	Job *work.JobData
}

// ShareSubmitEvent signals that a miner submitted a share.
type ShareSubmitEvent struct {
	Submission *stratum.ShareSubmission
}

// P2PShareEvent signals that a share was received from the P2P network.
type P2PShareEvent struct {
	Share *p2p.ShareMsg
}

// ChainEvent signals a sharechain state change.
type ChainEvent struct {
	Event sharechain.Event
}
