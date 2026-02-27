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

// PushSSHConfig copies local ~/.ssh/config to remote with transformed paths
func PushSSHConfig(alias string) error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".ssh", "config")

	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read local SSH config: %w", err)
	}

	// Transform IdentityFile paths to use ~/ instead of absolute paths
	transformed := transformConfigPaths(string(content), home)

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "ssh-config-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(transformed); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Ensure remote .ssh directory exists
	if err := newSSHCmd(alias, "mkdir -p ~/.ssh && chmod 700 ~/.ssh").Run(); err != nil {
		return fmt.Errorf("failed to create remote .ssh directory: %w", err)
	}

	// Copy config using scp
	if err := scpToRemote(alias, tmpFile.Name(), "~/.ssh/config"); err != nil {
		return err
	}

	// Set correct permissions
	return newSSHCmd(alias, "chmod 600 ~/.ssh/config").Run()
}

// transformConfigPaths converts absolute IdentityFile paths to use ~/
func transformConfigPaths(content, homeDir string) string {
	lines := strings.Split(content, "\n")
	sshDir := filepath.Join(homeDir, ".ssh")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		if strings.HasPrefix(lower, "identityfile ") || strings.HasPrefix(lower, "identityfile\t") {
			// Extract the path portion
			parts := strings.SplitN(trimmed, " ", 2)
			if len(parts) != 2 {
				parts = strings.SplitN(trimmed, "\t", 2)
			}
			if len(parts) == 2 {
				path := strings.TrimSpace(parts[1])
				newPath := transformKeyPath(path, homeDir, sshDir)
				if newPath != path {
					// Preserve original indentation
					indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					lines[i] = indent + "IdentityFile " + newPath
				}
			}
		}
	}

	return strings.Join(lines, "\n")
}

// transformKeyPath converts an absolute key path to use ~/.ssh/
func transformKeyPath(path, homeDir, sshDir string) string {
	// Already uses ~, leave it alone
	if strings.HasPrefix(path, "~/") {
		return path
	}

	// Check if path is under the .ssh directory
	if strings.HasPrefix(path, sshDir+"/") || strings.HasPrefix(path, sshDir+"\\") {
		relPath := path[len(sshDir)+1:]
		return "~/.ssh/" + filepath.ToSlash(relPath)
	}

	// Check if path is under home directory (e.g., ~/.config/keys/)
	if strings.HasPrefix(path, homeDir+"/") || strings.HasPrefix(path, homeDir+"\\") {
		relPath := path[len(homeDir)+1:]
		return "~/" + filepath.ToSlash(relPath)
	}

	// For paths outside home directory, try to extract just the .ssh portion
	// e.g., /root/.ssh/id_rsa -> ~/.ssh/id_rsa
	if idx := strings.Index(path, "/.ssh/"); idx != -1 {
		return "~" + path[idx:]
	}

	// Can't transform, return as-is
	return path
}

// PushSSHKeys copies private keys to remote
func PushSSHKeys(alias string, keyPaths []string) error {
	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")

	// If no keys specified, find all key pairs (including in subdirectories)
	if len(keyPaths) == 0 {
		err := filepath.WalkDir(sshDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // Skip entries we can't read
			}

			// Skip hidden files/dirs and the root .ssh dir itself
			if strings.HasPrefix(d.Name(), ".") && path != sshDir {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip non-key files
			if d.IsDir() {
				return nil
			}

			name := d.Name()
			if strings.HasSuffix(name, ".pub") ||
				name == "config" ||
				name == "known_hosts" ||
				name == "known_hosts.old" ||
				name == "authorized_keys" {
				return nil
			}

			// Check if this file has a matching .pub (indicates it's a key pair)
			pubPath := path + ".pub"
			if _, err := os.Stat(pubPath); err == nil {
				keyPaths = append(keyPaths, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to scan .ssh directory: %w", err)
		}
	}

	if len(keyPaths) == 0 {
		return fmt.Errorf("no SSH keys found to push")
	}

	// Ensure remote .ssh exists with correct permissions
	if err := newSSHCmd(alias, "mkdir -p ~/.ssh && chmod 700 ~/.ssh").Run(); err != nil {
		return fmt.Errorf("failed to create remote .ssh directory: %w", err)
	}

	// Copy each key pair using scp
	for _, keyPath := range keyPaths {
		// Determine the relative path from .ssh dir for remote destination
		// Use filepath.ToSlash to ensure POSIX paths for remote host
		relPath, err := filepath.Rel(sshDir, keyPath)
		if err != nil {
			relPath = filepath.Base(keyPath)
		}
		relPath = filepath.ToSlash(relPath)
		remotePath := ".ssh/" + relPath

		// Ensure remote subdirectory exists if needed
		if idx := strings.LastIndex(remotePath, "/"); idx > len(".ssh") {
			remoteDir := remotePath[:idx]
			if err := newSSHCmd(alias, fmt.Sprintf("mkdir -p ~/%s", remoteDir)).Run(); err != nil {
				return fmt.Errorf("failed to create remote directory %s: %w", remoteDir, err)
			}
		}

		// Copy private key
		if err := scpToRemote(alias, keyPath, "~/"+remotePath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", keyPath, err)
		}
		// Set permissions on private key (must succeed for security)
		if err := newSSHCmd(alias, fmt.Sprintf("chmod 600 ~/%s", remotePath)).Run(); err != nil {
			return fmt.Errorf("failed to set permissions on %s: %w", remotePath, err)
		}

		// Copy public key if exists
		pubPath := keyPath + ".pub"
		if _, err := os.Stat(pubPath); err == nil {
			if err := scpToRemote(alias, pubPath, "~/"+remotePath+".pub"); err != nil {
				return fmt.Errorf("failed to copy %s: %w", pubPath, err)
			}
			// Public key permissions less critical, but still set them
			newSSHCmd(alias, fmt.Sprintf("chmod 644 ~/%s.pub", remotePath)).Run()
		}
	}

	return nil
}

// scpToRemote copies a local file or directory to a remote path using scp
func scpToRemote(alias, localPath, remotePath string) error {
	info, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	args := []string{"-q"}
	if info.IsDir() {
		args = append(args, "-r")
	}
	args = append(args, localPath, alias+":"+remotePath)

	cmd := exec.Command("scp", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// newSSHCmd creates an SSH command with stdin/stdout/stderr wired up
func newSSHCmd(alias string, remoteCmd string) *exec.Cmd {
	cmd := exec.Command("ssh", alias, remoteCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// PushTmuxProfile pushes a tmux.conf to the remote host
func PushTmuxProfile(alias, configContent string) error {
	// Write to temp file
	tmpFile, err := os.CreateTemp("", "tmux-conf-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Copy to remote ~/.tmux.conf using scp
	if err := scpToRemote(alias, tmpFile.Name(), "~/.tmux.conf"); err != nil {
		return fmt.Errorf("failed to copy tmux.conf: %w", err)
	}

	// Set correct permissions
	if err := newSSHCmd(alias, "chmod 644 ~/.tmux.conf").Run(); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Kill tmux server so next connection starts fresh with new config
	// source-file doesn't apply all settings (like set-clipboard, allow-passthrough)
	newSSHCmd(alias, "tmux kill-server 2>/dev/null || true").Run()

	return nil
}
