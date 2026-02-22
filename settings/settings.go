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

type fileData struct {
	ActiveModel string            `json:"active_model"`
	APIKeys     map[string]string `json:"api_keys"`
}

type Settings struct {
	mu          sync.Mutex
	activeModel string
	apiKeys     map[string]string
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
		dir:     d,
		apiKeys: make(map[string]string),
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
	if fd.APIKeys != nil {
		s.apiKeys = fd.APIKeys
	}
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
		APIKeys:     s.apiKeys,
	}
	data, err := json.MarshalIndent(fd, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, settingsFile), data, 0o644)
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
func (s *Settings) APIKey(keyName string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.apiKeys[keyName]
}

// SetAPIKey stores an API key under the given provider key name and persists to disk.
func (s *Settings) SetAPIKey(keyName, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apiKeys[keyName] = key
	return s.saveLocked()
}
