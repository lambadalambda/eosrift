package cli

import (
	"flag"
	"fmt"
	"io"

	"eosrift.com/eosrift/internal/client"
	"eosrift.com/eosrift/internal/config"
)

func runConfig(args []string, configPath string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		configUsage(stderr)
		return 2
	}

	switch args[0] {
	case "help", "-h", "--help":
		configUsage(stdout)
		return 0
	case "add-authtoken":
		return runConfigAddAuthtoken(args[1:], configPath, stdout, stderr)
	case "set-server":
		return runConfigSetServer(args[1:], configPath, stdout, stderr)
	case "set-host-header":
		return runConfigSetHostHeader(args[1:], configPath, stdout, stderr)
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
	fmt.Fprintln(w, "  set-server      save server address to config")
	fmt.Fprintln(w, "  set-host-header save host header mode for HTTP tunnels")
	fmt.Fprintln(w, "  check           validate config file")
}

func runConfigAddAuthtoken(args []string, configPath string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("config add-authtoken", flag.ContinueOnError)
	fs.SetOutput(stderr)
	help := fs.Bool("help", false, "Show help")
	fs.BoolVar(help, "h", false, "Show help")
	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintln(out, "usage: eosrift config add-authtoken <token>")
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *help {
		fs.SetOutput(stdout)
		fs.Usage()
		return 0
	}
	if fs.NArg() != 1 {
		fs.Usage()
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

func runConfigSetServer(args []string, configPath string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("config set-server", flag.ContinueOnError)
	fs.SetOutput(stderr)
	help := fs.Bool("help", false, "Show help")
	fs.BoolVar(help, "h", false, "Show help")
	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintln(out, "usage: eosrift config set-server <server-addr>")
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *help {
		fs.SetOutput(stdout)
		fs.Usage()
		return 0
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}

	serverAddr := fs.Arg(0)

	cfg, _, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	cfg.ServerAddr = serverAddr
	if err := config.Save(configPath, cfg); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	fmt.Fprintf(stdout, "Server address saved to %s\n", configPath)
	return 0
}

func runConfigSetHostHeader(args []string, configPath string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("config set-host-header", flag.ContinueOnError)
	fs.SetOutput(stderr)
	help := fs.Bool("help", false, "Show help")
	fs.BoolVar(help, "h", false, "Show help")
	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintln(out, "usage: eosrift config set-host-header <preserve|rewrite|value>")
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *help {
		fs.SetOutput(stdout)
		fs.Usage()
		return 0
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}

	hostHeader := fs.Arg(0)
	if err := client.ValidateHostHeaderMode(hostHeader); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	cfg, _, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	cfg.HostHeader = hostHeader
	if err := config.Save(configPath, cfg); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	fmt.Fprintf(stdout, "Host header mode saved to %s\n", configPath)
	return 0
}

func runConfigCheck(args []string, configPath string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("config check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	help := fs.Bool("help", false, "Show help")
	fs.BoolVar(help, "h", false, "Show help")
	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintln(out, "usage: eosrift config check")
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *help {
		fs.SetOutput(stdout)
		fs.Usage()
		return 0
	}
	if fs.NArg() != 0 {
		fs.Usage()
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
	if cfg.HostHeader != "" {
		if err := client.ValidateHostHeaderMode(cfg.HostHeader); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
	}

	return 0
}
