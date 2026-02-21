package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Host represents an SSH host configuration
type Host struct {
	Alias        string
	HostName     string
	User         string
	Port         string
	IdentityFile string
}

// ConfigPath returns the path to ~/.ssh/config
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ssh", "config")
}

// SSHDir returns the path to ~/.ssh
func SSHDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ssh")
}

// ParseConfig reads and parses ~/.ssh/config
func ParseConfig() ([]Host, error) {
	path := ConfigPath()

	// Ensure .ssh directory exists
	if err := os.MkdirAll(SSHDir(), 0700); err != nil {
		return nil, err
	}

	// Create config file if it doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		f.Close()
		os.Chmod(path, 0600)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var hosts []Host
	var current *Host

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key-value pairs
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			// Try with tabs
			parts = strings.SplitN(line, "\t", 2)
			if len(parts) != 2 {
				continue
			}
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch strings.ToLower(key) {
		case "host":
			// Skip wildcard hosts
			if strings.Contains(value, "*") {
				current = nil
				continue
			}
			if current != nil {
				hosts = append(hosts, *current)
			}
			current = &Host{Alias: value}
		case "hostname":
			if current != nil {
				current.HostName = value
			}
		case "user":
			if current != nil {
				current.User = value
			}
		case "port":
			if current != nil {
				current.Port = value
			}
		case "identityfile":
			if current != nil {
				current.IdentityFile = value
			}
		}
	}

	// Don't forget the last host
	if current != nil {
		hosts = append(hosts, *current)
	}

	return hosts, scanner.Err()
}

// FindHost finds a host by alias
func FindHost(alias string) (*Host, error) {
	hosts, err := ParseConfig()
	if err != nil {
		return nil, err
	}

	for _, h := range hosts {
		if h.Alias == alias {
			return &h, nil
		}
	}
	return nil, nil
}

// ListKeyFiles returns all public key files in ~/.ssh
func ListKeyFiles() ([]string, error) {
	sshDir := SSHDir()
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var keys []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pub") {
			// Return the private key path (without .pub)
			keyPath := filepath.Join(sshDir, strings.TrimSuffix(entry.Name(), ".pub"))
			if _, err := os.Stat(keyPath); err == nil {
				keys = append(keys, keyPath)
			}
		}
	}

	return keys, nil
}
