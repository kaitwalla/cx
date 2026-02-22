package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	repoOwner      = "kaitwalla"
	repoName       = "cx"
	releaseTag     = "latest"
	checkInterval  = 24 * time.Hour
	lastCheckFile  = ".cx_last_update_check"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// Release represents a GitHub release
type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Check checks for updates and returns release info
func Check() (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s",
		repoOwner, repoName, releaseTag)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("no releases found")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &release, nil
}

// getBinaryName returns the expected binary name for the current platform
func getBinaryName() string {
	return fmt.Sprintf("cx-%s-%s", runtime.GOOS, runtime.GOARCH)
}

// findAsset finds the download URL for the current platform
func findAsset(release *Release) (string, error) {
	binaryName := getBinaryName()
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("no binary found for %s", binaryName)
}

// findChecksumAsset finds the checksum file URL for the current platform
func findChecksumAsset(release *Release) (string, error) {
	checksumName := getBinaryName() + ".sha256"
	for _, asset := range release.Assets {
		if asset.Name == checksumName {
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("no checksum found for %s", checksumName)
}

// findVersionAsset finds the VERSION file URL
func findVersionAsset(release *Release) (string, error) {
	for _, asset := range release.Assets {
		if asset.Name == "VERSION" {
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("no VERSION file found in release")
}

// downloadVersion downloads and returns the release version from VERSION file
func downloadVersion(url string) (string, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("version download failed: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read version: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// downloadChecksum downloads and parses the expected checksum
func downloadChecksum(url string) (string, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download checksum: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("checksum download failed: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read checksum: %w", err)
	}

	// Parse sha256sum format: "hash  filename"
	parts := strings.Fields(string(data))
	if len(parts) < 1 {
		return "", fmt.Errorf("invalid checksum format")
	}

	return parts[0], nil
}

// verifyChecksum computes SHA256 of a file and compares to expected
func verifyChecksum(filePath, expected string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}

// SelfUpdate downloads and replaces the current binary
func SelfUpdate() error {
	release, err := Check()
	if err != nil {
		return err
	}

	downloadURL, err := findAsset(release)
	if err != nil {
		return err
	}

	checksumURL, err := findChecksumAsset(release)
	if err != nil {
		return err
	}

	fmt.Printf("Downloading %s...\n", release.Name)

	// Download expected checksum first
	expectedChecksum, err := downloadChecksum(checksumURL)
	if err != nil {
		return err
	}

	// Download new binary
	resp, err := httpClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create temp file in same directory (for atomic rename)
	tmpFile, err := os.CreateTemp("", "cx-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Copy download to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write update: %w", err)
	}
	tmpFile.Close()

	// Verify checksum before replacing
	fmt.Println("Verifying checksum...")
	if err := verifyChecksum(tmpPath, expectedChecksum); err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to chmod: %w", err)
	}

	// Replace old binary
	if err := os.Rename(tmpPath, execPath); err != nil {
		// If rename fails (cross-device), try copy
		return copyFile(tmpPath, execPath)
	}

	fmt.Println("Update complete!")
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	fmt.Println("Update complete!")
	return nil
}

// lastCheckPath returns the path to the last check timestamp file
func lastCheckPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cx", lastCheckFile)
}

// shouldCheck returns true if enough time has passed since last check
func shouldCheck() bool {
	data, err := os.ReadFile(lastCheckPath())
	if err != nil {
		return true // No file or error, check anyway
	}

	lastCheck, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		return true
	}

	return time.Since(lastCheck) >= checkInterval
}

// recordCheck saves the current time as the last check time
func recordCheck() {
	path := lastCheckPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte(time.Now().Format(time.RFC3339)), 0644)
}

// isNewer returns true if the release version is newer than current
func isNewer(releaseTag, currentVersion string) bool {
	// Strip 'v' prefix if present
	release := strings.TrimPrefix(releaseTag, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	// Don't update dev builds
	if current == "dev" || current == "" {
		return false
	}

	return release != current
}

// getReleaseVersion fetches the actual version from the VERSION file in the release.
// Falls back to release tag name for backward compatibility with older releases.
func getReleaseVersion(release *Release) string {
	versionURL, err := findVersionAsset(release)
	if err != nil {
		// Backward compatibility: older releases without VERSION file
		// Fall back to tag name (will always trigger update for old clients)
		return release.TagName
	}

	version, err := downloadVersion(versionURL)
	if err != nil || version == "" {
		return release.TagName
	}

	return version
}

// AutoUpdate checks for updates on launch (once per day) and auto-updates if available.
// After updating, it re-executes the new binary with the same arguments.
func AutoUpdate(currentVersion string) {
	if !shouldCheck() {
		return
	}

	release, err := Check()
	if err != nil {
		return // Silent fail - don't interrupt user
	}

	recordCheck()

	releaseVersion := getReleaseVersion(release)
	if !isNewer(releaseVersion, currentVersion) {
		return
	}

	fmt.Printf("Updating cx to %s...\n", releaseVersion)

	if err := SelfUpdate(); err != nil {
		fmt.Fprintf(os.Stderr, "Auto-update failed: %v\n", err)
		return
	}

	// Re-exec the updated binary
	execPath, err := os.Executable()
	if err != nil {
		return
	}

	syscall.Exec(execPath, os.Args, os.Environ())
}
