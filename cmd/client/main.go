package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"eosrift.com/eosrift/internal/client"
	"eosrift.com/eosrift/internal/inspect"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "http":
		httpCmd(os.Args[2:])
	case "tcp":
		tcpCmd(os.Args[2:])
	case "version":
		fmt.Println("eosrift (dev)")
	default:
		usage()
		os.Exit(2)
	}
}

func httpCmd(args []string) {
	fs := flag.NewFlagSet("http", flag.ExitOnError)

	controlURL := fs.String("server", getenv("EOSRIFT_CONTROL_URL", "ws://127.0.0.1:8080/control"), "Control URL (ws/wss)")
	authtokenDefault := getenv("EOSRIFT_AUTHTOKEN", "")
	if authtokenDefault == "" {
		authtokenDefault = getenv("EOSRIFT_AUTH_TOKEN", "")
	}
	authtoken := fs.String("authtoken", authtokenDefault, "Auth token")
	inspectEnabled := fs.Bool("inspect", true, "Enable local inspector")
	inspectAddr := fs.String("inspect-addr", getenv("EOSRIFT_INSPECT_ADDR", "127.0.0.1:4040"), "Inspector listen address")

	_ = fs.Parse(args)

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: eosrift http [--server ws(s)://host/control] <local-port|local-addr>")
		os.Exit(2)
	}

	localAddr := fs.Arg(0)
	if !strings.Contains(localAddr, ":") {
		localAddr = "127.0.0.1:" + localAddr
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var store *inspect.Store
	if *inspectEnabled {
		store = inspect.NewStore(inspect.StoreConfig{MaxEntries: 200})

		ln, err := net.Listen("tcp", *inspectAddr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "warning: inspector disabled:", err)
			store = nil
		} else {
			srv := &http.Server{Handler: inspect.Handler(store)}

			go func() {
				<-ctx.Done()
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = srv.Shutdown(shutdownCtx)
			}()

			go func() { _ = srv.Serve(ln) }()

			fmt.Printf("Inspector http://%s\n", ln.Addr().String())
		}
	}

	tunnel, err := client.StartHTTPTunnelWithOptions(ctx, *controlURL, localAddr, client.HTTPTunnelOptions{
		Authtoken: *authtoken,
		Inspector: store,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer tunnel.Close()

	fmt.Printf("Forwarding %s -> %s\n", tunnel.URL, localAddr)

	if err := tunnel.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func tcpCmd(args []string) {
	fs := flag.NewFlagSet("tcp", flag.ExitOnError)

	controlURL := fs.String("server", getenv("EOSRIFT_CONTROL_URL", "ws://127.0.0.1:8080/control"), "Control URL (ws/wss)")
	authtokenDefault := getenv("EOSRIFT_AUTHTOKEN", "")
	if authtokenDefault == "" {
		authtokenDefault = getenv("EOSRIFT_AUTH_TOKEN", "")
	}
	authtoken := fs.String("authtoken", authtokenDefault, "Auth token")

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

	tunnel, err := client.StartTCPTunnelWithOptions(ctx, *controlURL, localAddr, client.TCPTunnelOptions{
		Authtoken: *authtoken,
	})
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
	fmt.Fprintln(os.Stderr, "  http      start an HTTP tunnel")
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
