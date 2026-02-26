package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	// DefaultSessionName is the default tmux session name
	DefaultSessionName = "cx"
)

// IsITerm returns true if running inside iTerm2
func IsITerm() bool {
	// iTerm2 sets TERM_PROGRAM=iTerm.app or LC_TERMINAL=iTerm2
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		return true
	}
	if os.Getenv("LC_TERMINAL") == "iTerm2" {
		return true
	}
	// Also check ITERM_SESSION_ID which is set in iTerm sessions
	if os.Getenv("ITERM_SESSION_ID") != "" {
		return true
	}
	return false
}

// BuildTmuxCommand builds the command to create or attach to a tmux session
// If sessionName is empty, uses the default session name
func BuildTmuxCommand(sessionName string) string {
	return BuildTmuxCommandWithOptions(sessionName, false)
}

// BuildTmuxCommandWithOptions builds the command with optional control mode
// controlMode: if true, adds -CC flag for iTerm2 integration
func BuildTmuxCommandWithOptions(sessionName string, controlMode bool) string {
	// tmux new-session -A -s <name>
	// -A: Attach to session if it exists, create if it doesn't
	// -s: Session name
	// -CC: Control mode for iTerm2 native integration
	if sessionName == "" {
		sessionName = DefaultSessionName
	}
	// Escape single quotes for safe shell interpolation
	escaped := strings.ReplaceAll(sessionName, "'", "'\\''")

	if controlMode {
		// -CC: control mode for iTerm2 native integration
		// -p: use new profile (opens in current window as new tab)
		return fmt.Sprintf("tmux -CC -p new-session -A -s '%s'", escaped)
	}
	return fmt.Sprintf("tmux new-session -A -s '%s'", escaped)
}

// BuildTmuxWithInstallCommand builds a command that checks for tmux and installs if needed
func BuildTmuxWithInstallCommand() string {
	return `
if command -v tmux >/dev/null 2>&1; then
    tmux new-session -A -s cx
else
    echo "tmux is not installed."
    echo "Install it with:"
    echo "  Debian/Ubuntu: sudo apt-get install tmux"
    echo "  RHEL/CentOS:   sudo yum install tmux"
    echo "  macOS:         brew install tmux"
    echo ""
    read -p "Would you like to try to install tmux? [y/N] " answer
    case $answer in
        [Yy]*)
            if [ -f /etc/debian_version ]; then
                sudo apt-get update && sudo apt-get install -y tmux
            elif [ -f /etc/redhat-release ]; then
                sudo yum install -y tmux
            elif [ "$(uname)" = "Darwin" ]; then
                brew install tmux
            else
                echo "Unknown system. Please install tmux manually."
                exit 1
            fi
            tmux new-session -A -s cx
            ;;
        *)
            echo "Connecting without tmux..."
            exec $SHELL
            ;;
    esac
fi
`
}

// SessionExists checks if a tmux session exists locally
func SessionExists(sessionName string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	return cmd.Run() == nil
}

// ListSessions lists all local tmux sessions
func ListSessions() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// No sessions is not an error
		if strings.Contains(err.Error(), "no server running") {
			return nil, nil
		}
		return nil, err
	}

	sessions := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(sessions) == 1 && sessions[0] == "" {
		return nil, nil
	}

	return sessions, nil
}

// AttachSession attaches to an existing tmux session
func AttachSession(sessionName string) error {
	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	return cmd.Run()
}

// NewSession creates a new tmux session
func NewSession(sessionName string) error {
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
	return cmd.Run()
}

// KillSession kills a tmux session
func KillSession(sessionName string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	return cmd.Run()
}
