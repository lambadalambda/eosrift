package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"eosrift.com/eosrift/internal/config"
)

var version = "dev"

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	global := flag.NewFlagSet("eosrift", flag.ContinueOnError)
	global.SetOutput(stderr)

	defaultConfigPath := getenv("EOSRIFT_CONFIG", config.DefaultPath())
	configPath := global.String("config", defaultConfigPath, "Config file path")
	var help bool
	global.BoolVar(&help, "help", false, "Show help")
	global.BoolVar(&help, "h", false, "Show help")

	global.Usage = func() {
		usage(stderr)
	}

	if err := global.Parse(args); err != nil {
		return 2
	}

	if help {
		usage(stdout)
		return 0
	}

	rest := global.Args()
	if len(rest) == 0 {
		usage(stderr)
		return 2
	}

	switch rest[0] {
	case "help":
		usage(stdout)
		return 0
	case "version":
		fmt.Fprintf(stdout, "eosrift version %s\n", version)
		return 0
	case "config":
		return runConfig(rest[1:], *configPath, stdout, stderr)
	case "http":
		return runHTTP(ctx, rest[1:], *configPath, stdout, stderr)
	case "tcp":
		return runTCP(ctx, rest[1:], *configPath, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", rest[0])
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: eosrift [--config <path>] <command> [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  http      start an HTTP tunnel")
	fmt.Fprintln(w, "  tcp       start a TCP tunnel")
	fmt.Fprintln(w, "  config    manage client config")
	fmt.Fprintln(w, "  version   print version information")
	fmt.Fprintln(w, "  help      show help")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "environment:")
	fmt.Fprintln(w, "  EOSRIFT_CONFIG        config path override")
	fmt.Fprintln(w, "  EOSRIFT_SERVER_ADDR   server base address (recommended)")
	fmt.Fprintln(w, "  EOSRIFT_CONTROL_URL   legacy: full ws(s) control URL")
	fmt.Fprintln(w, "  EOSRIFT_AUTHTOKEN     client auth token")
	fmt.Fprintln(w, "  EOSRIFT_INSPECT_ADDR  local inspector listen address")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "examples:")
	fmt.Fprintln(w, "  eosrift config add-authtoken <token>")
	fmt.Fprintln(w, "  eosrift config set-server https://eosrift.com")
	fmt.Fprintln(w, "  eosrift http 8080 --server https://eosrift.com")
	fmt.Fprintln(w, "  eosrift tcp  5432 --server https://eosrift.com")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
