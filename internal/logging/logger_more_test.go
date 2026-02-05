package logging

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in     string
		want   Level
		wantOK bool
	}{
		{"", LevelInfo, true},
		{"debug", LevelDebug, true},
		{"INFO", LevelInfo, true},
		{"warn", LevelWarn, true},
		{"warning", LevelWarn, true},
		{"error", LevelError, true},
		{"nope", LevelInfo, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got, ok := ParseLevel(tc.in)
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("ParseLevel(%q) = (%v, %v), want (%v, %v)", tc.in, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestParseFormat(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in     string
		want   Format
		wantOK bool
	}{
		{"", FormatText, true},
		{"text", FormatText, true},
		{"json", FormatJSON, true},
		{"nope", FormatText, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got, ok := ParseFormat(tc.in)
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("ParseFormat(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestLogger_TextOutput_SortsAndQuotes(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	l := New(Options{
		Out:    &buf,
		Format: FormatText,
		Level:  LevelInfo,
		Now:    func() time.Time { return time.Unix(0, 0).UTC() },
	})

	l.Info("hello", F("b", "two"), F("a", 1), F("", "ignored"))

	got := buf.String()
	want := "1970-01-01T00:00:00Z level=info msg=\"hello\" a=1 b=\"two\"\n"
	if got != want {
		t.Fatalf("got = %q, want %q", got, want)
	}
}

func TestFormatTextValue(t *testing.T) {
	t.Parallel()

	if got := formatTextValue(errors.New("nope")); got != "\"nope\"" {
		t.Fatalf("error = %q, want %q", got, "\"nope\"")
	}
	if got := formatTextValue("a b"); got != "\"a b\"" {
		t.Fatalf("whitespace = %q, want %q", got, "\"a b\"")
	}

	ts := time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)
	if got := formatTextValue(ts); !strings.Contains(got, "2026-02-05T00:00:00Z") {
		t.Fatalf("time = %q, want ts substring", got)
	}
}

func TestJSONValue(t *testing.T) {
	t.Parallel()

	if got := jsonValue(nil); got != nil {
		t.Fatalf("nil = %#v, want nil", got)
	}
	if got := jsonValue([]byte("hi")); got != "hi" {
		t.Fatalf("bytes = %#v, want %q", got, "hi")
	}
	if got := jsonValue(errors.New("nope")); got != "nope" {
		t.Fatalf("error = %#v, want %q", got, "nope")
	}

	ts := time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)
	if got := jsonValue(ts); got != ts.Format(time.RFC3339Nano) {
		t.Fatalf("time = %#v, want %q", got, ts.Format(time.RFC3339Nano))
	}
}
