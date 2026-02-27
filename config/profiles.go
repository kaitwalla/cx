package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// TmuxProfile represents a tmux configuration profile
type TmuxProfile struct {
	Name           string `json:"name"`
	PrefixKey      string `json:"prefix_key"`
	StatusFG       string `json:"status_fg"`
	StatusBG       string `json:"status_bg"`
	StatusPosition string `json:"status_position"`
}

// ProfileStore holds all tmux profiles
type ProfileStore struct {
	Profiles []TmuxProfile `json:"profiles"`
}

// HostProfiles maps host aliases to their assigned profile names
type HostProfiles struct {
	Assignments map[string]string `json:"assignments"`
}

// profilesPath returns the path to ~/.config/cx/profiles.json
func profilesPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "cx", "profiles.json"), nil
}

// hostProfilesPath returns the path to ~/.config/cx/host_profiles.json
func hostProfilesPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "cx", "host_profiles.json"), nil
}

// DefaultProfiles returns the built-in default profiles
// Colors are chosen to match their keybinding letter for easy recall
func DefaultProfiles() []TmuxProfile {
	return []TmuxProfile{
		{
			Name:           "blue",
			PrefixKey:      "C-b",
			StatusFG:       "white",
			StatusBG:       "#4169E1",
			StatusPosition: "bottom",
		},
		{
			Name:           "green",
			PrefixKey:      "C-g",
			StatusFG:       "black",
			StatusBG:       "#00AA00",
			StatusPosition: "top",
		},
		{
			Name:           "aqua",
			PrefixKey:      "C-q",
			StatusFG:       "black",
			StatusBG:       "#00CED1",
			StatusPosition: "top",
		},
		{
			Name:           "amber",
			PrefixKey:      "C-a",
			StatusFG:       "black",
			StatusBG:       "#FFBF00",
			StatusPosition: "bottom",
		},
		{
			Name:           "rose",
			PrefixKey:      "C-r",
			StatusFG:       "white",
			StatusBG:       "#FF007F",
			StatusPosition: "bottom",
		},
		{
			Name:           "violet",
			PrefixKey:      "C-v",
			StatusFG:       "white",
			StatusBG:       "#8A2BE2",
			StatusPosition: "bottom",
		},
	}
}

// LoadProfiles loads profiles from disk, seeding defaults if needed
func LoadProfiles() (*ProfileStore, error) {
	path, err := profilesPath()
	if err != nil {
		return nil, err
	}

	// Ensure config directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	store := &ProfileStore{}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Seed with default profiles
			store.Profiles = DefaultProfiles()
			if err := store.Save(); err != nil {
				return nil, err
			}
			return store, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, store); err != nil {
		return nil, err
	}

	// If no profiles, seed defaults
	if len(store.Profiles) == 0 {
		store.Profiles = DefaultProfiles()
		if err := store.Save(); err != nil {
			return nil, fmt.Errorf("saving default profiles: %w", err)
		}
	}

	return store, nil
}

// Save writes profiles to disk
func (s *ProfileStore) Save() error {
	path, err := profilesPath()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetProfile returns a profile by name
func (s *ProfileStore) GetProfile(name string) *TmuxProfile {
	for i := range s.Profiles {
		if s.Profiles[i].Name == name {
			return &s.Profiles[i]
		}
	}
	return nil
}

// ListProfiles returns all profile names
func (s *ProfileStore) ListProfiles() []string {
	names := make([]string, len(s.Profiles))
	for i, p := range s.Profiles {
		names[i] = p.Name
	}
	return names
}

// GenerateConfig generates the full tmux.conf content for a profile
func (p *TmuxProfile) GenerateConfig() string {
	return fmt.Sprintf(`# cx tmux profile: %s

# Prefix
unbind C-b
set -g prefix %s
bind %s send-prefix

# Status bar
set -g status-style bg=%s,fg=%s
set -g window-status-current-style bg=%s,fg=%s,bold
set -g status-position %s

# Mouse
set -g mouse on
bind m set -g mouse \; display "Mouse: #{?mouse,ON,OFF}"

# Scrolling
set -g history-limit 10000
bind -n WheelUpPane if-shell -F -t = "#{mouse_any_flag}" "send-keys -M" "if -Ft= '#{pane_in_mode}' 'send-keys -M' 'select-pane -t=; copy-mode -e; send-keys -M'"
bind -n WheelDownPane select-pane -t= \; send-keys -M

# System Clipboard Integration via OSC 52 (works over SSH)
set -g set-clipboard on
set -g allow-passthrough on
`,
		p.Name,
		p.PrefixKey,
		p.PrefixKey,
		p.StatusBG, p.StatusFG,
		p.StatusFG, p.StatusBG,
		p.StatusPosition,
	)
}

// LoadHostProfiles loads the host-to-profile mappings
func LoadHostProfiles() (*HostProfiles, error) {
	path, err := hostProfilesPath()
	if err != nil {
		return nil, err
	}

	hp := &HostProfiles{
		Assignments: make(map[string]string),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return hp, nil
		}
		return nil, fmt.Errorf("reading host profiles: %w", err)
	}

	if err := json.Unmarshal(data, hp); err != nil {
		return nil, fmt.Errorf("parsing host profiles: %w", err)
	}

	return hp, nil
}

// Save writes host profile mappings to disk
func (hp *HostProfiles) Save() error {
	path, err := hostProfilesPath()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(hp, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetHostProfile returns the profile name for a host
func (hp *HostProfiles) GetHostProfile(alias string) string {
	return hp.Assignments[alias]
}

// SetHostProfile sets the profile for a host
func (hp *HostProfiles) SetHostProfile(alias, profileName string) error {
	if profileName == "" {
		delete(hp.Assignments, alias)
	} else {
		hp.Assignments[alias] = profileName
	}
	return hp.Save()
}
