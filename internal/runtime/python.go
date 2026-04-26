package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	pythonVersion = "3.12.8"
	pythonTag     = "20241219"
)

// PythonProvider downloads and manages a python-build-standalone runtime.
type PythonProvider struct{}

func (p *PythonProvider) ID() string { return "python" }

// ExePath returns the path to the python executable within the given runtime directory.
// It first tries to read PYTHON.json for the python_exe field; otherwise falls back
// to a platform-specific default.
func (p *PythonProvider) ExePath(runtimeDir string) string {
	// Try reading PYTHON.json for the canonical path.
	jsonPath := filepath.Join(runtimeDir, "PYTHON.json")
	if data, err := os.ReadFile(jsonPath); err == nil {
		var meta struct {
			PythonExe string `json:"python_exe"`
		}
		if err := json.Unmarshal(data, &meta); err == nil && meta.PythonExe != "" {
			return filepath.Join(runtimeDir, filepath.FromSlash(meta.PythonExe))
		}
	}

	// Fallback.
	osName, _ := osArch()
	if osName == "windows" {
		return filepath.Join(runtimeDir, "python.exe")
	}
	return filepath.Join(runtimeDir, "bin", "python3")
}

// IsInstalled checks whether the python executable exists in the runtime directory.
func (p *PythonProvider) IsInstalled(runtimeDir string) bool {
	_, err := os.Stat(p.ExePath(runtimeDir))
	return err == nil
}

// Download downloads and extracts the python-build-standalone runtime into runtimeDir.
func (p *PythonProvider) Download(runtimeDir string, onProgress func(DownloadProgress)) error {
	osName, arch := osArch()

	// Map Go OS/arch to python-build-standalone naming.
	var pbsOS string
	switch osName {
	case "windows":
		pbsOS = "pc-windows-msvc"
	case "darwin":
		pbsOS = "apple-darwin"
	case "linux":
		pbsOS = "unknown-linux-gnu"
	default:
		return fmt.Errorf("unsupported OS: %s", osName)
	}

	var pbsArch string
	switch arch {
	case "amd64":
		pbsArch = "x86_64"
	case "arm64":
		pbsArch = "aarch64"
	default:
		return fmt.Errorf("unsupported arch: %s", arch)
	}

	// Construct download URL.
	// Example: https://github.com/astral-sh/python-build-standalone/releases/download/20241219/cpython-3.12.8+20241219-x86_64-pc-windows-msvc-install_only_stripped.tar.gz
	fileName := fmt.Sprintf("cpython-%s+%s-%s-%s-install_only_stripped.tar.gz",
		pythonVersion, pythonTag, pbsArch, pbsOS)
	url := fmt.Sprintf("https://github.com/astral-sh/python-build-standalone/releases/download/%s/%s",
		pythonTag, fileName)

	// Create a temporary directory for the download.
	tmpDir, err := os.MkdirTemp("", "python-download-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, "python.tar.gz")

	// Download the archive.
	if err := downloadFile(url, archivePath, "python", onProgress); err != nil {
		return fmt.Errorf("downloading python: %w", err)
	}

	// Extract to temp dir.
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return fmt.Errorf("creating extract dir: %w", err)
	}

	if err := extractTarGz(archivePath, extractDir); err != nil {
		return fmt.Errorf("extracting tar.gz: %w", err)
	}

	// The archive contains a top-level "python/" directory.
	innerDir := filepath.Join(extractDir, "python")
	if _, err := os.Stat(innerDir); err != nil {
		return fmt.Errorf("expected inner directory 'python' not found: %w", err)
	}

	// Ensure parent of runtimeDir exists, then rename.
	if err := os.MkdirAll(filepath.Dir(runtimeDir), 0o755); err != nil {
		return fmt.Errorf("creating runtime parent dir: %w", err)
	}
	// Remove runtimeDir if it exists (e.g. partial previous install).
	os.RemoveAll(runtimeDir)

	if err := os.Rename(innerDir, runtimeDir); err != nil {
		// Rename can fail across volumes; fall back to copy.
		if err := copyDir(innerDir, runtimeDir); err != nil {
			return fmt.Errorf("moving python to runtime dir: %w", err)
		}
	}

	return nil
}

// Verify runs `python --version` and returns the version string.
func (p *PythonProvider) Verify(runtimeDir string) (string, error) {
	cmd := exec.Command(p.ExePath(runtimeDir), "--version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("verifying python: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
