package sharechain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBoltStore_AddAndGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewBoltStore(filepath.Join(dir, "test.db"), testLogger())
	if err != nil {
		t.Fatalf("NewBoltStore: %v", err)
	}
	defer store.Close()

	share := makeTestShare([32]byte{}, testMiner1, 1700000000)
	hash := share.Hash()

	if err := store.Add(share); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, ok := store.Get(hash)
	if !ok {
		t.Fatal("share not found after Add")
	}
	if got.MinerAddress != testMiner1 {
		t.Errorf("miner address = %s, want miner1", got.MinerAddress)
	}
	if got.Header.Nonce != share.Header.Nonce {
		t.Errorf("nonce = %d, want %d", got.Header.Nonce, share.Header.Nonce)
	}
	if store.Count() != 1 {
		t.Errorf("count = %d, want 1", store.Count())
	}
}

func TestBoltStore_DuplicateAdd(t *testing.T) {
	dir := t.TempDir()
	store, err := NewBoltStore(filepath.Join(dir, "test.db"), testLogger())
	if err != nil {
		t.Fatalf("NewBoltStore: %v", err)
	}
	defer store.Close()

	share := makeTestShare([32]byte{}, testMiner1, 1700000000)
	_ = store.Add(share)
	err = store.Add(share)
	if err == nil {
		t.Error("expected error on duplicate add")
	}
}

func TestBoltStore_Tip(t *testing.T) {
	dir := t.TempDir()
	store, err := NewBoltStore(filepath.Join(dir, "test.db"), testLogger())
	if err != nil {
		t.Fatalf("NewBoltStore: %v", err)
	}
	defer store.Close()

	_, ok := store.Tip()
	if ok {
		t.Error("empty store should not have tip")
	}

	share := makeTestShare([32]byte{}, testMiner1, 1700000000)
	hash := share.Hash()
	_ = store.Add(share)
	_ = store.SetTip(hash)

	tip, ok := store.Tip()
	if !ok {
		t.Fatal("tip not found after SetTip")
	}
	if tip.Hash() != hash {
		t.Error("tip hash mismatch")
	}
}

func TestBoltStore_GetAncestors(t *testing.T) {
	dir := t.TempDir()
	store, err := NewBoltStore(filepath.Join(dir, "test.db"), testLogger())
	if err != nil {
		t.Fatalf("NewBoltStore: %v", err)
	}
	defer store.Close()

	var prevHash [32]byte
	for i := 0; i < 5; i++ {
		share := makeTestShare(prevHash, testMiner1, uint32(1700000000+i*30))
		_ = store.Add(share)
		prevHash = share.Hash()
	}
	_ = store.SetTip(prevHash)

	ancestors := store.GetAncestors(prevHash, 10)
	if len(ancestors) != 5 {
		t.Errorf("got %d ancestors, want 5", len(ancestors))
	}
}

func TestBoltStore_PersistenceAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Phase 1: create store, add shares, set tip, close.
	var tipHash [32]byte
	{
		store, err := NewBoltStore(dbPath, testLogger())
		if err != nil {
			t.Fatalf("NewBoltStore (phase 1): %v", err)
		}

		var prevHash [32]byte
		for i := 0; i < 5; i++ {
			share := makeTestShare(prevHash, testMiner1, uint32(1700000000+i*30))
			if err := store.Add(share); err != nil {
				t.Fatalf("Add %d: %v", i, err)
			}
			prevHash = share.Hash()
		}
		tipHash = prevHash
		if err := store.SetTip(tipHash); err != nil {
			t.Fatalf("SetTip: %v", err)
		}

		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}

	// Phase 2: reopen and verify everything survived.
	{
		store, err := NewBoltStore(dbPath, testLogger())
		if err != nil {
			t.Fatalf("NewBoltStore (phase 2): %v", err)
		}
		defer store.Close()

		if store.Count() != 5 {
			t.Errorf("count after reopen = %d, want 5", store.Count())
		}

		tip, ok := store.Tip()
		if !ok {
			t.Fatal("tip not found after reopen")
		}
		if tip.Hash() != tipHash {
			t.Error("tip hash mismatch after reopen")
		}

		// Verify share data integrity.
		if tip.MinerAddress != testMiner1 {
			t.Errorf("miner address = %s, want miner1", tip.MinerAddress)
		}
		if tip.ShareTarget == nil || tip.ShareTarget.Sign() == 0 {
			t.Error("share target not restored")
		}

		// Verify chain walking works.
		ancestors := store.GetAncestors(tipHash, 10)
		if len(ancestors) != 5 {
			t.Errorf("ancestors after reopen = %d, want 5", len(ancestors))
		}
	}

	// Verify the db file actually exists on disk.
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file does not exist")
	}
}
