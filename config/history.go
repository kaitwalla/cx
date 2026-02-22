package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// History stores the last used timestamp for each host alias
type History struct {
	LastUsed map[string]time.Time `json:"last_used"`
}

// historyPath returns the path to the history file
func historyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "cx_history.json"), nil
}

// LoadHistory loads the history from disk
func LoadHistory() (*History, error) {
	h := &History{
		LastUsed: make(map[string]time.Time),
	}

	path, err := historyPath()
	if err != nil {
		return h, nil // Return empty history if can't get path
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return h, nil
		}
		return h, nil // Return empty history on read error
	}

	if err := json.Unmarshal(data, h); err != nil {
		return h, nil // Return empty history on parse error
	}

	return h, nil
}

// Save writes the history to disk
func (h *History) Save() error {
	path, err := historyPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// RecordUsage records that a host was used now
func RecordUsage(alias string) error {
	h, err := LoadHistory()
	if err != nil {
		h = &History{LastUsed: make(map[string]time.Time)}
	}

	h.LastUsed[alias] = time.Now()
	return h.Save()
}

// SortByLastUsed sorts hosts by last used time (most recent first)
// Hosts without history are placed at the end, sorted alphabetically
func SortByLastUsed(hosts []Host) []Host {
	h, _ := LoadHistory()

	sort.Slice(hosts, func(i, j int) bool {
		ti, hasI := h.LastUsed[hosts[i].Alias]
		tj, hasJ := h.LastUsed[hosts[j].Alias]

		// Both have history: sort by most recent
		if hasI && hasJ {
			return ti.After(tj)
		}
		// Only i has history: i comes first
		if hasI {
			return true
		}
		// Only j has history: j comes first
		if hasJ {
			return false
		}
		// Neither has history: sort alphabetically
		return hosts[i].Alias < hosts[j].Alias
	})

	return hosts
}
