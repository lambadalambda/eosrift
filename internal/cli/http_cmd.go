package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"eosrift.com/eosrift/internal/client"
	"eosrift.com/eosrift/internal/config"
	"eosrift.com/eosrift/internal/inspect"
)

func runHTTP(ctx context.Context, args []string, configPath string, stdout, stderr io.Writer) int {
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

	inspectDefault := true
	if cfg.Inspect != nil {
		inspectDefault = *cfg.Inspect
	}

	inspectAddrDefault := getenv("EOSRIFT_INSPECT_ADDR", "")
	if inspectAddrDefault == "" {
		inspectAddrDefault = cfg.InspectAddr
	}
	if inspectAddrDefault == "" {
		inspectAddrDefault = "127.0.0.1:4040"
	}

	hostHeaderDefault := cfg.HostHeader
	if strings.TrimSpace(hostHeaderDefault) == "" {
		hostHeaderDefault = "preserve"
	}

	fs := flag.NewFlagSet("http", flag.ContinueOnError)
	fs.SetOutput(stderr)

	serverAddr := fs.String("server", serverDefault, "Server address (https://host, http://host:port, or ws(s)://host/control)")
	authtoken := fs.String("authtoken", authtokenDefault, "Auth token")
	subdomain := fs.String("subdomain", "", "Reserved subdomain to request (requires server-side reservation)")
	domain := fs.String("domain", "", "Domain to request (must be under the server tunnel domain; auto-reserved on first use)")
	hostHeader := fs.String("host-header", hostHeaderDefault, "Host header mode: preserve (default), rewrite, or a literal value")
	inspectEnabled := fs.Bool("inspect", inspectDefault, "Enable local inspector")
	inspectAddr := fs.String("inspect-addr", inspectAddrDefault, "Inspector listen address")
	help := fs.Bool("help", false, "Show help")
	fs.BoolVar(help, "h", false, "Show help")

	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintln(out, "usage: eosrift http [flags] <local-port|local-addr>")
		fs.PrintDefaults()
	}

	if err := parseInterspersedFlags(fs, args); err != nil {
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

	if strings.TrimSpace(*subdomain) != "" && strings.TrimSpace(*domain) != "" {
		fmt.Fprintln(stderr, "error: only one of --subdomain or --domain may be set")
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

	var store *inspect.Store
	if *inspectEnabled {
		store = inspect.NewStore(inspect.StoreConfig{MaxEntries: 200})
	}

	tunnel, err := client.StartHTTPTunnelWithOptions(ctx, controlURL, localAddr, client.HTTPTunnelOptions{
		Authtoken:  *authtoken,
		Subdomain:  *subdomain,
		Domain:     *domain,
		HostHeader: *hostHeader,
		Inspector:  store,
	})
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	defer tunnel.Close()

	inspectorURL := ""
	if store != nil {
		ln, err := listenTCPWithPortFallback(*inspectAddr, 5000)
		if err != nil {
			fmt.Fprintln(stderr, "warning: inspector disabled:", err)
		} else {
			inspectorURL = "http://" + displayHostPort(ln.Addr().String())

			srv := &http.Server{
				Handler: inspect.Handler(store, inspect.HandlerOptions{
					Replay: func(ctx context.Context, entry inspect.Entry) (inspect.ReplayResult, error) {
						u, err := url.ParseRequestURI(entry.Path)
						if err != nil {
							return inspect.ReplayResult{}, err
						}

						dst := &url.URL{
							Scheme:   "http",
							Host:     localAddr,
							Path:     u.Path,
							RawQuery: u.RawQuery,
						}

						req, err := http.NewRequestWithContext(ctx, entry.Method, dst.String(), nil)
						if err != nil {
							return inspect.ReplayResult{}, err
						}

						if entry.Host != "" {
							req.Host = entry.Host
						}

						req.Header = entry.RequestHeaders.Clone()
						stripHopByHopHeaders(req.Header)
						req.Header.Del("Content-Length")
						req.Header.Del("Transfer-Encoding")

						clientHTTP := &http.Client{Timeout: 10 * time.Second}
						resp, err := clientHTTP.Do(req)
						if err != nil {
							return inspect.ReplayResult{}, err
						}
						_, _ = io.Copy(io.Discard, resp.Body)
						_ = resp.Body.Close()

						return inspect.ReplayResult{StatusCode: resp.StatusCode}, nil
					},
				}),
			}

			go func() {
				<-ctx.Done()
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = srv.Shutdown(shutdownCtx)
			}()

			go func() { _ = srv.Serve(ln) }()
		}
	}

	printSession(stdout, sessionOutput{
		Version:        version,
		Status:         "online",
		ForwardingFrom: tunnel.URL,
		ForwardingTo:   displayHostPort(localAddr),
		Inspector:      inspectorURL,
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
