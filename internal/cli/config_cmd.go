package cli

import (
	"flag"
	"fmt"
	"io"

	"eosrift.com/eosrift/internal/config"
)

func runConfig(args []string, configPath string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		configUsage(stderr)
		return 2
	}

	switch args[0] {
	case "add-authtoken":
		return runConfigAddAuthtoken(args[1:], configPath, stdout, stderr)
	case "check":
		return runConfigCheck(args[1:], configPath, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown config command: %s\n\n", args[0])
		configUsage(stderr)
		return 2
	}
}

func configUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: eosrift config <command> [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  add-authtoken   save authtoken to config")
	fmt.Fprintln(w, "  check           validate config file")
}

func runConfigAddAuthtoken(args []string, configPath string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("config add-authtoken", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: eosrift config add-authtoken <token>")
		return 2
	}

	token := fs.Arg(0)

	cfg, _, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	cfg.Authtoken = token
	if err := config.Save(configPath, cfg); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	fmt.Fprintf(stdout, "Authtoken saved to %s\n", configPath)
	return 0
}

func runConfigCheck(args []string, configPath string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("config check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: eosrift config check")
		return 2
	}

	cfg, ok, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if !ok {
		fmt.Fprintln(stderr, "error: config file not found:", configPath)
		return 1
	}

	fmt.Fprintf(stdout, "Config OK: %s\n", configPath)
	if cfg.Authtoken == "" {
		fmt.Fprintln(stdout, "warning: authtoken is empty (set one with `eosrift config add-authtoken ...`)")
	}
	if cfg.ServerAddr == "" {
		fmt.Fprintln(stdout, "warning: server_addr is empty (set `server_addr:` in the config or pass --server)")
	}

	return 0
}
