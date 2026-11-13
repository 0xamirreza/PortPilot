// Package logging writes structured text logs to stderr, the XDG cache, and in-memory diagnostics.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

// Entry represents one structured log event according to standard logging formats.
type Entry struct {
	Timestamp time.Time
	Level     slog.Level
	Message   string
	Attrs     map[string]string
	Raw       string
}

// Store maintains an in-memory ring buffer of recent logs for live diagnostics.
type Store struct {
	mu      sync.RWMutex
	entries []Entry
	max     int
	logPath string
}

// DefaultStore is the process-wide diagnostics log store.
var DefaultStore = &Store{max: 1000}

// SetLogPath configures the target log file location.
func (s *Store) SetLogPath(p string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logPath = p
}

// LogPath returns the configured log file path.
func (s *Store) LogPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.logPath
}

// Add appends a new entry to the store.
func (s *Store) Add(e Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, e)
	if s.max > 0 && len(s.entries) > s.max {
		s.entries = s.entries[len(s.entries)-s.max:]
	}
}

// All returns a snapshot of all stored entries, newest first.
func (s *Store) All() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, len(s.entries))
	for i, e := range s.entries {
		out[len(s.entries)-1-i] = e
	}
	return out
}

// Clear empties in-memory entries and truncates the log file.
func (s *Store) Clear() error {
	s.mu.Lock()
	s.entries = nil
	path := s.logPath
	s.mu.Unlock()

	if path != "" {
		return os.WriteFile(path, []byte{}, 0o600)
	}
	return nil
}

// Rotate purges entries older than days both on disk and in memory.
func (s *Store) Rotate(days int) error {
	if days <= 0 {
		return nil
	}
	cutoff := time.Now().AddDate(0, 0, -days)

	s.mu.Lock()
	path := s.logPath
	var kept []Entry
	for _, e := range s.entries {
		if !e.Timestamp.Before(cutoff) {
			kept = append(kept, e)
		}
	}
	s.entries = kept
	s.mu.Unlock()

	if path != "" {
		return Rotate(path, days)
	}
	return nil
}

// Rotate purges lines older than days from the specified file on disk.
func Rotate(path string, days int) error {
	if days <= 0 {
		return nil
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := strings.Split(string(data), "\n")
	var keptLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		entry, ok := ParseLine(trimmed)
		if ok && entry.Timestamp.Before(cutoff) {
			continue
		}
		keptLines = append(keptLines, line)
	}
	newContent := ""
	if len(keptLines) > 0 {
		newContent = strings.Join(keptLines, "\n") + "\n"
	}
	tmpPath := path + ".rot.tmp"
	if err := os.WriteFile(tmpPath, []byte(newContent), 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// ParseLine parses a standard slog logfmt line into an Entry.
func ParseLine(line string) (Entry, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return Entry{}, false
	}
	e := Entry{
		Timestamp: time.Now(),
		Level:     slog.LevelInfo,
		Attrs:     make(map[string]string),
		Raw:       line,
	}

	tokens := tokenizeLogfmt(line)
	for _, tok := range tokens {
		parts := strings.SplitN(tok, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := parts[0]
		v := strings.Trim(parts[1], `"`)
		switch k {
		case "time":
			if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
				e.Timestamp = t
			} else if t, err := time.Parse(time.RFC3339, v); err == nil {
				e.Timestamp = t
			}
		case "level":
			switch strings.ToUpper(v) {
			case "DEBUG":
				e.Level = slog.LevelDebug
			case "INFO":
				e.Level = slog.LevelInfo
			case "WARN", "WARNING":
				e.Level = slog.LevelWarn
			case "ERROR":
				e.Level = slog.LevelError
			}
		case "msg":
			e.Message = v
		default:
			e.Attrs[k] = v
		}
	}
	return e, true
}

func tokenizeLogfmt(s string) []string {
	var tokens []string
	var cur strings.Builder
	inQuotes := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' {
			inQuotes = !inQuotes
			cur.WriteByte(ch)
		} else if ch == ' ' && !inQuotes {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		} else {
			cur.WriteByte(ch)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

type ringHandler struct {
	inner slog.Handler
	store *Store
}

func (h *ringHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *ringHandler) Handle(ctx context.Context, r slog.Record) error {
	attrs := make(map[string]string)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.String()
		return true
	})
	h.store.Add(Entry{
		Timestamp: r.Time,
		Level:     r.Level,
		Message:   r.Message,
		Attrs:     attrs,
	})
	return h.inner.Handle(ctx, r)
}

func (h *ringHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ringHandler{inner: h.inner.WithAttrs(attrs), store: h.store}
}

func (h *ringHandler) WithGroup(name string) slog.Handler {
	return &ringHandler{inner: h.inner.WithGroup(name), store: h.store}
}

// New builds a structured text logger and attaches it to DefaultStore.
func New(path string) (*slog.Logger, *os.File, error) {
	DefaultStore.SetLogPath(path)

	// Load existing logs from file into DefaultStore if present
	if data, err := os.ReadFile(path); err == nil {
		lines := strings.Split(string(data), "\n")
		// Limit to last 500 lines for initial load
		start := 0
		if len(lines) > 500 {
			start = len(lines) - 500
		}
		for _, line := range lines[start:] {
			if e, ok := ParseLine(line); ok {
				DefaultStore.Add(e)
			}
		}
	}

	var writers []io.Writer
	writers = append(writers, os.Stderr)
	var file *os.File
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err == nil {
		file = f
		_ = os.Chmod(path, 0o600)
		writers = append(writers, file)
	}
	level := slog.LevelDebug
	if env := os.Getenv("PORTPILOT_LOG"); env != "" {
		switch strings.ToLower(env) {
		case "info":
			level = slog.LevelInfo
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}
	textH := slog.NewTextHandler(io.MultiWriter(writers...), &slog.HandlerOptions{Level: level})
	composite := &ringHandler{inner: textH, store: DefaultStore}
	return slog.New(composite), file, err
}

// Entries returns recent entries from DefaultStore, newest first.
func Entries() []Entry {
	return DefaultStore.All()
}
