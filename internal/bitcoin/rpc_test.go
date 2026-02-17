package bitcoin

import (
	"context"
	"fmt"
	"testing"
)

func TestMockRPC_GetBlockTemplate(t *testing.T) {
	mock := NewMockRPC()
	ctx := context.Background()

	tmpl, err := mock.GetBlockTemplate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl.Height != 800000 {
		t.Errorf("height = %d, want 800000", tmpl.Height)
	}
	if tmpl.CoinbaseValue != 5000000000 {
		t.Errorf("coinbase value = %d, want 5000000000", tmpl.CoinbaseValue)
	}
}

func TestMockRPC_GetBlockTemplate_Error(t *testing.T) {
	mock := NewMockRPC()
	mock.GetBlockTemplateErr = fmt.Errorf("connection refused")
	ctx := context.Background()

	_, err := mock.GetBlockTemplate(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestMockRPC_SubmitBlock(t *testing.T) {
	mock := NewMockRPC()
	ctx := context.Background()

	err := mock.SubmitBlock(ctx, "deadbeef")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.SubmittedBlocks) != 1 || mock.SubmittedBlocks[0] != "deadbeef" {
		t.Error("block not recorded")
	}
}

func TestMockRPC_GetBlockCount(t *testing.T) {
	mock := NewMockRPC()
	ctx := context.Background()

	count, err := mock.GetBlockCount(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 799999 {
		t.Errorf("block count = %d, want 799999", count)
	}
}

func TestMockRPC_GetBestBlockHash(t *testing.T) {
	mock := NewMockRPC()
	ctx := context.Background()

	hash, err := mock.GetBestBlockHash(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != mock.BestBlockHash {
		t.Errorf("hash mismatch")
	}
}

func TestRPCError(t *testing.T) {
	err := &RPCError{Code: -1, Message: "test error"}
	if err.Error() != "RPC error -1: test error" {
		t.Errorf("unexpected error string: %s", err.Error())
	}
}
