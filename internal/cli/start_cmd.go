package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
)

func runStart(ctx context.Context, args []string, configPath string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(stderr)

	all := fs.Bool("all", false, "Start all tunnels defined in config")
	help := fs.Bool("help", false, "Show help")
	fs.BoolVar(help, "h", false, "Show help")

	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintln(out, "usage: eosrift start [flags] [<tunnel> ...]")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Start one or more tunnels defined under `tunnels:` in eosrift.yml.")
		fmt.Fprintln(out, "")
		fs.PrintDefaults()
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "examples:")
		fmt.Fprintln(out, "  eosrift start web")
		fmt.Fprintln(out, "  eosrift start --all")
	}

	if err := parseInterspersedFlags(fs, args); err != nil {
		return 2
	}
	if *help {
		fs.SetOutput(stdout)
		fs.Usage()
		return 0
	}

	if *all && fs.NArg() > 0 {
		fmt.Fprintln(stderr, "error: --all cannot be combined with tunnel names")
		return 2
	}
	if !*all && fs.NArg() == 0 {
		fs.Usage()
		return 2
	}

	// Implemented in Milestone 10.
	_ = ctx
	_ = configPath
	fmt.Fprintln(stderr, "error: start is not implemented yet")
	return 1
}
