package gui

import (
	"log/slog"
	"testing"
	"time"

	"github.com/0xamirreza/portpilot/internal/app"
	"github.com/0xamirreza/portpilot/internal/logging"
)

func TestDiagCapturesAndFiltersLogs(t *testing.T) {
	logging.DefaultStore.Clear()

	logging.DefaultStore.Add(logging.Entry{
		Timestamp: time.Now(),
		Level:     slog.LevelInfo,
		Message:   "test info log",
		Attrs:     map[string]string{"foo": "bar"},
	})
	logging.DefaultStore.Add(logging.Entry{
		Timestamp: time.Now(),
		Level:     slog.LevelError,
		Message:   "test error log",
		Attrs:     map[string]string{"status": "failed"},
	})

	entries := logging.DefaultStore.All()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	ui := &UI{}
	view := app.View{
		LogPath: "/tmp/test.log",
	}
	_ = view

	ui.selectLogEntry(&entries[0])
	if ui.diagSelectedEntry == nil || ui.diagSelectedEntry.Message != "test error log" {
		t.Fatalf("expected selected entry message 'test error log', got %q", ui.diagSelectedEntry.Message)
	}
	if ui.diagJsonEditor.Text() == "" {
		t.Fatalf("expected non-empty JSON editor text")
	}
}

