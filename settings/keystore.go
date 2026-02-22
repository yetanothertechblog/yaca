package settings

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/zalando/go-keyring"
)

const keyringService = "yaca"

// KeyStore provides secure storage for API keys with multiple backends.
// Priority: OS Keychain > Environment Variables > In-memory cache
type KeyStore struct {
	mu       sync.Mutex
	fallback map[string]string // In-memory cache when keyring unavailable
	disabled bool              // True if keyring is known to be unavailable
}

// NewKeyStore creates a new KeyStore instance.
func NewKeyStore() *KeyStore {
	return &KeyStore{
		fallback: make(map[string]string),
	}
}

// Get retrieves an API key. It checks:
// 1. Environment variable (YACA_<KEYNAME>_API_KEY)
// 2. OS keychain
// 3. In-memory cache (session only)
func (k *KeyStore) Get(keyName string) string {
	k.mu.Lock()
	defer k.mu.Unlock()

	// 1. Check environment variable first (for CI/CD)
	envKey := fmt.Sprintf("YACA_%s_API_KEY", keyName)
	if val := os.Getenv(envKey); val != "" {
		return val
	}

	// 2. Try OS keychain
	if !k.disabled {
		secret, err := keyring.Get(keyringService, keyName)
		if err == nil {
			return secret
		}
		// Mark keyring as disabled if it's not available (e.g., headless environment)
		if errors.Is(err, keyring.ErrNotFound) {
			// Key doesn't exist, not a keyring issue
		} else {
			// Keyring is unavailable (no GUI, no keyring daemon, etc.)
			k.disabled = true
		}
	}

	// 3. Check fallback storage (in-memory cache)
	return k.fallback[keyName]
}

// Set stores an API key securely.
// Priority: OS keychain > Fallback with warning
func (k *KeyStore) Set(keyName, value string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Try OS keychain first
	if !k.disabled {
		err := keyring.Set(keyringService, keyName, value)
		if err == nil {
			// Also update fallback in case keyring becomes unavailable later
			k.fallback[keyName] = value
			return nil
		}
		// If keyring fails, continue with fallback but warn
		k.disabled = true
	}

	// Fallback: store in memory (will be lost on restart without env vars)
	// This is a degraded mode - user should set env vars or fix keyring
	k.fallback[keyName] = value
	return fmt.Errorf("keyring unavailable, key stored in memory only (will not persist). Set %s environment variable for persistence", fmt.Sprintf("YACA_%s_API_KEY", keyName))
}

// Delete removes an API key from all storage backends.
func (k *KeyStore) Delete(keyName string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Remove from fallback
	delete(k.fallback, keyName)

	// Remove from keychain
	if !k.disabled {
		err := keyring.Delete(keyringService, keyName)
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return err
		}
	}
	return nil
}

// IsKeyringAvailable returns true if the OS keychain is accessible.
func (k *KeyStore) IsKeyringAvailable() bool {
	return !k.disabled
}
