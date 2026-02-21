package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// KeyType represents the type of SSH key
type KeyType string

const (
	KeyTypeED25519 KeyType = "ed25519"
	KeyTypeRSA     KeyType = "rsa"
)

// GenerateKey generates a new SSH key pair
func GenerateKey(keyPath string, keyType KeyType, comment string, passphrase string) error {
	// Ensure .ssh directory exists
	sshDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	// Check if key already exists
	if _, err := os.Stat(keyPath); err == nil {
		return fmt.Errorf("key already exists at %s", keyPath)
	}

	args := []string{
		"-t", string(keyType),
		"-f", keyPath,
		"-N", passphrase,
	}

	if comment != "" {
		args = append(args, "-C", comment)
	}

	if keyType == KeyTypeRSA {
		args = append(args, "-b", "4096")
	}

	cmd := exec.Command("ssh-keygen", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-keygen failed: %w", err)
	}

	return nil
}

// GenerateKeyNonInteractive generates a key without prompts
func GenerateKeyNonInteractive(keyPath string, keyType KeyType, comment string) error {
	// Ensure .ssh directory exists
	sshDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	// Check if key already exists
	if _, err := os.Stat(keyPath); err == nil {
		return fmt.Errorf("key already exists at %s", keyPath)
	}

	args := []string{
		"-t", string(keyType),
		"-f", keyPath,
		"-N", "", // No passphrase
		"-q",     // Quiet mode
	}

	if comment != "" {
		args = append(args, "-C", comment)
	}

	if keyType == KeyTypeRSA {
		args = append(args, "-b", "4096")
	}

	cmd := exec.Command("ssh-keygen", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-keygen failed: %w - %s", err, string(output))
	}

	return nil
}

// GetPublicKeyPath returns the public key path for a private key
func GetPublicKeyPath(privateKeyPath string) string {
	return privateKeyPath + ".pub"
}

// ReadPublicKey reads the contents of a public key file
func ReadPublicKey(keyPath string) (string, error) {
	pubKeyPath := GetPublicKeyPath(keyPath)
	data, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}
	return string(data), nil
}

// DefaultKeyPath returns the default path for a new key
func DefaultKeyPath(name string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ssh", fmt.Sprintf("id_%s_%s", KeyTypeED25519, name))
}

// KeyExists checks if a key exists at the given path
func KeyExists(keyPath string) bool {
	_, err := os.Stat(keyPath)
	return err == nil
}
