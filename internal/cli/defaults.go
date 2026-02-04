package cli

import (
	"strings"

	"eosrift.com/eosrift/internal/config"
)

func resolveServerAddrDefault(cfg config.File) string {
	serverDefault := getenv("EOSRIFT_SERVER_ADDR", "")
	if serverDefault == "" {
		serverDefault = getenv("EOSRIFT_CONTROL_URL", "")
	}
	if serverDefault == "" {
		serverDefault = cfg.ServerAddr
	}
	if serverDefault == "" {
		serverDefault = "https://eosrift.com"
	}
	return serverDefault
}

func resolveAuthtokenDefault(cfg config.File) string {
	authtokenDefault := getenv("EOSRIFT_AUTHTOKEN", "")
	if authtokenDefault == "" {
		authtokenDefault = getenv("EOSRIFT_AUTH_TOKEN", "")
	}
	if authtokenDefault == "" {
		authtokenDefault = cfg.Authtoken
	}
	return authtokenDefault
}

func resolveInspectEnabledDefault(cfg config.File) bool {
	inspectDefault := true
	if cfg.Inspect != nil {
		inspectDefault = *cfg.Inspect
	}
	return inspectDefault
}

func resolveInspectAddrDefault(cfg config.File) string {
	inspectAddrDefault := getenv("EOSRIFT_INSPECT_ADDR", "")
	if inspectAddrDefault == "" {
		inspectAddrDefault = cfg.InspectAddr
	}
	if inspectAddrDefault == "" {
		inspectAddrDefault = "127.0.0.1:4040"
	}
	return inspectAddrDefault
}

func resolveHostHeaderDefault(cfg config.File) string {
	hostHeaderDefault := cfg.HostHeader
	if strings.TrimSpace(hostHeaderDefault) == "" {
		hostHeaderDefault = "preserve"
	}
	return hostHeaderDefault
}
