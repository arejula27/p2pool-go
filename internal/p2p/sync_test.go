package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"go.uber.org/zap"
)

// newTestHost creates a libp2p host on an ephemeral local port for testing.
func newTestHost(t *testing.T) host.Host {
	t.Helper()
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("create test host: %v", err)
	}
	t.Cleanup(func() { h.Close() })
	return h
}

// connectHosts connects host B to host A.
func connectHosts(t *testing.T, a, b host.Host) {
	t.Helper()
	aInfo := peer.AddrInfo{ID: a.ID(), Addrs: a.Addrs()}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := b.Connect(ctx, aInfo); err != nil {
		t.Fatalf("connect hosts: %v", err)
	}
}

func TestSyncProtocol_RoundTrip(t *testing.T) {
	logger := zap.NewNop()

	hostA := newTestHost(t)
	hostB := newTestHost(t)

	// Canned shares for host A to serve
	cannedShares := []ShareMsg{
		{
			Type:         MsgTypeShare,
			Version:      536870912,
			Timestamp:    1700000000,
			Bits:         0x1d00ffff,
			Nonce:        100,
			ShareVersion: 1,
			MinerAddress: "tb1qtest1",
		},
		{
			Type:         MsgTypeShare,
			Version:      536870912,
			Timestamp:    1700000030,
			Bits:         0x1d00ffff,
			Nonce:        200,
			ShareVersion: 1,
			MinerAddress: "tb1qtest2",
		},
	}
	cannedShares[1].PrevShareHash[0] = 0xaa

	// Host A serves shares — handler returns canned shares regardless of locators
	NewSyncer(hostA, func(req *ShareLocatorReq) *ShareLocatorResp {
		return &ShareLocatorResp{
			Type:   MsgTypeLocatorResp,
			Shares: cannedShares,
		}
	}, logger)

	// Host B creates a syncer to request from A
	syncerB := NewSyncer(hostB, func(req *ShareLocatorReq) *ShareLocatorResp {
		return nil
	}, logger)

	connectHosts(t, hostA, hostB)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := syncerB.RequestLocator(ctx, hostA.ID(), nil, 100)
	if err != nil {
		t.Fatalf("RequestLocator: %v", err)
	}

	if len(resp.Shares) != 2 {
		t.Fatalf("expected 2 shares, got %d", len(resp.Shares))
	}

	if resp.Shares[0].MinerAddress != "tb1qtest1" {
		t.Errorf("share[0] miner = %q, want tb1qtest1", resp.Shares[0].MinerAddress)
	}
	if resp.Shares[1].MinerAddress != "tb1qtest2" {
		t.Errorf("share[1] miner = %q, want tb1qtest2", resp.Shares[1].MinerAddress)
	}
	if resp.Shares[1].PrevShareHash[0] != 0xaa {
		t.Errorf("share[1] PrevShareHash[0] = %x, want aa", resp.Shares[1].PrevShareHash[0])
	}
}

func TestSyncProtocol_EmptyChain(t *testing.T) {
	logger := zap.NewNop()

	hostA := newTestHost(t)
	hostB := newTestHost(t)

	// Host A has an empty chain — returns empty response
	NewSyncer(hostA, func(req *ShareLocatorReq) *ShareLocatorResp {
		return &ShareLocatorResp{
			Type:   MsgTypeLocatorResp,
			Shares: nil,
		}
	}, logger)

	syncerB := NewSyncer(hostB, func(req *ShareLocatorReq) *ShareLocatorResp {
		return nil
	}, logger)

	connectHosts(t, hostA, hostB)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := syncerB.RequestLocator(ctx, hostA.ID(), nil, 100)
	if err != nil {
		t.Fatalf("RequestLocator: %v", err)
	}

	if len(resp.Shares) != 0 {
		t.Errorf("expected 0 shares, got %d", len(resp.Shares))
	}
}

func TestSyncProtocol_BatchSizeLimit(t *testing.T) {
	logger := zap.NewNop()

	hostA := newTestHost(t)
	hostB := newTestHost(t)

	// Handler checks that MaxCount was clamped to maxSyncBatchSize
	var receivedMaxCount int
	NewSyncer(hostA, func(req *ShareLocatorReq) *ShareLocatorResp {
		receivedMaxCount = req.MaxCount
		return &ShareLocatorResp{Type: MsgTypeLocatorResp}
	}, logger)

	syncerB := NewSyncer(hostB, func(req *ShareLocatorReq) *ShareLocatorResp {
		return nil
	}, logger)

	connectHosts(t, hostA, hostB)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Request more than maxSyncBatchSize
	_, err := syncerB.RequestLocator(ctx, hostA.ID(), nil, 500)
	if err != nil {
		t.Fatalf("RequestLocator: %v", err)
	}

	if receivedMaxCount != maxSyncBatchSize {
		t.Errorf("MaxCount = %d, want %d (clamped)", receivedMaxCount, maxSyncBatchSize)
	}
}

func TestSyncProtocol_LocatorForkPoint(t *testing.T) {
	logger := zap.NewNop()

	hostA := newTestHost(t)
	hostB := newTestHost(t)

	// Server has chain A→B→C→D. Build 4 shares with linked PrevShareHash.
	hashA := [32]byte{0x01}
	hashB := [32]byte{0x02}
	hashC := [32]byte{0x03}
	hashD := [32]byte{0x04}

	shareA := ShareMsg{Type: MsgTypeShare, MinerAddress: "A", ShareVersion: 1, Nonce: 1}
	shareB := ShareMsg{Type: MsgTypeShare, MinerAddress: "B", ShareVersion: 1, Nonce: 2, PrevShareHash: hashA}
	shareC := ShareMsg{Type: MsgTypeShare, MinerAddress: "C", ShareVersion: 1, Nonce: 3, PrevShareHash: hashB}
	shareD := ShareMsg{Type: MsgTypeShare, MinerAddress: "D", ShareVersion: 1, Nonce: 4, PrevShareHash: hashC}

	// Map hash→share and build main chain order
	chain := map[[32]byte]ShareMsg{
		hashA: shareA,
		hashB: shareB,
		hashC: shareC,
		hashD: shareD,
	}
	mainChainOrder := [][32]byte{hashA, hashB, hashC, hashD} // oldest-first

	// Host A: find fork point from locators, return shares after it
	NewSyncer(hostA, func(req *ShareLocatorReq) *ShareLocatorResp {
		// Find fork point
		forkIdx := -1
		for _, loc := range req.Locators {
			for i, h := range mainChainOrder {
				if h == loc {
					forkIdx = i
					break
				}
			}
			if forkIdx >= 0 {
				break
			}
		}

		// If no locator matches, start from genesis
		startIdx := 0
		if forkIdx >= 0 {
			startIdx = forkIdx + 1 // after the fork point
		}

		var shares []ShareMsg
		for i := startIdx; i < len(mainChainOrder); i++ {
			shares = append(shares, chain[mainChainOrder[i]])
		}

		return &ShareLocatorResp{
			Type:   MsgTypeLocatorResp,
			Shares: shares,
		}
	}, logger)

	syncerB := NewSyncer(hostB, func(req *ShareLocatorReq) *ShareLocatorResp {
		return nil
	}, logger)

	connectHosts(t, hostA, hostB)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Client sends locator [B] — should get [C, D] back
	resp, err := syncerB.RequestLocator(ctx, hostA.ID(), [][32]byte{hashB}, 100)
	if err != nil {
		t.Fatalf("RequestLocator: %v", err)
	}

	if len(resp.Shares) != 2 {
		t.Fatalf("expected 2 shares (C, D), got %d", len(resp.Shares))
	}

	if resp.Shares[0].MinerAddress != "C" {
		t.Errorf("share[0] miner = %q, want C", resp.Shares[0].MinerAddress)
	}
	if resp.Shares[1].MinerAddress != "D" {
		t.Errorf("share[1] miner = %q, want D", resp.Shares[1].MinerAddress)
	}
}
