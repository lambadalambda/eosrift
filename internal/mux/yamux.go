package mux

import (
	"io"

	"github.com/hashicorp/yamux"
)

func QuietYamuxConfig() *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.LogOutput = io.Discard
	cfg.Logger = nil
	return cfg
}

