package mux

import (
	"io"
	"testing"
)

func TestQuietYamuxConfig(t *testing.T) {
	t.Parallel()

	cfg := QuietYamuxConfig()
	if cfg == nil {
		t.Fatal("cfg is nil")
	}

	if cfg.LogOutput != io.Discard {
		t.Fatalf("LogOutput = %v, want io.Discard", cfg.LogOutput)
	}

	if cfg.Logger != nil {
		t.Fatalf("Logger = %v, want nil", cfg.Logger)
	}
}

