package bitcoin

import (
	"context"
	"sync"
)

// MockRPC implements BitcoinRPC for testing.
type MockRPC struct {
	mu sync.Mutex

	BlockTemplate   *BlockTemplate
	BlockCount      int64
	BestBlockHash   string
	SubmittedBlocks []string

	// Error overrides
	GetBlockTemplateErr error
	SubmitBlockErr      error
	GetBlockCountErr    error
	GetBestBlockHashErr error
}

// NewMockRPC creates a new mock Bitcoin RPC client with sensible defaults.
func NewMockRPC() *MockRPC {
	return &MockRPC{
		BlockTemplate: &BlockTemplate{
			Version:           536870912,
			PreviousBlockHash: "0000000000000003fa0d845513ea5014a7859d411f5f4a91eaab24eb47a18f39",
			Transactions:      []TemplateTransaction{},
			CoinbaseValue:     5000000000,
			Target:            "00000000ffff0000000000000000000000000000000000000000000000000000",
			CurTime:           1700000000,
			Bits:              "1d00ffff",
			Height:            800000,
		},
		BlockCount:    799999,
		BestBlockHash: "0000000000000003fa0d845513ea5014a7859d411f5f4a91eaab24eb47a18f39",
	}
}

func (m *MockRPC) GetBlockTemplate(_ context.Context) (*BlockTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.GetBlockTemplateErr != nil {
		return nil, m.GetBlockTemplateErr
	}
	return m.BlockTemplate, nil
}

func (m *MockRPC) SubmitBlock(_ context.Context, blockHex string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SubmitBlockErr != nil {
		return m.SubmitBlockErr
	}
	m.SubmittedBlocks = append(m.SubmittedBlocks, blockHex)
	return nil
}

func (m *MockRPC) GetBlockCount(_ context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.GetBlockCountErr != nil {
		return 0, m.GetBlockCountErr
	}
	return m.BlockCount, nil
}

func (m *MockRPC) GetBestBlockHash(_ context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.GetBestBlockHashErr != nil {
		return "", m.GetBestBlockHashErr
	}
	return m.BestBlockHash, nil
}
