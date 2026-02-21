package tmux

import (
	"os/exec"
	"runtime"
	"strings"
)

// OSType represents the operating system type
type OSType string

const (
	OSLinux   OSType = "linux"
	OSDarwin  OSType = "darwin"
	OSUnknown OSType = "unknown"
)

// DetectOS returns the current operating system type
func DetectOS() OSType {
	switch runtime.GOOS {
	case "linux":
		return OSLinux
	case "darwin":
		return OSDarwin
	default:
		return OSUnknown
	}
}

// IsTmuxAvailable checks if tmux is installed locally
func IsTmuxAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// GetTmuxVersion returns the installed tmux version
func GetTmuxVersion() (string, error) {
	cmd := exec.Command("tmux", "-V")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// RemoteDetectOS detects the OS of a remote host
func RemoteDetectOS(sshAlias string) (OSType, error) {
	cmd := exec.Command("ssh", sshAlias, "uname", "-s")
	output, err := cmd.Output()
	if err != nil {
		return OSUnknown, err
	}

	osName := strings.TrimSpace(strings.ToLower(string(output)))
	switch osName {
	case "linux":
		return OSLinux, nil
	case "darwin":
		return OSDarwin, nil
	default:
		return OSUnknown, nil
	}
}

// RemoteIsTmuxAvailable checks if tmux is installed on a remote host
func RemoteIsTmuxAvailable(sshAlias string) bool {
	cmd := exec.Command("ssh", sshAlias, "command", "-v", "tmux")
	return cmd.Run() == nil
}

// DetectLinuxDistro detects the Linux distribution on a remote host
type LinuxDistro string

const (
	DistroDebian  LinuxDistro = "debian"
	DistroRedHat  LinuxDistro = "redhat"
	DistroArch    LinuxDistro = "arch"
	DistroUnknown LinuxDistro = "unknown"
)

// RemoteDetectDistro detects the Linux distribution on a remote host
func RemoteDetectDistro(sshAlias string) LinuxDistro {
	// Check for Debian/Ubuntu
	cmd := exec.Command("ssh", sshAlias, "test", "-f", "/etc/debian_version")
	if cmd.Run() == nil {
		return DistroDebian
	}

	// Check for RHEL/CentOS/Fedora
	cmd = exec.Command("ssh", sshAlias, "test", "-f", "/etc/redhat-release")
	if cmd.Run() == nil {
		return DistroRedHat
	}

	// Check for Arch
	cmd = exec.Command("ssh", sshAlias, "test", "-f", "/etc/arch-release")
	if cmd.Run() == nil {
		return DistroArch
	}

	return DistroUnknown
}
