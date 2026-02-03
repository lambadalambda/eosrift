package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"eosrift.com/eosrift/internal/client"
	"eosrift.com/eosrift/internal/config"
)

func runTCP(ctx context.Context, args []string, configPath string, stdout, stderr io.Writer) int {
	cfg, _, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

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

	authtokenDefault := getenv("EOSRIFT_AUTHTOKEN", "")
	if authtokenDefault == "" {
		authtokenDefault = getenv("EOSRIFT_AUTH_TOKEN", "")
	}
	if authtokenDefault == "" {
		authtokenDefault = cfg.Authtoken
	}

	fs := flag.NewFlagSet("tcp", flag.ContinueOnError)
	fs.SetOutput(stderr)

	serverAddr := fs.String("server", serverDefault, "Server address (https://host, http://host:port, or ws(s)://host/control)")
	authtoken := fs.String("authtoken", authtokenDefault, "Auth token")

	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: eosrift tcp [flags] <local-port|local-addr>")
		fs.PrintDefaults()
	}

	if err := parseInterspersedFlags(fs, args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}

	localAddr := fs.Arg(0)
	if !strings.Contains(localAddr, ":") {
		localAddr = "127.0.0.1:" + localAddr
	}

	controlURL, err := config.ControlURLFromServerAddr(*serverAddr)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	tunnel, err := client.StartTCPTunnelWithOptions(ctx, controlURL, localAddr, client.TCPTunnelOptions{
		Authtoken: *authtoken,
	})
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	defer tunnel.Close()

	host := controlHost(controlURL)
	printSession(stdout, sessionOutput{
		Version:        version,
		Status:         "online",
		ForwardingFrom: fmt.Sprintf("tcp://%s:%d", host, tunnel.RemotePort),
		ForwardingTo:   displayHostPort(localAddr),
	})

	if err := tunnel.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		if ctx.Err() != nil {
			return 0
		}
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	return 0
}
