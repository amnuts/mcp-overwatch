package logging

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRingBufferAddRetrieve(t *testing.T) {
	l, err := NewLogger(3, "")
	if err != nil {
		t.Fatal(err)
	}

	l.Add(Entry{ServerID: "s1", Direction: "in", Summary: "msg1"})
	l.Add(Entry{ServerID: "s1", Direction: "out", Summary: "msg2"})

	entries := l.Recent(10) // ask for more than exist
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Summary != "msg1" || entries[1].Summary != "msg2" {
		t.Error("entries not in expected order")
	}
}

func TestRingBufferOverflow(t *testing.T) {
	l, err := NewLogger(2, "")
	if err != nil {
		t.Fatal(err)
	}

	l.Add(Entry{Summary: "a"})
	l.Add(Entry{Summary: "b"})
	l.Add(Entry{Summary: "c"}) // overwrites "a"

	entries := l.Recent(5)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Summary != "b" || entries[1].Summary != "c" {
		t.Errorf("expected [b, c], got [%s, %s]", entries[0].Summary, entries[1].Summary)
	}
}

func TestJSONRPCParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`{"jsonrpc":"2.0","method":"tools/list","id":1}`, "request: tools/list"},
		{`{"jsonrpc":"2.0","method":"notifications/cancelled"}`, "notification: notifications/cancelled"},
		{`{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`, "result response"},
		{`{"jsonrpc":"2.0","id":1,"error":{"code":-1,"message":"fail"}}`, "error response"},
		{`not json`, "invalid JSON"},
	}
	for _, tt := range tests {
		got := ParseJSONRPC([]byte(tt.input))
		if got != tt.expected {
			t.Errorf("ParseJSONRPC(%s) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFileWriting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	l, err := NewLogger(10, path)
	if err != nil {
		t.Fatal(err)
	}

	l.Add(Entry{ServerID: "s1", Direction: "in", Summary: "hello"})
	l.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("expected log file to have content")
	}
}

func TestOnEntryCallback(t *testing.T) {
	l, err := NewLogger(10, "")
	if err != nil {
		t.Fatal(err)
	}

	var got Entry
	l.OnEntry(func(e Entry) {
		got = e
	})

	l.Add(Entry{Summary: "callback test"})
	if got.Summary != "callback test" {
		t.Errorf("callback not invoked, got summary=%q", got.Summary)
	}
}

func TestCleanOldLogs(t *testing.T) {
	dir := t.TempDir()

	// Create an "old" file
	oldFile := filepath.Join(dir, "old.log")
	os.WriteFile(oldFile, []byte("old"), 0644)
	// Backdate it
	oldTime := time.Now().Add(-48 * time.Hour)
	os.Chtimes(oldFile, oldTime, oldTime)

	// Create a "new" file
	newFile := filepath.Join(dir, "new.log")
	os.WriteFile(newFile, []byte("new"), 0644)

	err := CleanOldLogs(dir, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("old file should have been removed")
	}
	if _, err := os.Stat(newFile); err != nil {
		t.Error("new file should still exist")
	}
}
