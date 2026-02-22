package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	repoOwner  = "kaitwalla"
	repoName   = "cx"
	releaseTag = "latest"
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
