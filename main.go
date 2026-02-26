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
			// Parse host alias and optional flags
			hostAlias := os.Args[1]
			sessionName := hostAlias // Default session name is the host alias
			var command string

			// Parse remaining arguments
			for i := 2; i < len(os.Args); i++ {
				arg := os.Args[i]
				switch arg {
				case "--cmd", "-c", "--command":
					if i+1 < len(os.Args) {
						command = os.Args[i+1]
						i++ // Skip the next argument since we consumed it
					} else {
						fmt.Fprintf(os.Stderr, "Error: %s requires a command argument\n", arg)
						os.Exit(1)
					}
				default:
					// Treat as session name if no flag prefix
					if !strings.HasPrefix(arg, "-") {
						sessionName = arg
					} else {
						fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", arg)
						os.Exit(1)
					}
				}
			}

			if err := directConnect(hostAlias, sessionName, command); err != nil {
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
	fmt.Println("  cx                              Launch interactive host selector")
	fmt.Println("  cx <host>                       Connect to host with tmux session")
	fmt.Println("  cx <host> <session>             Connect with custom tmux session name")
	fmt.Println("  cx <host> --cmd '<command>'     Run command on host (no tmux)")
	fmt.Println("  cx <host> <session> --cmd '...' Run command in named tmux session")
	fmt.Println("  cx update                       Update to latest release")
	fmt.Println("  cx version                      Show version info")
	fmt.Println("  cx help                         Show this help")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --cmd, -c, --command <command>    Run a command on the remote host")
}

// directConnect connects to a host directly without the TUI
// If command is non-empty, it runs that command instead of the default tmux session
func directConnect(hostAlias, sessionName, command string) error {
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

	var fullCmd string

	// Escape values for safe shell interpolation
	escapeShell := func(s string) string {
		return strings.ReplaceAll(s, "'", "'\\''")
	}
	escapedHost := escapeShell(hostAlias)
	escapedSession := escapeShell(sessionName)

	// Check if we're in iTerm2 for control mode
	useControlMode := tmux.IsITerm()

	if command != "" {
		// Run the specified command
		escapedUserCmd := escapeShell(command)
		// If sessionName differs from hostAlias, user wants tmux with the command
		if sessionName != hostAlias {
			// Run command inside a tmux session
			// tmux new-session -A -s <session> '<command>'
			var tmuxCmd string
			if useControlMode {
				tmuxCmd = fmt.Sprintf("tmux -CC -p new-session -A -s '%s' '%s'",
					escapedSession, escapedUserCmd)
			} else {
				tmuxCmd = fmt.Sprintf("tmux new-session -A -s '%s' '%s'",
					escapedSession, escapedUserCmd)
			}
			ensureCmd := tmux.BuildEnsureTmuxCommand(tmuxCmd)
			escapedCmd := escapeShell(ensureCmd)
			fullCmd = fmt.Sprintf("clear && ssh '%s' -t '%s'", escapedHost, escapedCmd)
		} else {
			// Run command directly without tmux
			fullCmd = fmt.Sprintf("clear && ssh '%s' '%s'", escapedHost, escapedUserCmd)
		}
	} else {
		// Default behavior: connect with tmux session
		tmuxCmd := tmux.BuildTmuxCommandWithOptions(sessionName, useControlMode)
		ensureCmd := tmux.BuildEnsureTmuxCommand(tmuxCmd)
		escapedCmd := escapeShell(ensureCmd)
		fullCmd = fmt.Sprintf("clear && ssh '%s' -t '%s'", escapedHost, escapedCmd)
	}

	// Execute
	cmd := exec.Command("bash", "-c", fullCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
