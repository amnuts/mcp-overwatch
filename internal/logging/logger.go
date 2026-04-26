package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry represents a single log entry.
type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	ServerID  string    `json:"server_id"`
	Direction string    `json:"direction"` // "in" or "out"
	Summary   string    `json:"summary"`
	Raw       string    `json:"raw,omitempty"`
}

// Logger is a ring-buffer logger with file output and callbacks.
type Logger struct {
	entries  []Entry
	head     int
	count    int
	capacity int
	mu       sync.RWMutex
	filePath string
	file     *os.File
	callback func(Entry)
}

// NewLogger creates a logger with a ring buffer of the given capacity.
// If filePath is non-empty, entries are also written to that file.
func NewLogger(capacity int, filePath string) (*Logger, error) {
	l := &Logger{
		entries:  make([]Entry, capacity),
		capacity: capacity,
		filePath: filePath,
	}
	if filePath != "" {
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create log dir: %w", err)
		}
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("open log file: %w", err)
		}
		l.file = f
	}
	return l, nil
}

// OnEntry sets a callback invoked for each new entry.
func (l *Logger) OnEntry(cb func(Entry)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.callback = cb
}

// Add adds an entry to the ring buffer.
func (l *Logger) Add(e Entry) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	l.mu.Lock()
	l.entries[l.head] = e
	l.head = (l.head + 1) % l.capacity
	if l.count < l.capacity {
		l.count++
	}
	cb := l.callback
	l.mu.Unlock()

	if l.file != nil {
		l.writeToFile(e)
	}

	if cb != nil {
		cb(e)
	}
}

// Recent returns the most recent n entries (oldest first).
func (l *Logger) Recent(n int) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if n > l.count {
		n = l.count
	}
	result := make([]Entry, n)
	start := (l.head - n + l.capacity) % l.capacity
	for i := 0; i < n; i++ {
		result[i] = l.entries[(start+i)%l.capacity]
	}
	return result
}

// Close closes the log file if open.
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) writeToFile(e Entry) {
	line := fmt.Sprintf("%s [%s] %s %s\n",
		e.Timestamp.Format(time.RFC3339),
		e.ServerID,
		e.Direction,
		e.Summary,
	)
	l.file.WriteString(line)
}
