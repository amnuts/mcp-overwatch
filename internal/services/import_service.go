package services

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/wailsapp/wails/v3/pkg/application"

	"mcp-overwatch/internal/catalogue"
	"mcp-overwatch/internal/detect"
	"mcp-overwatch/internal/logging"
	"mcp-overwatch/internal/paths"
)

// ImportService handles importing MCP servers from GitHub repos and local directories.
type ImportService struct {
	store    *catalogue.Store
	paths    *paths.Paths
	logger   *logging.Logger
	wailsApp *application.App
}

// NewImportService creates an ImportService with the given dependencies.
func NewImportService(store *catalogue.Store, p *paths.Paths, logger *logging.Logger) *ImportService {
	return &ImportService{store: store, paths: p, logger: logger}
}

// SetWailsApp sets the Wails application reference for dialog support.
func (s *ImportService) SetWailsApp(app *application.App) {
	s.wailsApp = app
}

// gitHubURLPattern matches GitHub repository URLs, optionally with /tree/{branch}/{subpath}.
var gitHubURLPattern = regexp.MustCompile(`^(?:https?://)?(?:www\.)?github\.com/([^/]+)/([^/]+?)(?:\.git)?(?:/tree/([^/]+)(?:/(.+?))?)?/?$`)

// logErr logs an import error and returns it unchanged.
func (s *ImportService) logErr(context string, err error) error {
	s.logger.Add(logging.Entry{
		ServerID:  "system",
		Direction: "in",
		Summary:   fmt.Sprintf("Import failed (%s): %s", context, err),
	})
	return err
}

// ImportFromGitHub clones a GitHub repository and imports it as an MCP server.
func (s *ImportService) ImportFromGitHub(rawURL string) (*catalogue.InstalledServer, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, s.logErr("github", errors.New("please enter a GitHub repository URL"))
	}

	// Normalize the URL.
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	// Validate URL format.
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return nil, s.logErr("github", errors.New("please enter a valid GitHub repository URL (e.g., https://github.com/owner/repo)"))
	}

	matches := gitHubURLPattern.FindStringSubmatch(rawURL)
	if matches == nil {
		return nil, s.logErr("github", fmt.Errorf("invalid GitHub URL: %s", rawURL))
	}
	owner := matches[1]
	repo := matches[2]
	// branch (matches[3]) is used for clone ref but not required.
	subPath := ""
	if len(matches) > 4 {
		subPath = strings.TrimSuffix(matches[4], "/")
	}

	// Build clone URL.
	cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)

	// Destination directory — reuse existing clone if importing a different subpath.
	repoDir := filepath.Join(s.paths.Repos(), fmt.Sprintf("%s-%s", owner, repo))

	alreadyCloned := false
	if _, err := os.Stat(repoDir); err == nil {
		if subPath == "" {
			return nil, s.logErr("github", fmt.Errorf("repository already imported at %s — uninstall the existing server first", repoDir))
		}
		alreadyCloned = true
	}

	if !alreadyCloned {
		s.logger.Add(logging.Entry{
			ServerID:  "system",
			Direction: "out",
			Summary:   fmt.Sprintf("Cloning %s/%s from GitHub...", owner, repo),
		})

		// Clone the repository.
		_, err = git.PlainClone(repoDir, false, &git.CloneOptions{
			URL:   cloneURL,
			Depth: 1,
		})
		if err != nil {
			_ = os.RemoveAll(repoDir)
			if strings.Contains(err.Error(), "authentication") || strings.Contains(err.Error(), "not found") {
				return nil, s.logErr("github", fmt.Errorf("repository not found — check the URL and ensure the repository is public"))
			}
			return nil, s.logErr("github", fmt.Errorf("clone failed: %w", err))
		}
	}

	// Determine the detection directory — subpath within the repo if specified.
	detectDir := repoDir
	if subPath != "" {
		detectDir = filepath.Join(repoDir, filepath.FromSlash(subPath))
		if _, err := os.Stat(detectDir); err != nil {
			if !alreadyCloned {
				_ = os.RemoveAll(repoDir)
			}
			return nil, s.logErr("github", fmt.Errorf("subdirectory not found in repository: %s", subPath))
		}
	}

	// Detect MCP server configuration.
	detected, err := detect.DetectMCPServer(detectDir)
	if err != nil {
		if !alreadyCloned {
			_ = os.RemoveAll(repoDir)
		}
		return nil, s.logErr("github", errors.New("no MCP server configuration found — the repository must contain a package.json, pyproject.toml, or Go module with MCP SDK dependencies"))
	}

	// Create server ID from owner/repo and optional subpath.
	serverID := fmt.Sprintf("github-%s-%s", owner, repo)
	if subPath != "" {
		// Use the last segment of the subpath for a readable ID.
		subName := filepath.Base(subPath)
		serverID = fmt.Sprintf("github-%s-%s-%s", owner, repo, subName)
	}

	// Check for duplicate.
	if existing, _ := s.store.GetInstalledServer(serverID); existing != nil {
		if !alreadyCloned {
			_ = os.RemoveAll(repoDir)
		}
		return nil, s.logErr("github", fmt.Errorf("a server with ID %q is already installed", serverID))
	}

	srv := catalogue.InstalledServer{
		ID:              serverID,
		Source:          "github",
		DisplayName:     detected.DisplayName,
		Description:     detected.Description,
		RegistryType:    detected.RegistryType,
		TransportType:   "stdio",
		Command:         detected.Command,
		CommandArgsJSON: mustJSON(detected.Args),
		RuntimePath:     detectDir,
		IsActive:        false,
		Status:          "stopped",
		InstalledAt:     time.Now().UTC(),
	}

	if err := s.store.InsertInstalledServer(srv); err != nil {
		_ = os.RemoveAll(repoDir)
		return nil, s.logErr("github", fmt.Errorf("saving server: %w", err))
	}

	s.logger.Add(logging.Entry{
		ServerID:  serverID,
		Direction: "in",
		Summary:   fmt.Sprintf("Imported %s from GitHub (%s)", detected.DisplayName, detected.RegistryType),
	})

	return &srv, nil
}

// ImportFromLocal imports an MCP server from a local directory.
// If copyToOverwatch is true, the directory is copied into the MCP Overwatch data directory.
func (s *ImportService) ImportFromLocal(dir string, copyToOverwatch bool) (*catalogue.InstalledServer, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, s.logErr("local", errors.New("please select a directory"))
	}

	// Validate directory exists.
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, s.logErr("local", fmt.Errorf("directory does not exist: %s", dir))
		}
		return nil, s.logErr("local", fmt.Errorf("cannot access directory: %w", err))
	}
	if !info.IsDir() {
		return nil, s.logErr("local", fmt.Errorf("path is not a directory: %s", dir))
	}

	targetDir := dir
	baseName := filepath.Base(dir)

	s.logger.Add(logging.Entry{
		ServerID:  "system",
		Direction: "out",
		Summary:   fmt.Sprintf("Importing local directory %s (copy=%t)", dir, copyToOverwatch),
	})

	if copyToOverwatch {
		targetDir = filepath.Join(s.paths.Repos(), baseName)
		if _, err := os.Stat(targetDir); err == nil {
			return nil, s.logErr("local", fmt.Errorf("directory already exists in MCP Overwatch: %s — choose a different name or remove the existing one", baseName))
		}
		if err := copyDir(dir, targetDir); err != nil {
			_ = os.RemoveAll(targetDir)
			return nil, s.logErr("local", fmt.Errorf("copying directory: %w", err))
		}
	}

	// Detect MCP server configuration.
	detected, err := detect.DetectMCPServer(targetDir)
	if err != nil {
		if copyToOverwatch {
			_ = os.RemoveAll(targetDir)
		}
		return nil, s.logErr("local", errors.New("no MCP server configuration found — the directory must contain a package.json, pyproject.toml, or Go module with MCP SDK dependencies"))
	}

	// Create server ID.
	serverID := fmt.Sprintf("local-%s", baseName)

	// Check for duplicate.
	if existing, _ := s.store.GetInstalledServer(serverID); existing != nil {
		if copyToOverwatch {
			_ = os.RemoveAll(targetDir)
		}
		return nil, s.logErr("local", fmt.Errorf("a server with ID %q is already installed", serverID))
	}

	srv := catalogue.InstalledServer{
		ID:              serverID,
		Source:          "local",
		DisplayName:     detected.DisplayName,
		Description:     detected.Description,
		RegistryType:    detected.RegistryType,
		TransportType:   "stdio",
		Command:         detected.Command,
		CommandArgsJSON: mustJSON(detected.Args),
		RuntimePath:     targetDir,
		IsActive:        false,
		Status:          "stopped",
		InstalledAt:     time.Now().UTC(),
	}

	if err := s.store.InsertInstalledServer(srv); err != nil {
		if copyToOverwatch {
			_ = os.RemoveAll(targetDir)
		}
		return nil, s.logErr("local", fmt.Errorf("saving server: %w", err))
	}

	s.logger.Add(logging.Entry{
		ServerID:  serverID,
		Direction: "in",
		Summary:   fmt.Sprintf("Imported %s from local directory (%s)", detected.DisplayName, detected.RegistryType),
	})

	return &srv, nil
}

// BrowseDirectory opens a native directory picker dialog and returns the selected path.
func (s *ImportService) BrowseDirectory() (string, error) {
	if s.wailsApp == nil {
		return "", errors.New("application not initialized")
	}
	dialog := application.OpenFileDialogStruct{}
	result, err := dialog.
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle("Select MCP Server Directory").
		PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("dialog error: %w", err)
	}
	return result, nil
}

// copyDir recursively copies a directory from src to dst.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// Skip common non-essential directories.
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "node_modules" || name == "__pycache__" || name == ".venv" || name == "venv" {
				continue
			}
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// Note: mustJSON is defined in server_service.go and shared across this package.
