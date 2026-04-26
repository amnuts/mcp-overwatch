package stats

import (
	"path/filepath"
	"testing"
	"time"

	"mcp-overwatch/internal/database"
)

func TestRecordAndFlush(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "stats.db")

	db, err := database.OpenStats(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := NewStore(db)
	collector := NewCollector(store, 1*time.Hour) // long interval so we flush manually

	collector.Record(Event{
		ServerID:  "s1",
		EventType: "tool_call",
		ToolName:  "web_search",
		LatencyMs: 42,
	})
	collector.Record(Event{
		ServerID:  "s1",
		EventType: "tool_error",
		ToolName:  "web_search",
		LatencyMs: 100,
		ErrorMsg:  "timeout",
	})

	if err := collector.Flush(); err != nil {
		t.Fatal(err)
	}

	// Verify events in DB
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM server_events").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected 2 events, got %d", count)
	}

	// Verify specific fields
	var toolName, errMsg string
	err = db.QueryRow("SELECT tool_name, error_message FROM server_events WHERE event_type = 'tool_error'").Scan(&toolName, &errMsg)
	if err != nil {
		t.Fatal(err)
	}
	if toolName != "web_search" || errMsg != "timeout" {
		t.Errorf("got tool=%s err=%s", toolName, errMsg)
	}

	collector.Stop()
}

func TestFlushEmptyBuffer(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "stats.db")

	db, err := database.OpenStats(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := NewStore(db)
	collector := NewCollector(store, 1*time.Hour)

	// Flushing empty buffer should not error
	if err := collector.Flush(); err != nil {
		t.Fatal(err)
	}

	collector.Stop()
}

func TestAggregationQueries(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "stats.db")

	db, err := database.OpenStats(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := NewStore(db)
	// Use time.Now() (with its monotonic clock reading) rather than .UTC()
	// to exercise the same code path as production — modernc serialises the
	// monotonic suffix in MAX(timestamp), which parseDBTime must strip.
	now := time.Now()

	events := []Event{
		{ServerID: "s1", EventType: "tool_call", ToolName: "search", LatencyMs: 100, Timestamp: now.Add(-30 * time.Minute)},
		{ServerID: "s1", EventType: "tool_call", ToolName: "search", LatencyMs: 200, Timestamp: now.Add(-20 * time.Minute)},
		{ServerID: "s1", EventType: "tool_error", ToolName: "search", LatencyMs: 50, ErrorMsg: "boom", Timestamp: now.Add(-10 * time.Minute)},
		{ServerID: "s2", EventType: "tool_call", ToolName: "fetch", LatencyMs: 300, Timestamp: now.Add(-5 * time.Minute)},
		// Outside the 24h window — should be excluded
		{ServerID: "s1", EventType: "tool_call", ToolName: "search", LatencyMs: 999, Timestamp: now.Add(-48 * time.Hour)},
	}
	if err := store.InsertBatch(events); err != nil {
		t.Fatal(err)
	}

	since := now.Add(-24 * time.Hour)

	sum, err := store.SummarySince(since)
	if err != nil {
		t.Fatal(err)
	}
	if sum.TotalCalls != 4 {
		t.Errorf("SummarySince total_calls: got %d want 4", sum.TotalCalls)
	}
	if sum.TotalErrors != 1 {
		t.Errorf("SummarySince total_errors: got %d want 1", sum.TotalErrors)
	}
	if sum.UniqueServers != 2 {
		t.Errorf("SummarySince unique_servers: got %d want 2", sum.UniqueServers)
	}
	if sum.UniqueTools != 2 {
		t.Errorf("SummarySince unique_tools: got %d want 2", sum.UniqueTools)
	}
	wantAvg := float64(100+200+50+300) / 4.0
	if sum.AvgLatencyMs != wantAvg {
		t.Errorf("SummarySince avg_latency: got %f want %f", sum.AvgLatencyMs, wantAvg)
	}

	servers, err := store.ServerStatsSince(since)
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 2 {
		t.Fatalf("ServerStatsSince: got %d rows want 2", len(servers))
	}
	// Ordered by calls DESC — s1 has 3, s2 has 1
	if servers[0].ServerID != "s1" || servers[0].Calls != 3 || servers[0].Errors != 1 {
		t.Errorf("ServerStatsSince s1: got %+v", servers[0])
	}
	if servers[1].ServerID != "s2" || servers[1].Calls != 1 {
		t.Errorf("ServerStatsSince s2: got %+v", servers[1])
	}
	// LastActivity must round-trip — within a minute of the latest event.
	if delta := now.Sub(servers[0].LastActivity); delta < 9*time.Minute || delta > 11*time.Minute {
		t.Errorf("ServerStatsSince s1 LastActivity: got %v (delta from now: %v), want ~10 min ago", servers[0].LastActivity, delta)
	}
	if servers[1].LastActivity.IsZero() {
		t.Errorf("ServerStatsSince s2 LastActivity is zero")
	}

	tools, err := store.ToolStatsSince("s1", since)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].ToolName != "search" || tools[0].Calls != 3 || tools[0].Errors != 1 {
		t.Errorf("ToolStatsSince s1: got %+v", tools)
	}
	if tools[0].LastCall.IsZero() {
		t.Errorf("ToolStatsSince LastCall is zero")
	}

	buckets, err := store.ActivityBuckets("s1", since)
	if err != nil {
		t.Fatal(err)
	}
	// since is 24h ago but earliest event is 30 min ago, so the series should
	// be clamped to a small range — at most a couple of hourly buckets.
	if len(buckets) < 1 || len(buckets) > 3 {
		t.Errorf("ActivityBuckets count: got %d want 1-3 (clamped to earliest event)", len(buckets))
	}
	if len(buckets) > 0 && buckets[0].BucketSeconds != int64(time.Hour.Seconds()) {
		t.Errorf("ActivityBuckets size: got %ds want 3600 (hourly)", buckets[0].BucketSeconds)
	}
	var totalCalls int64
	for _, b := range buckets {
		totalCalls += b.Calls
	}
	if totalCalls != 3 {
		t.Errorf("ActivityBuckets total calls: got %d want 3", totalCalls)
	}
}

func TestStoreCleanOld(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "stats.db")

	db, err := database.OpenStats(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := NewStore(db)

	// Insert an old event directly
	oldTime := time.Now().Add(-48 * time.Hour)
	err = store.InsertBatch([]Event{
		{ServerID: "s1", EventType: "tool_call", ToolName: "old_tool", Timestamp: oldTime},
		{ServerID: "s1", EventType: "tool_call", ToolName: "new_tool", Timestamp: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = store.CleanOld(24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM server_events").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 event after cleanup, got %d", count)
	}
}
