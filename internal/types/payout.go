package types

// PayoutEntry represents a single payout in the coinbase transaction.
type PayoutEntry struct {
	Address string `json:"address"`
	Amount  int64  `json:"amount"` // Satoshis
}

// BlockTemplateData is an intermediate representation of a block template
// used by the work generator.
type BlockTemplateData struct {
	Height            int64
	PrevBlockHash     string
	Version           string // hex
	Bits              string // hex compact target
	CurTime           string // hex timestamp
	CoinbaseValue     int64
	WitnessCommitment string // hex
	Network           string
	TxHashes          []string // transaction hashes (hex)
}
