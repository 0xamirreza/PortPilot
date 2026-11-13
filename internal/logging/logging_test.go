package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseLine(t *testing.T) {
	line := `time=2026-11-08T22:03:00.725+03:30 level=INFO msg=open config=/path/config.json db=/path/db.sqlite theme=dark`
	e, ok := ParseLine(line)
	if !ok {
		t.Fatal("expected ok parse")
	}
	if e.Level != slog.LevelInfo {
		t.Fatalf("expected INFO, got %v", e.Level)
	}
	if e.Message != "open" {
		t.Fatalf("expected open, got %s", e.Message)
	}
	if e.Attrs["config"] != "/path/config.json" {
		t.Fatalf("expected /path/config.json, got %s", e.Attrs["config"])
	}
}

func TestRotatePrunesOldLogs(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	now := time.Now()
	oldTime := now.AddDate(0, 0, -10).Format(time.RFC3339)
	newTime := now.AddDate(0, 0, -1).Format(time.RFC3339)

	content := "time=" + oldTime + " level=INFO msg=\"old message\"\n" +
		"time=" + newTime + " level=WARN msg=\"new message\"\n"

	if err := os.WriteFile(logPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	// Rotate with 7 days retention
	if err := Rotate(logPath, 7); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	str := string(data)
	if len(str) == 0 {
		t.Fatal("expected retained line, got empty")
	}
	if str == content {
		t.Fatal("expected old line to be pruned")
	}
	if e, ok := ParseLine(str); !ok || e.Message != "new message" {
		t.Fatalf("unexpected line: %s", str)
	}
}

