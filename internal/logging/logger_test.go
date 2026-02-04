package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLogger_LevelFiltering(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	l := New(Options{
		Out:    &buf,
		Format: FormatText,
		Level:  LevelInfo,
		Now:    func() time.Time { return time.Unix(0, 0).UTC() },
	})

	l.Debug("debug msg")
	l.Info("info msg")

	out := buf.String()
	if strings.Contains(out, "debug msg") {
		t.Fatalf("expected debug to be filtered, got %q", out)
	}
	if !strings.Contains(out, "info msg") {
		t.Fatalf("expected info to be logged, got %q", out)
	}
}

func TestLogger_JSONOutput(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 4, 5, 0, 0, 0, time.UTC)

	var buf bytes.Buffer
	l := New(Options{
		Out:    &buf,
		Format: FormatJSON,
		Level:  LevelInfo,
		Now:    func() time.Time { return now },
	})

	l.Info("hello", F("a", 1), F("b", "two"))

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatalf("expected output")
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("unmarshal: %v (line=%q)", err, line)
	}

	if got := m["level"]; got != "info" {
		t.Fatalf("level = %#v, want %q", got, "info")
	}
	if got := m["msg"]; got != "hello" {
		t.Fatalf("msg = %#v, want %q", got, "hello")
	}
	if got := m["ts"]; got != now.Format(time.RFC3339Nano) {
		t.Fatalf("ts = %#v, want %q", got, now.Format(time.RFC3339Nano))
	}
	if got := m["a"]; got != float64(1) {
		t.Fatalf("a = %#v, want %v", got, 1)
	}
	if got := m["b"]; got != "two" {
		t.Fatalf("b = %#v, want %q", got, "two")
	}
}

func TestLogger_WithBaseFields(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 4, 5, 0, 0, 0, time.UTC)

	var buf bytes.Buffer
	l := New(Options{
		Out:    &buf,
		Format: FormatJSON,
		Level:  LevelDebug,
		Now:    func() time.Time { return now },
	})

	l2 := l.With(F("req_id", "abc"))
	l2.Error("boom", F("err", errors.New("nope")))

	line := strings.TrimSpace(buf.String())
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := m["req_id"]; got != "abc" {
		t.Fatalf("req_id = %#v, want %q", got, "abc")
	}
	if got := m["err"]; got != "nope" {
		t.Fatalf("err = %#v, want %q", got, "nope")
	}
}

