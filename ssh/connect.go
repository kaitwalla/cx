package ssh

import (
	"fmt"
	"os"
	"os/exec"
)

// Connect establishes an SSH connection using the system ssh command
func Connect(alias, hostname, user, port, identityFile string) error {
	args := []string{}

	// Use alias if available (ssh config handles the rest)
	if alias != "" {
		args = append(args, alias)
	} else {
		// Build connection string manually
		if user != "" {
			args = append(args, fmt.Sprintf("%s@%s", user, hostname))
		} else {
			args = append(args, hostname)
		}

		if port != "" && port != "22" {
			args = append(args, "-p", port)
		}

		if identityFile != "" {
			args = append(args, "-i", identityFile)
		}
	}

	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ConnectWithCommand runs a command on the remote host
func ConnectWithCommand(alias, command string) error {
	cmd := exec.Command("ssh", "-t", alias, command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// TestConnection tests if we can connect to a host
func TestConnection(alias string) error {
	cmd := exec.Command("ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", alias, "exit", "0")
	return cmd.Run()
}

// RunCommand runs a command on a remote host and returns output
func RunCommand(alias, command string) (string, error) {
	cmd := exec.Command("ssh", alias, command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
