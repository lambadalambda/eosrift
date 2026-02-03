package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"eosrift.com/eosrift/internal/client"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "tcp":
		tcpCmd(os.Args[2:])
	case "version":
		fmt.Println("eosrift (dev)")
	default:
		usage()
		os.Exit(2)
	}
}

func tcpCmd(args []string) {
	fs := flag.NewFlagSet("tcp", flag.ExitOnError)

	controlURL := fs.String("server", getenv("EOSRIFT_CONTROL_URL", "ws://127.0.0.1:8080/control"), "Control URL (ws/wss)")

	_ = fs.Parse(args)

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: eosrift tcp [--server ws(s)://host/control] <local-port|local-addr>")
		os.Exit(2)
	}

	localAddr := fs.Arg(0)
	if !strings.Contains(localAddr, ":") {
		localAddr = "127.0.0.1:" + localAddr
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tunnel, err := client.StartTCPTunnel(ctx, *controlURL, localAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer tunnel.Close()

	host := controlHost(*controlURL)
	fmt.Printf("Forwarding tcp://%s:%d -> %s\n", host, tunnel.RemotePort, localAddr)

	if err := tunnel.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: eosrift <command> [args]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  tcp       start a TCP tunnel")
	fmt.Fprintln(os.Stderr, "  version   print version information")
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func controlHost(controlURL string) string {
	u, err := url.Parse(controlURL)
	if err != nil {
		return controlURL
	}
	if h := u.Hostname(); h != "" {
		return h
	}
	return controlURL
}
