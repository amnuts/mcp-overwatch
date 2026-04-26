package supervisor

import (
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func newTestCallbacks() (StatusCallback, LogCallback, *[]string) {
	var logs []string
	onStatus := func(serverID string, status ServerStatus) {
		logs = append(logs, "status:"+serverID+":"+string(status))
	}
	onLog := func(serverID string, stream string, message string) {
		logs = append(logs, "log:"+serverID+":"+stream+":"+message)
	}
	return onStatus, onLog, &logs
}

func TestRegisterAndGet(t *testing.T) {
	onStatus, onLog, _ := newTestCallbacks()
	sup := New(onStatus, onLog)
	defer sup.Shutdown()

	ms := &ManagedServer{
		ID:          "test-server",
		DisplayName: "Test Server",
		Command:     "echo",
		Args:        []string{"hello"},
	}

	sup.Register(ms)

	got := sup.Get("test-server")
	if got == nil {
		t.Fatal("expected to get registered server, got nil")
	}
	if got.ID != "test-server" {
		t.Errorf("expected ID 'test-server', got %q", got.ID)
	}
	if got.Status() != StatusStopped {
		t.Errorf("expected status Stopped, got %q", got.Status())
	}

	// Non-existent server should return nil
	if sup.Get("nonexistent") != nil {
		t.Error("expected nil for non-existent server")
	}
}

func TestListActive(t *testing.T) {
	onStatus, onLog, _ := newTestCallbacks()
	sup := New(onStatus, onLog)
	defer sup.Shutdown()

	servers := []*ManagedServer{
		{ID: "server-1", DisplayName: "Server 1", Command: "echo"},
		{ID: "server-2", DisplayName: "Server 2", Command: "echo"},
		{ID: "server-3", DisplayName: "Server 3", Command: "echo"},
	}

	for _, ms := range servers {
		sup.Register(ms)
	}

	// No servers running initially
	active := sup.ListActive()
	if len(active) != 0 {
		t.Errorf("expected 0 active servers, got %d", len(active))
	}

	// Manually set some to running for testing
	servers[0].mu.Lock()
	servers[0].status = StatusRunning
	servers[0].mu.Unlock()

	servers[2].mu.Lock()
	servers[2].status = StatusRunning
	servers[2].mu.Unlock()

	active = sup.ListActive()
	if len(active) != 2 {
		t.Errorf("expected 2 active servers, got %d", len(active))
	}

	// Verify the correct servers are returned
	ids := make(map[string]bool)
	for _, a := range active {
		ids[a.ID] = true
	}
	if !ids["server-1"] || !ids["server-3"] {
		t.Errorf("expected server-1 and server-3 in active list, got %v", ids)
	}
}

func TestStopAlreadyStopped(t *testing.T) {
	onStatus, onLog, _ := newTestCallbacks()
	sup := New(onStatus, onLog)
	defer sup.Shutdown()

	ms := &ManagedServer{
		ID:      "stopped-server",
		Command: "echo",
	}
	sup.Register(ms)

	// Stop an already-stopped server should be a no-op
	err := sup.Stop("stopped-server")
	if err != nil {
		t.Errorf("expected no error stopping an already-stopped server, got: %v", err)
	}
	if ms.Status() != StatusStopped {
		t.Errorf("expected status Stopped, got %q", ms.Status())
	}
}

func TestStopUnknownServer(t *testing.T) {
	onStatus, onLog, _ := newTestCallbacks()
	sup := New(onStatus, onLog)
	defer sup.Shutdown()

	err := sup.Stop("nonexistent")
	if err == nil {
		t.Error("expected error stopping unknown server")
	}
}

func TestStartUnknownServer(t *testing.T) {
	onStatus, onLog, _ := newTestCallbacks()
	sup := New(onStatus, onLog)
	defer sup.Shutdown()

	err := sup.Start("nonexistent")
	if err == nil {
		t.Error("expected error starting unknown server")
	}
}

func TestManagedServerAccessors(t *testing.T) {
	now := time.Now()
	ms := &ManagedServer{
		ID:          "accessor-test",
		DisplayName: "Accessor Test",
		Command:     "test-cmd",
		Args:        []string{"--flag"},
		Env:         []string{"KEY=VAL"},
		status:      StatusRunning,
		tools: []mcp.Tool{
			mcp.NewTool("tool1"),
		},
		resources: []mcp.Resource{
			mcp.NewResource("file:///test", "test-resource"),
		},
		prompts: []mcp.Prompt{
			mcp.NewPrompt("prompt1"),
		},
		lastUsed:   now,
		errorCount: 3,
	}

	if ms.Status() != StatusRunning {
		t.Errorf("expected StatusRunning, got %q", ms.Status())
	}

	tools := ms.Tools()
	if len(tools) != 1 || tools[0].Name != "tool1" {
		t.Errorf("unexpected tools: %+v", tools)
	}

	resources := ms.Resources()
	if len(resources) != 1 || resources[0].Name != "test-resource" {
		t.Errorf("unexpected resources: %+v", resources)
	}

	prompts := ms.Prompts()
	if len(prompts) != 1 || prompts[0].Name != "prompt1" {
		t.Errorf("unexpected prompts: %+v", prompts)
	}

	if !ms.LastUsed().Equal(now) {
		t.Errorf("expected lastUsed %v, got %v", now, ms.LastUsed())
	}

	if ms.ErrorCount() != 3 {
		t.Errorf("expected errorCount 3, got %d", ms.ErrorCount())
	}

	// Verify accessors return copies (mutating returned slice should not affect original)
	tools[0] = mcp.NewTool("mutated")
	if ms.Tools()[0].Name == "mutated" {
		t.Error("Tools() should return a copy, not a reference to internal slice")
	}
}

func TestErrorCountIncrement(t *testing.T) {
	ms := &ManagedServer{
		ID:      "error-test",
		Command: "test-cmd",
		status:  StatusRunning,
	}

	if ms.ErrorCount() != 0 {
		t.Errorf("expected initial errorCount 0, got %d", ms.ErrorCount())
	}

	// Simulate incrementing error count (as handleCrash would)
	ms.mu.Lock()
	ms.errorCount++
	ms.mu.Unlock()
	if ms.ErrorCount() != 1 {
		t.Errorf("expected errorCount 1, got %d", ms.ErrorCount())
	}

	ms.mu.Lock()
	ms.errorCount++
	ms.mu.Unlock()
	if ms.ErrorCount() != 2 {
		t.Errorf("expected errorCount 2, got %d", ms.ErrorCount())
	}

	// Simulate max retries exceeded
	ms.mu.Lock()
	ms.errorCount = 5
	ms.status = StatusError
	ms.mu.Unlock()
	if ms.Status() != StatusError {
		t.Errorf("expected StatusError after max retries, got %q", ms.Status())
	}
	if ms.ErrorCount() != 5 {
		t.Errorf("expected errorCount 5, got %d", ms.ErrorCount())
	}
}

func TestShutdown(t *testing.T) {
	onStatus, onLog, _ := newTestCallbacks()
	sup := New(onStatus, onLog)

	ms1 := &ManagedServer{ID: "s1", Command: "echo", status: StatusStopped}
	ms2 := &ManagedServer{ID: "s2", Command: "echo", status: StatusStopped}
	sup.Register(ms1)
	sup.Register(ms2)

	// Shutdown should not panic even with stopped servers
	sup.Shutdown()

	// Context should be cancelled after shutdown
	select {
	case <-sup.ctx.Done():
		// expected
	default:
		t.Error("expected supervisor context to be cancelled after shutdown")
	}
}

func TestHandleCrashMaxRetries(t *testing.T) {
	onStatus, onLog, logs := newTestCallbacks()
	sup := New(onStatus, onLog)
	defer sup.Shutdown()

	ms := &ManagedServer{
		ID:         "crash-test",
		Command:    "nonexistent",
		status:     StatusRunning,
		errorCount: 4, // Next increment will hit 5
		stopCh:     make(chan struct{}),
	}
	sup.Register(ms)

	// Call handleCrash — should hit max retries immediately
	sup.handleCrash(ms)

	if ms.Status() != StatusError {
		t.Errorf("expected StatusError after max retries, got %q", ms.Status())
	}
	if ms.ErrorCount() != 5 {
		t.Errorf("expected errorCount 5, got %d", ms.ErrorCount())
	}

	// Verify status callback was fired
	found := false
	for _, log := range *logs {
		if log == "status:crash-test:error" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected status callback for error state, logs: %v", *logs)
	}
}
