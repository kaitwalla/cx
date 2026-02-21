package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CopyID copies a public key to a remote host
func CopyID(keyPath, host, user string, port string) error {
	args := []string{"-i", keyPath}

	if port != "" && port != "22" {
		args = append(args, "-p", port)
	}

	target := host
	if user != "" {
		target = fmt.Sprintf("%s@%s", user, host)
	}
	args = append(args, target)

	cmd := exec.Command("ssh-copy-id", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// CopyIDWithPassword copies a public key using sshpass for non-interactive auth
func CopyIDWithPassword(keyPath, host, user, port, password string) error {
	// Check if sshpass is available
	if _, err := exec.LookPath("sshpass"); err != nil {
		// Fall back to regular ssh-copy-id
		return CopyID(keyPath, host, user, port)
	}

	args := []string{"-p", password, "ssh-copy-id", "-i", keyPath}

	if port != "" && port != "22" {
		args = append(args, "-p", port)
	}

	target := host
	if user != "" {
		target = fmt.Sprintf("%s@%s", user, host)
	}
	args = append(args, target)

	cmd := exec.Command("sshpass", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// CopyKeyManually copies a public key by directly appending to authorized_keys
// This is useful when ssh-copy-id is not available
func CopyKeyManually(keyPath, host, user, port string) error {
	pubKeyPath := GetPublicKeyPath(keyPath)
	pubKey, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	// Build the remote command
	remoteCmd := fmt.Sprintf(
		"mkdir -p ~/.ssh && chmod 700 ~/.ssh && echo %q >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys",
		strings.TrimSpace(string(pubKey)),
	)

	// Build SSH args
	sshArgs := []string{}
	if port != "" && port != "22" {
		sshArgs = append(sshArgs, "-p", port)
	}

	target := host
	if user != "" {
		target = fmt.Sprintf("%s@%s", user, host)
	}
	sshArgs = append(sshArgs, target, remoteCmd)

	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// CheckKeyDeployed tests if a key is already deployed to a host
func CheckKeyDeployed(host, user, port, keyPath string) bool {
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
	}

	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}

	if port != "" && port != "22" {
		args = append(args, "-p", port)
	}

	target := host
	if user != "" {
		target = fmt.Sprintf("%s@%s", user, host)
	}
	args = append(args, target, "exit", "0")

	cmd := exec.Command("ssh", args...)
	err := cmd.Run()
	return err == nil
}
