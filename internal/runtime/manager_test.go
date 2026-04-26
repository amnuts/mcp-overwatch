package runtime

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"
)

func TestNodeExePath(t *testing.T) {
	n := &NodeProvider{}
	runtimeDir := "/tmp/test-node"

	path := n.ExePath(runtimeDir)

	if goruntime.GOOS == "windows" {
		expected := filepath.Join(runtimeDir, "node.exe")
		if path != expected {
			t.Errorf("expected %s, got %s", expected, path)
		}
	} else {
		expected := filepath.Join(runtimeDir, "bin", "node")
		if path != expected {
			t.Errorf("expected %s, got %s", expected, path)
		}
	}
}

func TestNodeNpxPath(t *testing.T) {
	n := &NodeProvider{}
	runtimeDir := "/tmp/test-node"

	path := n.NpxPath(runtimeDir)

	if goruntime.GOOS == "windows" {
		expected := filepath.Join(runtimeDir, "npx.cmd")
		if path != expected {
			t.Errorf("expected %s, got %s", expected, path)
		}
	} else {
		expected := filepath.Join(runtimeDir, "bin", "npx")
		if path != expected {
			t.Errorf("expected %s, got %s", expected, path)
		}
	}
}

func TestNodeIsInstalled(t *testing.T) {
	n := &NodeProvider{}

	tmpDir := t.TempDir()

	// Not installed yet.
	if n.IsInstalled(tmpDir) {
		t.Error("expected IsInstalled to return false for empty dir")
	}

	// Create fake exe.
	exePath := n.ExePath(tmpDir)
	if err := os.MkdirAll(filepath.Dir(exePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	if !n.IsInstalled(tmpDir) {
		t.Error("expected IsInstalled to return true after creating exe")
	}
}

func TestPythonExePath(t *testing.T) {
	p := &PythonProvider{}
	tmpDir := t.TempDir()

	// Without PYTHON.json, should return fallback path.
	path := p.ExePath(tmpDir)

	if goruntime.GOOS == "windows" {
		expected := filepath.Join(tmpDir, "python.exe")
		if path != expected {
			t.Errorf("expected %s, got %s", expected, path)
		}
	} else {
		expected := filepath.Join(tmpDir, "bin", "python3")
		if path != expected {
			t.Errorf("expected %s, got %s", expected, path)
		}
	}
}

func TestPythonExePathWithJSON(t *testing.T) {
	p := &PythonProvider{}
	tmpDir := t.TempDir()

	// Write a PYTHON.json with python_exe field.
	jsonContent := `{"python_exe": "bin/python3.12"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "PYTHON.json"), []byte(jsonContent), 0o644); err != nil {
		t.Fatal(err)
	}

	path := p.ExePath(tmpDir)
	expected := filepath.Join(tmpDir, "bin", "python3.12")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestPythonIsInstalled(t *testing.T) {
	p := &PythonProvider{}

	tmpDir := t.TempDir()

	// Not installed yet.
	if p.IsInstalled(tmpDir) {
		t.Error("expected IsInstalled to return false for empty dir")
	}

	// Create fake exe.
	exePath := p.ExePath(tmpDir)
	if err := os.MkdirAll(filepath.Dir(exePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	if !p.IsInstalled(tmpDir) {
		t.Error("expected IsInstalled to return true after creating exe")
	}
}

func TestManagerGet(t *testing.T) {
	m := NewManager("/tmp/runtimes")

	// Known providers.
	node, err := m.Get("node")
	if err != nil {
		t.Fatalf("expected no error for 'node', got %v", err)
	}
	if node.ID() != "node" {
		t.Errorf("expected ID 'node', got %s", node.ID())
	}

	python, err := m.Get("python")
	if err != nil {
		t.Fatalf("expected no error for 'python', got %v", err)
	}
	if python.ID() != "python" {
		t.Errorf("expected ID 'python', got %s", python.ID())
	}

	// Unknown provider.
	_, err = m.Get("ruby")
	if err == nil {
		t.Error("expected error for unknown runtime 'ruby'")
	}
}

func TestManagerRuntimeDir(t *testing.T) {
	m := NewManager("/tmp/runtimes")

	dir := m.RuntimeDir("node")
	expected := filepath.Join("/tmp/runtimes", "node")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestDownloadNode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	n := &NodeProvider{}
	tmpDir := t.TempDir()
	runtimeDir := filepath.Join(tmpDir, "node")

	err := n.Download(runtimeDir, func(p DownloadProgress) {
		// Progress callback — just verify it's called.
	})
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}

	if !n.IsInstalled(runtimeDir) {
		t.Error("node should be installed after download")
	}

	version, err := n.Verify(runtimeDir)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if version != "v"+nodeVersion {
		t.Errorf("expected version v%s, got %s", nodeVersion, version)
	}
}

func TestDownloadPython(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	p := &PythonProvider{}
	tmpDir := t.TempDir()
	runtimeDir := filepath.Join(tmpDir, "python")

	err := p.Download(runtimeDir, func(prog DownloadProgress) {
		// Progress callback — just verify it's called.
	})
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}

	if !p.IsInstalled(runtimeDir) {
		t.Error("python should be installed after download")
	}

	version, err := p.Verify(runtimeDir)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	expectedPrefix := "Python " + pythonVersion
	if version != expectedPrefix {
		t.Errorf("expected version %s, got %s", expectedPrefix, version)
	}
}
