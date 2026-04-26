package runtime

import (
	"fmt"
	"path/filepath"
	goruntime "runtime"
)

// DownloadProgress reports the progress of a runtime download.
type DownloadProgress struct {
	RuntimeID       string `json:"runtime"`
	BytesDownloaded int64  `json:"bytesDownloaded"`
	TotalBytes      int64  `json:"totalBytes"`
}

// RuntimeProvider is the interface that each runtime (Node.js, Python, etc.) must implement.
type RuntimeProvider interface {
	ID() string
	IsInstalled(runtimeDir string) bool
	Download(runtimeDir string, onProgress func(DownloadProgress)) error
	Verify(runtimeDir string) (string, error) // returns version string
	ExePath(runtimeDir string) string
}

// Manager manages portable runtime installations.
type Manager struct {
	basePath  string // e.g. {appdata}/runtimes
	providers map[string]RuntimeProvider
}

// NewManager creates a new runtime manager with Node.js and Python providers registered.
func NewManager(basePath string) *Manager {
	m := &Manager{
		basePath:  basePath,
		providers: make(map[string]RuntimeProvider),
	}
	m.providers["node"] = &NodeProvider{}
	m.providers["python"] = &PythonProvider{}
	return m
}

// Get returns the provider for the given runtime ID, or an error if not found.
func (m *Manager) Get(id string) (RuntimeProvider, error) {
	p, ok := m.providers[id]
	if !ok {
		return nil, fmt.Errorf("unknown runtime: %s", id)
	}
	return p, nil
}

// RuntimeDir returns the directory where the given runtime should be installed.
func (m *Manager) RuntimeDir(id string) string {
	return filepath.Join(m.basePath, id)
}

// osArch returns the current OS and architecture as reported by the Go runtime.
func osArch() (string, string) {
	os := goruntime.GOOS     // "windows", "darwin", "linux"
	arch := goruntime.GOARCH // "amd64", "arm64"
	return os, arch
}
