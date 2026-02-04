package mux

import (
	"io"
	"time"

	"github.com/hashicorp/yamux"
)

func QuietYamuxConfig() *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.EnableKeepAlive = true
	cfg.KeepAliveInterval = 25 * time.Second
	cfg.LogOutput = io.Discard
	cfg.Logger = nil
	return cfg
}
