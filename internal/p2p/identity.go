package p2p

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/crypto"
)

const identityKeyFile = "identity.key"

// LoadOrCreateIdentity loads a persistent libp2p identity key from dataDir,
// or generates and saves a new one if none exists. This ensures the node's
// peer ID is stable across restarts — critical for bootnode addresses.
func LoadOrCreateIdentity(dataDir string) (crypto.PrivKey, error) {
	keyPath := filepath.Join(dataDir, identityKeyFile)

	// Try to load existing key
	data, err := os.ReadFile(keyPath)
	if err == nil {
		key, err := crypto.UnmarshalPrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("unmarshal identity key: %w", err)
		}
		return key, nil
	}

	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read identity key: %w", err)
	}

	// Generate new key
	key, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate identity key: %w", err)
	}

	// Persist it
	raw, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshal identity key: %w", err)
	}

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	if err := os.WriteFile(keyPath, raw, 0600); err != nil {
		return nil, fmt.Errorf("write identity key: %w", err)
	}

	return key, nil
}
