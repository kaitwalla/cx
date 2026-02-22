package tmux

import (
	"fmt"
	"os"
	"os/exec"
)

// InstallCommand returns the command to install tmux for a given OS/distro
func InstallCommand(osType OSType, distro LinuxDistro) string {
	switch osType {
	case OSDarwin:
		return "brew install tmux"
	case OSLinux:
		switch distro {
		case DistroDebian:
			return "sudo apt-get update && sudo apt-get install -y tmux"
		case DistroRedHat:
			return "sudo yum install -y tmux"
		case DistroArch:
			return "sudo pacman -S --noconfirm tmux"
		default:
			return "# Please install tmux manually for your distribution"
		}
	default:
		return "# Please install tmux manually for your operating system"
	}
}

// RemoteInstallTmux installs tmux on a remote host
func RemoteInstallTmux(sshAlias string) error {
	// Detect remote OS
	osType, err := RemoteDetectOS(sshAlias)
	if err != nil {
		return fmt.Errorf("failed to detect remote OS: %w", err)
	}

	var installCmd string

	switch osType {
	case OSDarwin:
		installCmd = "brew install tmux"
	case OSLinux:
		distro := RemoteDetectDistro(sshAlias)
		switch distro {
		case DistroDebian:
			installCmd = "sudo apt-get update && sudo apt-get install -y tmux"
		case DistroRedHat:
			installCmd = "sudo yum install -y tmux"
		case DistroArch:
			installCmd = "sudo pacman -S --noconfirm tmux"
		default:
			return fmt.Errorf("unknown Linux distribution, please install tmux manually")
		}
	default:
		return fmt.Errorf("unsupported operating system: %s", osType)
	}

	cmd := exec.Command("ssh", "-t", sshAlias, installCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// GetInstallInstructions returns human-readable install instructions
func GetInstallInstructions(osType OSType, distro LinuxDistro) string {
	switch osType {
	case OSDarwin:
		return "Install tmux with: brew install tmux"
	case OSLinux:
		switch distro {
		case DistroDebian:
			return "Install tmux with: sudo apt-get install tmux"
		case DistroRedHat:
			return "Install tmux with: sudo yum install tmux"
		case DistroArch:
			return "Install tmux with: sudo pacman -S tmux"
		default:
			return "Please install tmux using your package manager"
		}
	default:
		return "Please install tmux for your operating system"
	}
}

// BuildEnsureTmuxCommand builds a bash command that checks for tmux,
// installs it if missing, then runs the given tmux command.
// IMPORTANT: tmuxCmd is interpolated directly into a shell command.
// Callers must ensure tmuxCmd does not contain untrusted user input.
func BuildEnsureTmuxCommand(tmuxCmd string) string {
	return fmt.Sprintf(`command -v tmux >/dev/null 2>&1 || {
  echo "tmux not found, installing..."
  if [ -f /etc/debian_version ]; then
    sudo apt-get update && sudo apt-get install -y tmux
  elif [ -f /etc/redhat-release ]; then
    sudo yum install -y tmux
  elif [ -f /etc/arch-release ]; then
    sudo pacman -S --noconfirm tmux
  elif [ "$(uname)" = "Darwin" ]; then
    brew install tmux
  else
    echo "Cannot auto-install tmux. Please install manually."
    exit 1
  fi
} && %s`, tmuxCmd)
}
