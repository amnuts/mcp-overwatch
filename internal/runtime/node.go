package runtime

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const nodeVersion = "22.12.0"

// NodeProvider downloads and manages a portable Node.js runtime.
type NodeProvider struct{}

func (n *NodeProvider) ID() string { return "node" }

// ExePath returns the path to the node executable within the given runtime directory.
func (n *NodeProvider) ExePath(runtimeDir string) string {
	os, _ := osArch()
	if os == "windows" {
		return filepath.Join(runtimeDir, "node.exe")
	}
	return filepath.Join(runtimeDir, "bin", "node")
}

// NpxPath returns the path to the npx executable within the given runtime directory.
func (n *NodeProvider) NpxPath(runtimeDir string) string {
	os, _ := osArch()
	if os == "windows" {
		return filepath.Join(runtimeDir, "npx.cmd")
	}
	return filepath.Join(runtimeDir, "bin", "npx")
}

// IsInstalled checks whether the node executable exists in the runtime directory.
func (n *NodeProvider) IsInstalled(runtimeDir string) bool {
	_, err := os.Stat(n.ExePath(runtimeDir))
	return err == nil
}

// Download downloads and extracts the Node.js runtime into runtimeDir.
func (n *NodeProvider) Download(runtimeDir string, onProgress func(DownloadProgress)) error {
	osName, arch := osArch()

	// Map Go OS/arch to Node.js naming conventions.
	nodeOS := osName
	if nodeOS == "windows" {
		nodeOS = "win"
	}
	nodeArch := arch
	if nodeArch == "amd64" {
		nodeArch = "x64"
	}

	ext := "tar.gz"
	if osName == "windows" {
		ext = "zip"
	}

	dirName := fmt.Sprintf("node-v%s-%s-%s", nodeVersion, nodeOS, nodeArch)
	url := fmt.Sprintf("https://nodejs.org/dist/v%s/%s.%s", nodeVersion, dirName, ext)

	// Create a temporary directory for the download.
	tmpDir, err := os.MkdirTemp("", "node-download-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, fmt.Sprintf("node.%s", ext))

	// Download the archive.
	if err := downloadFile(url, archivePath, "node", onProgress); err != nil {
		return fmt.Errorf("downloading node: %w", err)
	}

	// Extract to temp dir.
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return fmt.Errorf("creating extract dir: %w", err)
	}

	if osName == "windows" {
		if err := extractZip(archivePath, extractDir); err != nil {
			return fmt.Errorf("extracting zip: %w", err)
		}
	} else {
		if err := extractTarGz(archivePath, extractDir); err != nil {
			return fmt.Errorf("extracting tar.gz: %w", err)
		}
	}

	// The archive contains a top-level directory like node-v22.12.0-win-x64/.
	// Move its contents to runtimeDir.
	innerDir := filepath.Join(extractDir, dirName)
	if _, err := os.Stat(innerDir); err != nil {
		return fmt.Errorf("expected inner directory %s not found: %w", dirName, err)
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
			return fmt.Errorf("moving node to runtime dir: %w", err)
		}
	}

	return nil
}

// Verify runs `node --version` and returns the version string.
func (n *NodeProvider) Verify(runtimeDir string) (string, error) {
	cmd := exec.Command(n.ExePath(runtimeDir), "--version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("verifying node: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// downloadFile downloads a URL to a local file path, reporting progress via onProgress.
func downloadFile(url, destPath, runtimeID string, onProgress func(DownloadProgress)) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	totalBytes, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			if onProgress != nil {
				onProgress(DownloadProgress{
					RuntimeID:       runtimeID,
					BytesDownloaded: downloaded,
					TotalBytes:      totalBytes,
				})
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	return nil
}

// extractZip extracts a zip archive to destDir.
func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)

		// Prevent zip slip.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// extractTarGz extracts a .tar.gz archive to destDir.
func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, hdr.Name)

		// Prevent path traversal.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in tar: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyDir recursively copies a directory from src to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		// Check for symlink.
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, targetPath)
		}

		return copyFile(path, targetPath, info.Mode())
	})
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
