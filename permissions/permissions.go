package permissions

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const permFile = ".yaca/permissions.json"

type fileData struct {
	AlwaysAllow []string `json:"always_allow"`
}

type Permissions struct {
	mu      sync.Mutex
	entries []string
	dir     string
}

func Load(dir string) (*Permissions, error) {
	p := &Permissions{dir: dir}
	data, err := os.ReadFile(filepath.Join(dir, permFile))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return p, nil
		}
		return p, err
	}
	var fd fileData
	if err := json.Unmarshal(data, &fd); err != nil {
		return p, err
	}
	p.entries = fd.AlwaysAllow
	return p, nil
}

func (p *Permissions) Save() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.saveLocked()
}

func (p *Permissions) saveLocked() error {
	dir := filepath.Join(p.dir, ".yaca")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	fd := fileData{AlwaysAllow: p.entries}
	data, err := json.MarshalIndent(fd, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(p.dir, permFile), data, 0o644)
}

// BashCommandPrefix extracts the first word from a bash tool call's JSON arguments.
// For args like `{"command":"git status"}`, it returns "git".
// Returns empty string if parsing fails or command is empty.
func BashCommandPrefix(argsJSON string) string {
	var parsed struct {
		Command string `json:"command"`
	}
	if json.Unmarshal([]byte(argsJSON), &parsed) != nil || parsed.Command == "" {
		return ""
	}
	cmd := strings.TrimSpace(parsed.Command)
	if i := strings.IndexAny(cmd, " \t"); i > 0 {
		return cmd[:i]
	}
	return cmd
}

// BashEntry returns the permission entry string for a bash command prefix,
// e.g. "bash:git *".
func BashEntry(prefix string) string {
	return "bash:" + prefix + " *"
}

func (p *Permissions) IsAllowed(toolName string, argsJSON string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	var bashCmd string
	if toolName == "bash" {
		bashCmd = parseBashCommand(argsJSON)
	}

	for _, entry := range p.entries {
		if entry == toolName {
			return true
		}
		if toolName == "bash" && strings.HasPrefix(entry, "bash:") && bashCmd != "" {
			pattern := entry[len("bash:"):]
			if matched, _ := filepath.Match(pattern, bashCmd); matched {
				return true
			}
		}
	}
	return false
}

// parseBashCommand extracts the raw command string from bash tool JSON args.
func parseBashCommand(argsJSON string) string {
	var parsed struct {
		Command string `json:"command"`
	}
	if json.Unmarshal([]byte(argsJSON), &parsed) != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Command)
}

func (p *Permissions) Add(entry string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, e := range p.entries {
		if e == entry {
			return nil
		}
	}
	p.entries = append(p.entries, entry)
	return p.saveLocked()
}
