package detect

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoMCPServerFound is returned when no MCP server configuration is detected.
var ErrNoMCPServerFound = errors.New("no MCP server configuration found in this directory")

// DetectedConfig holds the result of scanning a directory for an MCP server.
type DetectedConfig struct {
	DisplayName  string
	Description  string
	RegistryType string   // "npm", "pypi", "go"
	Command      string   // e.g. "node", "npx", "python", "go"
	Args         []string // e.g. ["run", "."] or ["index.js"]
}

// DetectMCPServer inspects a directory to determine if it contains an MCP server
// and returns the detected configuration. It checks for Node.js, Python, and Go projects.
func DetectMCPServer(dir string) (*DetectedConfig, error) {
	// Try Node.js (package.json)
	if cfg, err := detectNode(dir); err == nil {
		return cfg, nil
	}

	// Try Python (pyproject.toml or setup.py)
	if cfg, err := detectPython(dir); err == nil {
		return cfg, nil
	}

	// Try Go (go.mod with mcp-go)
	if cfg, err := detectGo(dir); err == nil {
		return cfg, nil
	}

	return nil, ErrNoMCPServerFound
}

func detectNode(dir string) (*DetectedConfig, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Name         string            `json:"name"`
		Description  string            `json:"description"`
		Main         string            `json:"main"`
		Scripts      map[string]string `json:"scripts"`
		Dependencies map[string]string `json:"dependencies"`
		DevDeps      map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	// Check if this looks like an MCP server by examining dependencies and scripts.
	isMCP := false
	mcpDeps := []string{"@modelcontextprotocol/sdk", "mcp-framework", "@anthropic-ai/sdk"}
	for _, dep := range mcpDeps {
		if _, ok := pkg.Dependencies[dep]; ok {
			isMCP = true
			break
		}
		if _, ok := pkg.DevDeps[dep]; ok {
			isMCP = true
			break
		}
	}

	// Also check scripts for mcp-related keywords.
	if !isMCP {
		for _, script := range pkg.Scripts {
			lower := strings.ToLower(script)
			if strings.Contains(lower, "mcp") || strings.Contains(lower, "stdio") {
				isMCP = true
				break
			}
		}
	}

	if !isMCP {
		return nil, errors.New("not an MCP server")
	}

	displayName := pkg.Name
	if displayName == "" {
		displayName = filepath.Base(dir)
	}

	// Determine the entry point.
	command := "node"
	var args []string
	if _, ok := pkg.Scripts["start"]; ok {
		command = "npm"
		args = []string{"start"}
	} else if pkg.Main != "" {
		args = []string{pkg.Main}
	} else {
		// Fallback: try common entry points.
		for _, entry := range []string{"index.js", "src/index.js", "dist/index.js"} {
			if _, err := os.Stat(filepath.Join(dir, entry)); err == nil {
				args = []string{entry}
				break
			}
		}
		if args == nil {
			args = []string{"."}
		}
	}

	return &DetectedConfig{
		DisplayName:  displayName,
		Description:  pkg.Description,
		RegistryType: "npm",
		Command:      command,
		Args:         args,
	}, nil
}

func detectPython(dir string) (*DetectedConfig, error) {
	// Check pyproject.toml first.
	pyprojectPath := filepath.Join(dir, "pyproject.toml")
	if data, err := os.ReadFile(pyprojectPath); err == nil {
		content := string(data)
		lower := strings.ToLower(content)
		if strings.Contains(lower, "mcp") || strings.Contains(lower, "modelcontextprotocol") {
			name := filepath.Base(dir)
			// Try to extract project name from pyproject.toml.
			for _, line := range strings.Split(content, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "name") && strings.Contains(trimmed, "=") {
					parts := strings.SplitN(trimmed, "=", 2)
					if len(parts) == 2 {
						val := strings.TrimSpace(parts[1])
						val = strings.Trim(val, "\"'")
						if val != "" {
							name = val
						}
					}
				}
			}

			// Determine the command: if there's a __main__.py, use python -m;
			// otherwise use uv run with the project name.
			cmd := "python"
			var pyArgs []string
			if _, err := os.Stat(filepath.Join(dir, "__main__.py")); err == nil {
				pyArgs = []string{"-m", name}
			} else if _, err := os.Stat(filepath.Join(dir, "src", name, "__main__.py")); err == nil {
				pyArgs = []string{"-m", name}
			} else {
				// Fallback: assume the project defines an entry point via pyproject.toml.
				cmd = "uv"
				pyArgs = []string{"run", name}
			}

			return &DetectedConfig{
				DisplayName:  name,
				Description:  "",
				RegistryType: "pypi",
				Command:      cmd,
				Args:         pyArgs,
			}, nil
		}
	}

	// Check setup.py.
	setupPath := filepath.Join(dir, "setup.py")
	if data, err := os.ReadFile(setupPath); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "mcp") || strings.Contains(content, "modelcontextprotocol") {
			return &DetectedConfig{
				DisplayName:  filepath.Base(dir),
				RegistryType: "pypi",
				Command:      "python",
				Args:         []string{"-m", filepath.Base(dir)},
			}, nil
		}
	}

	return nil, errors.New("not a Python MCP server")
}

func detectGo(dir string) (*DetectedConfig, error) {
	goModPath := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}

	content := string(data)
	// Match known Go MCP SDK modules.
	isMCP := strings.Contains(content, "mcp-go") ||
		strings.Contains(content, "mark3labs/mcp") ||
		strings.Contains(content, "modelcontextprotocol/go-sdk")
	if !isMCP {
		return nil, errors.New("not a Go MCP server")
	}

	// Extract module name for display.
	name := filepath.Base(dir)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				// Use the last path segment of the module as the display name.
				modParts := strings.Split(parts[1], "/")
				name = modParts[len(modParts)-1]
			}
			break
		}
	}

	return &DetectedConfig{
		DisplayName:  name,
		RegistryType: "go",
		Command:      "go",
		Args:         []string{"run", "."},
	}, nil
}
