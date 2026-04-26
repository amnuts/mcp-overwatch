package stats

import (
	"database/sql"
	"strings"
	"time"
)

// Event represents a single server event to be stored.
type Event struct {
	ServerID        string
	EventType       string
	ToolName        string
	LatencyMs       int64
	PayloadBytesIn  int64
	PayloadBytesOut int64
	ErrorMsg        string
	Timestamp       time.Time
}

// Store provides persistence for server events.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store backed by the given database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// InsertBatch inserts a batch of events in a single transaction.
func (s *Store) InsertBatch(events []Event) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO server_events
		(server_id, event_type, tool_name, latency_ms, payload_bytes_in, payload_bytes_out, error_message, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range events {
		ts := e.Timestamp
		if ts.IsZero() {
			ts = time.Now()
		}
		_, err := stmt.Exec(e.ServerID, e.EventType, e.ToolName, e.LatencyMs, e.PayloadBytesIn, e.PayloadBytesOut, e.ErrorMsg, ts)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// CleanOld removes events older than the given duration.
func (s *Store) CleanOld(maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	_, err := s.db.Exec("DELETE FROM server_events WHERE timestamp < ?", cutoff)
	return err
}

// Summary aggregates totals across all servers since the given time.
type Summary struct {
	TotalCalls    int64   `json:"total_calls"`
	TotalErrors   int64   `json:"total_errors"`
	ErrorRate     float64 `json:"error_rate"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	UniqueServers int64   `json:"unique_servers"`
	UniqueTools   int64   `json:"unique_tools"`
}

// ServerStat is a per-server aggregate over a time window.
type ServerStat struct {
	ServerID     string    `json:"server_id"`
	Calls        int64     `json:"calls"`
	Errors       int64     `json:"errors"`
	ErrorRate    float64   `json:"error_rate"`
	AvgLatencyMs float64   `json:"avg_latency_ms"`
	P95LatencyMs int64     `json:"p95_latency_ms"`
	MaxLatencyMs int64     `json:"max_latency_ms"`
	LastActivity time.Time `json:"last_activity"`
}

// ToolStat is a per-tool aggregate within a server.
type ToolStat struct {
	ToolName     string    `json:"tool_name"`
	Calls        int64     `json:"calls"`
	Errors       int64     `json:"errors"`
	AvgLatencyMs float64   `json:"avg_latency_ms"`
	MaxLatencyMs int64     `json:"max_latency_ms"`
	LastCall     time.Time `json:"last_call"`
}

// Bucket holds the call/error counts for a single time bucket. BucketSeconds
// is the bucket width — chosen dynamically from the queried window so the
// returned series stays bounded (see ActivityBuckets).
type Bucket struct {
	BucketStart   time.Time `json:"bucket_start"`
	BucketSeconds int64     `json:"bucket_seconds"`
	Calls         int64     `json:"calls"`
	Errors        int64     `json:"errors"`
}

// SummarySince returns aggregate totals for events since `since`.
func (s *Store) SummarySince(since time.Time) (Summary, error) {
	var sum Summary
	row := s.db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN event_type = 'tool_error' THEN 1 ELSE 0 END), 0),
			COALESCE(AVG(latency_ms), 0),
			COUNT(DISTINCT server_id),
			COUNT(DISTINCT tool_name)
		FROM server_events
		WHERE timestamp >= ?`, since)
	if err := row.Scan(&sum.TotalCalls, &sum.TotalErrors, &sum.AvgLatencyMs, &sum.UniqueServers, &sum.UniqueTools); err != nil {
		return sum, err
	}
	if sum.TotalCalls > 0 {
		sum.ErrorRate = float64(sum.TotalErrors) / float64(sum.TotalCalls)
	}
	return sum, nil
}

// ServerStatsSince returns per-server aggregates since `since`, ordered by call count desc.
func (s *Store) ServerStatsSince(since time.Time) ([]ServerStat, error) {
	rows, err := s.db.Query(`
		SELECT
			server_id,
			COUNT(*) AS calls,
			COALESCE(SUM(CASE WHEN event_type = 'tool_error' THEN 1 ELSE 0 END), 0) AS errors,
			COALESCE(AVG(latency_ms), 0) AS avg_latency,
			COALESCE(MAX(latency_ms), 0) AS max_latency,
			MAX(timestamp) AS last_activity
		FROM server_events
		WHERE timestamp >= ?
		GROUP BY server_id
		ORDER BY calls DESC`, since)
	if err != nil {
		return nil, err
	}

	var out []ServerStat
	for rows.Next() {
		var st ServerStat
		var lastActivity sql.NullString
		if err := rows.Scan(&st.ServerID, &st.Calls, &st.Errors, &st.AvgLatencyMs, &st.MaxLatencyMs, &lastActivity); err != nil {
			rows.Close()
			return nil, err
		}
		if lastActivity.Valid {
			st.LastActivity = parseDBTime(lastActivity.String)
		}
		if st.Calls > 0 {
			st.ErrorRate = float64(st.Errors) / float64(st.Calls)
		}
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	// Compute percentiles in a second pass so we don't hold the outer
	// connection while issuing per-server queries (SQLite serialises).
	for i := range out {
		out[i].P95LatencyMs, _ = s.percentileLatency(out[i].ServerID, since, 0.95)
	}
	return out, nil
}

// ToolStatsSince returns per-tool aggregates for a single server since `since`.
func (s *Store) ToolStatsSince(serverID string, since time.Time) ([]ToolStat, error) {
	rows, err := s.db.Query(`
		SELECT
			COALESCE(tool_name, '') AS tool,
			COUNT(*) AS calls,
			COALESCE(SUM(CASE WHEN event_type = 'tool_error' THEN 1 ELSE 0 END), 0) AS errors,
			COALESCE(AVG(latency_ms), 0) AS avg_latency,
			COALESCE(MAX(latency_ms), 0) AS max_latency,
			MAX(timestamp) AS last_call
		FROM server_events
		WHERE server_id = ? AND timestamp >= ?
		GROUP BY tool_name
		ORDER BY calls DESC`, serverID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ToolStat
	for rows.Next() {
		var t ToolStat
		var lastCall sql.NullString
		if err := rows.Scan(&t.ToolName, &t.Calls, &t.Errors, &t.AvgLatencyMs, &t.MaxLatencyMs, &lastCall); err != nil {
			return nil, err
		}
		if lastCall.Valid {
			t.LastCall = parseDBTime(lastCall.String)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// parseDBTime parses a timestamp string returned by SQLite aggregate
// functions (MAX/MIN over a TIMESTAMP column), which come back as strings
// rather than time.Time via database/sql.
//
// modernc.org/sqlite emits MAX(timestamp) using Go's time.Time.String()
// format. When the original time.Time carried a monotonic-clock reading
// (anything from time.Now()), String() appends a " m=±<value>" suffix
// that breaks time.Parse — so we strip it before parsing.
func parseDBTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if i := strings.Index(s, " m="); i >= 0 {
		s = s[:i]
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// maxBuckets caps the number of buckets returned to ActivityBuckets, regardless
// of the queried window. Beyond this the chart stops being readable and the
// payload starts hurting IPC + render time.
const maxBuckets = 200

// bucketSizes lists the granularities ActivityBuckets will choose from, in
// ascending order. The smallest size whose total count fits under maxBuckets wins.
var bucketSizes = []time.Duration{
	time.Hour,
	6 * time.Hour,
	24 * time.Hour,
	7 * 24 * time.Hour,
	30 * 24 * time.Hour,
}

// chooseBucketSize picks a granularity from bucketSizes such that the number
// of buckets covering `span` does not exceed maxBuckets.
func chooseBucketSize(span time.Duration) time.Duration {
	for _, size := range bucketSizes {
		if int64(span/size) <= maxBuckets {
			return size
		}
	}
	return bucketSizes[len(bucketSizes)-1]
}

// ActivityBuckets returns time-bucketed call/error counts for a server (or all
// servers if serverID is empty), starting no earlier than `since` and no
// earlier than the first recorded event. Bucket granularity is chosen
// dynamically so the result is always at most ~maxBuckets entries — large
// windows (e.g. "all") fall back to daily or weekly granularity rather than
// returning hundreds of thousands of empty hourly slots.
//
// Returned in chronological order with empty buckets included.
//
// Bucketing is done in Go because modernc.org/sqlite stores time.Time values
// in a format that SQLite's strftime does not parse, so server-side bucketing
// returns NULL for every row.
func (s *Store) ActivityBuckets(serverID string, since time.Time) ([]Bucket, error) {
	// Clamp `since` to the earliest event so "all" doesn't generate buckets
	// back to the unix epoch when there are only days of real data.
	earliest, err := s.earliestTimestamp(serverID, since)
	if err != nil {
		return nil, err
	}
	if earliest.IsZero() {
		return []Bucket{}, nil
	}
	if earliest.After(since) {
		since = earliest
	}

	end := time.Now().UTC()
	size := chooseBucketSize(end.Sub(since))
	start := truncate(since.UTC(), size)
	endTrunc := truncate(end, size)

	var (
		rows *sql.Rows
	)
	if serverID == "" {
		rows, err = s.db.Query(`
			SELECT timestamp, event_type
			FROM server_events
			WHERE timestamp >= ?
			ORDER BY timestamp`, start)
	} else {
		rows, err = s.db.Query(`
			SELECT timestamp, event_type
			FROM server_events
			WHERE server_id = ? AND timestamp >= ?
			ORDER BY timestamp`, serverID, start)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	got := make(map[time.Time]*Bucket)
	for rows.Next() {
		var ts time.Time
		var eventType string
		if err := rows.Scan(&ts, &eventType); err != nil {
			return nil, err
		}
		key := truncate(ts.UTC(), size)
		b, ok := got[key]
		if !ok {
			b = &Bucket{BucketStart: key, BucketSeconds: int64(size.Seconds())}
			got[key] = b
		}
		b.Calls++
		if eventType == "tool_error" {
			b.Errors++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	bucketSecs := int64(size.Seconds())
	buckets := make([]Bucket, 0, int(endTrunc.Sub(start)/size)+1)
	for t := start; !t.After(endTrunc); t = t.Add(size) {
		if b, ok := got[t]; ok {
			buckets = append(buckets, *b)
		} else {
			buckets = append(buckets, Bucket{BucketStart: t, BucketSeconds: bucketSecs})
		}
	}
	return buckets, nil
}

// earliestTimestamp returns the timestamp of the earliest event matching the
// (optional) serverID filter at-or-after `since`. Zero time means no events.
func (s *Store) earliestTimestamp(serverID string, since time.Time) (time.Time, error) {
	var (
		raw sql.NullString
		err error
	)
	if serverID == "" {
		err = s.db.QueryRow(`SELECT MIN(timestamp) FROM server_events WHERE timestamp >= ?`, since).Scan(&raw)
	} else {
		err = s.db.QueryRow(`SELECT MIN(timestamp) FROM server_events WHERE server_id = ? AND timestamp >= ?`, serverID, since).Scan(&raw)
	}
	if err != nil {
		return time.Time{}, err
	}
	if !raw.Valid {
		return time.Time{}, nil
	}
	return parseDBTime(raw.String), nil
}

// truncate rounds t down to the nearest multiple of size in UTC. Mirrors
// time.Time.Truncate but on a UTC reference so day/week boundaries are stable.
func truncate(t time.Time, size time.Duration) time.Time {
	if size <= 0 {
		return t
	}
	if size >= 24*time.Hour {
		// Day-aligned: drop time-of-day, then snap to a multiple of `size` from
		// the unix epoch (which is itself midnight UTC).
		dayStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		offset := dayStart.Sub(time.Unix(0, 0).UTC()) % size
		return dayStart.Add(-offset)
	}
	return t.Truncate(size)
}

// percentileLatency returns the requested percentile of latency_ms for a server
// since `since`. Uses a nearest-rank approximation over the row set.
func (s *Store) percentileLatency(serverID string, since time.Time, pct float64) (int64, error) {
	rows, err := s.db.Query(`
		SELECT latency_ms FROM server_events
		WHERE server_id = ? AND timestamp >= ? AND latency_ms IS NOT NULL
		ORDER BY latency_ms ASC`, serverID, since)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var values []int64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return 0, err
		}
		values = append(values, v)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(values) == 0 {
		return 0, nil
	}
	idx := int(float64(len(values)-1) * pct)
	return values[idx], nil
}
