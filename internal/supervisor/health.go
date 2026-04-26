package supervisor

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/client"
)

// healthCheck periodically pings the MCP server to verify it is still responsive.
// On failure, it triggers crash recovery.
func (s *Supervisor) healthCheck(ms *ManagedServer) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ms.stopCh:
			return
		case <-ticker.C:
			ms.mu.Lock()
			c := ms.client
			ms.mu.Unlock()
			if c == nil {
				return
			}
			ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
			err := c.Ping(ctx)
			cancel()
			if err != nil {
				s.log(ms.ID, "lifecycle", "health check failed: "+err.Error())
				go s.handleCrash(ms)
				return
			}
		}
	}
}

// readStderr continuously reads from the MCP server subprocess's stderr
// and relays output via the log callback.
func (s *Supervisor) readStderr(ms *ManagedServer) {
	ms.mu.Lock()
	c := ms.client
	ms.mu.Unlock()
	if c == nil {
		return
	}
	stderr, ok := client.GetStderr(c)
	if !ok || stderr == nil {
		return
	}
	buf := make([]byte, 4096)
	for {
		select {
		case <-ms.stopCh:
			return
		default:
			n, err := stderr.Read(buf)
			if n > 0 {
				s.log(ms.ID, "stderr", string(buf[:n]))
			}
			if err != nil {
				return
			}
		}
	}
}

// handleCrash attempts to restart a crashed server with exponential backoff.
// After 5 consecutive failures, the server is marked as StatusError.
// If the server is explicitly stopped during recovery, the restart is abandoned.
func (s *Supervisor) handleCrash(ms *ManagedServer) {
	ms.mu.Lock()
	ms.errorCount++
	count := ms.errorCount
	if count >= 5 {
		ms.status = StatusError
		ms.mu.Unlock()
		s.onStatus(ms.ID, StatusError)
		s.log(ms.ID, "lifecycle", "max retries exceeded, marking as error")
		return
	}
	// Capture stopCh under lock so we can listen for an explicit stop during backoff.
	stopCh := ms.stopCh
	ms.mu.Unlock()

	backoff := time.Duration(1<<uint(count-1)) * time.Second
	if backoff > 60*time.Second {
		backoff = 60 * time.Second
	}

	s.log(ms.ID, "lifecycle", fmt.Sprintf("restarting in %v (attempt %d/5)", backoff, count))

	select {
	case <-time.After(backoff):
	case <-s.ctx.Done():
		return
	case <-stopCh:
		s.log(ms.ID, "lifecycle", "restart cancelled — server was stopped")
		return
	}

	// Double-check the server wasn't stopped between the select firing and now.
	ms.mu.Lock()
	if ms.status == StatusStopped {
		ms.mu.Unlock()
		s.log(ms.ID, "lifecycle", "restart cancelled — server was stopped")
		return
	}
	ms.mu.Unlock()

	if err := s.Start(ms.ID); err != nil {
		s.log(ms.ID, "lifecycle", "restart failed: "+err.Error())
		go s.handleCrash(ms)
	}
}
