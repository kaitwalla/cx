package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PushPublicKey deploys the public key to remote authorized_keys
func PushPublicKey(alias string, keyPath string) error {
	// If no key specified, try to find one
	if keyPath == "" {
		home, _ := os.UserHomeDir()
		candidates := []string{
			filepath.Join(home, ".ssh", "id_ed25519"),
			filepath.Join(home, ".ssh", "id_rsa"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				keyPath = c
				break
			}
		}
		if keyPath == "" {
			return fmt.Errorf("no SSH key found")
		}
	}

	// Use ssh-copy-id if available
	if _, err := exec.LookPath("ssh-copy-id"); err == nil {
		cmd := exec.Command("ssh-copy-id", "-i", keyPath, alias)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Fallback: manually append to authorized_keys
	pubKeyPath := keyPath + ".pub"
	pubKey, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	remoteCmd := fmt.Sprintf(
		"mkdir -p ~/.ssh && chmod 700 ~/.ssh && echo %q >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys",
		strings.TrimSpace(string(pubKey)),
	)

	cmd := exec.Command("ssh", alias, remoteCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PushSSHConfig copies local ~/.ssh/config to remote
func PushSSHConfig(alias string) error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".ssh", "config")

	// Read local config
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read local SSH config: %w", err)
	}

	// Create remote .ssh directory and write config
	remoteCmd := fmt.Sprintf(
		"mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat > ~/.ssh/config && chmod 600 ~/.ssh/config",
	)

	cmd := exec.Command("ssh", alias, remoteCmd)
	cmd.Stdin = strings.NewReader(string(content))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PushSSHKeys copies private keys to remote
func PushSSHKeys(alias string, keyPaths []string) error {
	home, _ := os.UserHomeDir()

	// If no keys specified, find all key pairs
	if len(keyPaths) == 0 {
		sshDir := filepath.Join(home, ".ssh")
		entries, err := os.ReadDir(sshDir)
		if err != nil {
			return fmt.Errorf("failed to read .ssh directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				name := entry.Name()
				// Look for private keys (files without .pub that have a .pub counterpart)
				if !strings.HasSuffix(name, ".pub") &&
					name != "config" &&
					name != "known_hosts" &&
					name != "authorized_keys" {
					pubPath := filepath.Join(sshDir, name+".pub")
					if _, err := os.Stat(pubPath); err == nil {
						keyPaths = append(keyPaths, filepath.Join(sshDir, name))
					}
				}
			}
		}
	}

	if len(keyPaths) == 0 {
		return fmt.Errorf("no SSH keys found to push")
	}

	// Ensure remote .ssh exists
	cmd := exec.Command("ssh", alias, "mkdir -p ~/.ssh && chmod 700 ~/.ssh")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create remote .ssh directory: %w", err)
	}

	// Copy each key pair
	for _, keyPath := range keyPaths {
		// Copy private key
		if err := copyFileToRemote(alias, keyPath, "~/.ssh/"+filepath.Base(keyPath), "600"); err != nil {
			return fmt.Errorf("failed to copy %s: %w", keyPath, err)
		}

		// Copy public key if exists
		pubPath := keyPath + ".pub"
		if _, err := os.Stat(pubPath); err == nil {
			if err := copyFileToRemote(alias, pubPath, "~/.ssh/"+filepath.Base(pubPath), "644"); err != nil {
				return fmt.Errorf("failed to copy %s: %w", pubPath, err)
			}
		}
	}

	return nil
}

// copyFileToRemote copies a local file to a remote path via SSH
func copyFileToRemote(alias, localPath, remotePath, mode string) error {
	content, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}

	remoteCmd := fmt.Sprintf("cat > %s && chmod %s %s", remotePath, mode, remotePath)
	cmd := exec.Command("ssh", alias, remoteCmd)
	cmd.Stdin = strings.NewReader(string(content))
	return cmd.Run()
}
