package services

import (
	"fmt"
	"time"

	"mcp-overwatch/internal/stats"
)

// StatsService exposes server statistics to the frontend.
type StatsService struct {
	store *stats.Store
}

// NewStatsService creates a StatsService backed by the given store.
func NewStatsService(store *stats.Store) *StatsService {
	return &StatsService{store: store}
}

// windowSince converts a window key ("24h", "7d", "30d", "all") into a UTC
// cutoff time. "all" returns the unix epoch so every event is included.
func windowSince(window string) time.Time {
	switch window {
	case "24h":
		return time.Now().UTC().Add(-24 * time.Hour)
	case "7d":
		return time.Now().UTC().Add(-7 * 24 * time.Hour)
	case "30d":
		return time.Now().UTC().Add(-30 * 24 * time.Hour)
	case "all":
		return time.Unix(0, 0).UTC()
	default:
		return time.Now().UTC().Add(-24 * time.Hour)
	}
}

// GetSummary returns aggregate totals across all servers for the given window.
func (s *StatsService) GetSummary(window string) (stats.Summary, error) {
	return s.store.SummarySince(windowSince(window))
}

// GetServerStats returns per-server aggregates for the given window.
func (s *StatsService) GetServerStats(window string) ([]stats.ServerStat, error) {
	return s.store.ServerStatsSince(windowSince(window))
}

// GetToolStats returns per-tool aggregates for a single server in the given window.
func (s *StatsService) GetToolStats(serverID, window string) ([]stats.ToolStat, error) {
	if serverID == "" {
		return nil, fmt.Errorf("serverID is required")
	}
	return s.store.ToolStatsSince(serverID, windowSince(window))
}

// GetActivityBuckets returns time-bucketed call/error counts for a server (or
// all servers if serverID is empty) for the given window. Bucket granularity
// is chosen by the store to keep the series bounded.
func (s *StatsService) GetActivityBuckets(serverID, window string) ([]stats.Bucket, error) {
	return s.store.ActivityBuckets(serverID, windowSince(window))
}
