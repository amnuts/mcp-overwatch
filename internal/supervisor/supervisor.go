package supervisor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// StatusCallback is invoked whenever a managed server's status changes.
type StatusCallback func(serverID string, status ServerStatus)

// LogCallback is invoked to relay log messages from managed servers.
type LogCallback func(serverID string, stream string, message string)

// Supervisor manages the lifecycle of MCP server subprocesses, including
// starting, stopping, health checking, and crash recovery with exponential backoff.
type Supervisor struct {
	servers  map[string]*ManagedServer
	mu       sync.RWMutex
	onStatus StatusCallback
	onLog    LogCallback
	ctx      context.Context
	cancel   context.CancelFunc
}

// New creates a Supervisor with the given status and log callbacks.
func New(onStatus StatusCallback, onLog LogCallback) *Supervisor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Supervisor{
		servers:  make(map[string]*ManagedServer),
		onStatus: onStatus,
		onLog:    onLog,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Register adds a ManagedServer to the supervisor. It does not start the server.
func (s *Supervisor) Register(ms *ManagedServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ms.status == "" {
		ms.status = StatusStopped
	}
	s.servers[ms.ID] = ms
}

// Get returns the ManagedServer with the given ID, or nil if not found.
func (s *Supervisor) Get(serverID string) *ManagedServer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.servers[serverID]
}

// ListActive returns all servers currently in the Running state.
func (s *Supervisor) ListActive() []*ManagedServer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var active []*ManagedServer
	for _, ms := range s.servers {
		ms.mu.Lock()
		status := ms.status
		ms.mu.Unlock()
		if status == StatusRunning {
			active = append(active, ms)
		}
	}
	return active
}

// Start connects to the MCP server using its configured transport, initializes
// the MCP protocol, caches tools/resources/prompts, and starts health check and
// (for stdio) stderr reader goroutines.
func (s *Supervisor) Start(serverID string) error {
	s.mu.RLock()
	ms, ok := s.servers[serverID]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("unknown server: %s", serverID)
	}

	// Validate transport config up front so callers can't crash us on nil pipes
	// or empty URLs.
	if err := validateTransport(ms); err != nil {
		ms.mu.Lock()
		ms.status = StatusError
		ms.mu.Unlock()
		s.onStatus(ms.ID, StatusError)
		s.log(ms.ID, "lifecycle", fmt.Sprintf("start aborted: %v", err))
		return fmt.Errorf("cannot start %s: %w", ms.ID, err)
	}

	// Set status to starting — release lock before blocking I/O
	ms.mu.Lock()
	ms.status = StatusStarting
	ms.stopCh = make(chan struct{})
	ms.mu.Unlock()
	s.onStatus(ms.ID, StatusStarting)
	s.log(ms.ID, "lifecycle", fmt.Sprintf("starting %s transport", transportKind(ms)))

	// Build the client for the configured transport. stdio auto-starts the
	// underlying transport; remote transports require an explicit Start.
	c, needsStart, err := buildClient(ms)
	if err != nil {
		ms.mu.Lock()
		ms.status = StatusError
		ms.mu.Unlock()
		s.onStatus(ms.ID, StatusError)
		s.log(ms.ID, "lifecycle", fmt.Sprintf("start failed: %v", err))
		return fmt.Errorf("failed to start %s: %w", ms.ID, err)
	}

	ctx := context.Background()
	if needsStart {
		startCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		if err := c.Start(startCtx); err != nil {
			cancel()
			_ = c.Close()
			ms.mu.Lock()
			ms.status = StatusError
			ms.mu.Unlock()
			s.onStatus(ms.ID, StatusError)
			s.log(ms.ID, "lifecycle", fmt.Sprintf("connect failed: %v", err))
			return fmt.Errorf("failed to connect %s: %w", ms.ID, err)
		}
		cancel()
	}

	// Initialize MCP protocol
	initReq := mcp.InitializeRequest{}
	initReq.Params.ClientInfo = mcp.Implementation{Name: "mcp-overwatch", Version: "1.0.0"}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	_, err = c.Initialize(ctx, initReq)
	if err != nil {
		c.Close()
		ms.mu.Lock()
		ms.status = StatusError
		ms.mu.Unlock()
		s.onStatus(ms.ID, StatusError)
		s.log(ms.ID, "lifecycle", fmt.Sprintf("initialize failed: %v", err))
		return fmt.Errorf("failed to initialize %s: %w", ms.ID, err)
	}

	// Cache tools (best-effort)
	var tools []mcp.Tool
	if toolsResult, toolsErr := c.ListTools(ctx, mcp.ListToolsRequest{}); toolsErr == nil {
		tools = toolsResult.Tools
	}

	// Cache resources (best-effort)
	var resources []mcp.Resource
	if resourcesResult, resErr := c.ListResources(ctx, mcp.ListResourcesRequest{}); resErr == nil {
		resources = resourcesResult.Resources
	}

	// Cache prompts (best-effort)
	var prompts []mcp.Prompt
	if promptsResult, promptsErr := c.ListPrompts(ctx, mcp.ListPromptsRequest{}); promptsErr == nil {
		prompts = promptsResult.Prompts
	}

	// Check if stop was requested during initialization (e.g. user toggled off
	// while the process was still starting). The stopCh is closed by Stop().
	ms.mu.Lock()
	select {
	case <-ms.stopCh:
		ms.mu.Unlock()
		c.Close()
		s.log(ms.ID, "lifecycle", "start cancelled — server was stopped during initialization")
		return fmt.Errorf("server %s was stopped during initialization", ms.ID)
	default:
	}

	// Commit final state
	ms.client = c
	ms.tools = tools
	ms.resources = resources
	ms.prompts = prompts
	ms.status = StatusRunning
	ms.errorCount = 0
	ms.lastUsed = time.Now()
	ms.mu.Unlock()
	s.onStatus(ms.ID, StatusRunning)
	s.log(ms.ID, "lifecycle", fmt.Sprintf("started — %d tool(s), %d resource(s), %d prompt(s)", len(tools), len(resources), len(prompts)))

	// Start health check goroutine
	go s.healthCheck(ms)

	// Start stderr reader goroutine
	go s.readStderr(ms)

	return nil
}

// Stop gracefully shuts down the MCP server subprocess.
func (s *Supervisor) Stop(serverID string) error {
	s.mu.RLock()
	ms, ok := s.servers[serverID]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("unknown server: %s", serverID)
	}

	ms.mu.Lock()
	if ms.status == StatusStopped {
		ms.mu.Unlock()
		return nil
	}
	if ms.stopCh != nil {
		select {
		case <-ms.stopCh:
			// already closed
		default:
			close(ms.stopCh)
		}
		ms.stopCh = nil
	}
	c := ms.client
	ms.client = nil
	ms.status = StatusStopped
	ms.mu.Unlock()

	if c != nil {
		c.Close()
	}
	s.onStatus(ms.ID, StatusStopped)
	s.log(ms.ID, "lifecycle", "stopped")
	return nil
}

// Restart stops then starts the server.
func (s *Supervisor) Restart(serverID string) error {
	s.log(serverID, "lifecycle", "restart requested")
	if err := s.Stop(serverID); err != nil {
		s.log(serverID, "lifecycle", fmt.Sprintf("restart failed during stop: %v", err))
		return err
	}
	return s.Start(serverID)
}

// Shutdown stops all servers and cancels the supervisor context.
func (s *Supervisor) Shutdown() {
	s.mu.RLock()
	ids := make([]string, 0, len(s.servers))
	for id := range s.servers {
		ids = append(ids, id)
	}
	s.mu.RUnlock()

	if len(ids) > 0 {
		s.log("system", "lifecycle", fmt.Sprintf("shutting down — stopping %d server(s)", len(ids)))
	}
	for _, id := range ids {
		_ = s.Stop(id)
	}
	s.cancel()
}

// log forwards a lifecycle message via the configured log callback, if any.
func (s *Supervisor) log(serverID, stream, message string) {
	if s.onLog != nil {
		s.onLog(serverID, stream, message)
	}
}
