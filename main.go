package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"cx/config"
	"cx/tmux"
	"cx/tui"
	"cx/update"

	tea "github.com/charmbracelet/bubbletea"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	// Set version for TUI display
	tui.Version = version

	// Auto-update check (once per day)
	update.AutoUpdate(version)

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "update":
			if err := update.SelfUpdate(); err != nil {
				fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
				os.Exit(1)
			}
			return
		case "version":
			fmt.Printf("cx version %s\n", version)
			return
		case "help", "-h", "--help":
			printHelp()
			return
		default:
			// Treat as host alias for direct connection
			hostAlias := os.Args[1]
			sessionName := hostAlias // Default session name is the host alias
			if len(os.Args) > 2 {
				sessionName = os.Args[2]
			}
			if err := directConnect(hostAlias, sessionName); err != nil {
				fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	p := tea.NewProgram(tui.NewApp(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("cx - SSH host manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cx                      Launch interactive host selector")
	fmt.Println("  cx <host>               Connect to host with tmux session")
	fmt.Println("  cx <host> <session>     Connect with custom tmux session name")
	fmt.Println("  cx update               Update to latest release")
	fmt.Println("  cx version              Show version info")
	fmt.Println("  cx help                 Show this help")
}

// directConnect connects to a host directly without the TUI
func directConnect(hostAlias, sessionName string) error {
	// Verify host exists in config
	host, err := config.FindHost(hostAlias)
	if err != nil {
		return err
	}
	if host == nil {
		return fmt.Errorf("host %q not found in ~/.ssh/config", hostAlias)
	}

	// Record usage for sorting in TUI
	config.RecordUsage(hostAlias)

	// Build the connection command
	sshCmd := fmt.Sprintf("ssh %s", hostAlias)
	tmuxCmd := tmux.BuildTmuxCommand(sessionName)
	ensureCmd := tmux.BuildEnsureTmuxCommand(tmuxCmd)
	escapedCmd := strings.ReplaceAll(ensureCmd, "'", "'\\''")
	fullCmd := fmt.Sprintf("clear && %s -t '%s'", sshCmd, escapedCmd)

	// Execute
	cmd := exec.Command("bash", "-c", fullCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
