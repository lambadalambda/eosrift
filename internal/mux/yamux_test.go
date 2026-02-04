package mux

import (
	"io"
	"testing"
	"time"
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

	if cfg.EnableKeepAlive != true {
		t.Fatalf("EnableKeepAlive = %v, want true", cfg.EnableKeepAlive)
	}

	if got, want := cfg.KeepAliveInterval, 25*time.Second; got != want {
		t.Fatalf("KeepAliveInterval = %s, want %s", got, want)
	}
}
