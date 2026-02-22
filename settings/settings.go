package settings

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

const settingsFile = ".yaca/settings.json"

// fileData is used for JSON persistence (API keys no longer stored in file)
type fileData struct {
	ActiveModel string `json:"active_model"`
}

type Settings struct {
	mu          sync.Mutex
	activeModel string
	keyStore    *KeyStore
	dir         string
}

func dir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

func Load() (*Settings, error) {
	d := dir()
	s := &Settings{
		dir:      d,
		keyStore: NewKeyStore(),
	}
	data, err := os.ReadFile(filepath.Join(d, settingsFile))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return s, nil
		}
		return s, err
	}
	var fd fileData
	if err := json.Unmarshal(data, &fd); err != nil {
		return s, err
	}
	s.activeModel = fd.ActiveModel
	return s, nil
}

func (s *Settings) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *Settings) saveLocked() error {
	dir := filepath.Join(s.dir, ".yaca")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	fd := fileData{
		ActiveModel: s.activeModel,
		// API keys are no longer stored in file - they go to keychain
	}
	data, err := json.MarshalIndent(fd, "", "  ")
	if err != nil {
		return err
	}
	// Use 0600 for settings file (contains user preferences)
	return os.WriteFile(filepath.Join(s.dir, settingsFile), data, 0o600)
}

func (s *Settings) ActiveModel() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeModel
}

func (s *Settings) SetActiveModel(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeModel = name
	return s.saveLocked()
}

// APIKey returns the stored API key for the given provider key name (e.g. "Z_API").
// Checks environment variables first, then OS keychain.
func (s *Settings) APIKey(keyName string) string {
	return s.keyStore.Get(keyName)
}

// SetAPIKey stores an API key securely in the OS keychain.
// Falls back to environment variable suggestion if keychain unavailable.
func (s *Settings) SetAPIKey(keyName, key string) error {
	return s.keyStore.Set(keyName, key)
}

// DeleteAPIKey removes an API key from secure storage.
func (s *Settings) DeleteAPIKey(keyName string) error {
	return s.keyStore.Delete(keyName)
}

// IsKeyringAvailable returns true if OS keychain is accessible.
func (s *Settings) IsKeyringAvailable() bool {
	return s.keyStore.IsKeyringAvailable()
}
