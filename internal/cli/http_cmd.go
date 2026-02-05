package cli

import (
	"context"
	"crypto/tls"
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
	"eosrift.com/eosrift/internal/control"
	"eosrift.com/eosrift/internal/inspect"
)

func runHTTP(ctx context.Context, args []string, configPath string, stdout, stderr io.Writer) int {
	cfg, _, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	serverDefault := resolveServerAddrDefault(cfg)
	authtokenDefault := resolveAuthtokenDefault(cfg)
	inspectDefault := resolveInspectEnabledDefault(cfg)
	inspectAddrDefault := resolveInspectAddrDefault(cfg)
	hostHeaderDefault := resolveHostHeaderDefault(cfg)

	fs := flag.NewFlagSet("http", flag.ContinueOnError)
	fs.SetOutput(stderr)

	serverAddr := fs.String("server", serverDefault, "Server address (https://host, http://host:port, or ws(s)://host/control)")
	authtoken := fs.String("authtoken", authtokenDefault, "Auth token")
	subdomain := fs.String("subdomain", "", "Reserved subdomain to request (requires server-side reservation)")
	domain := fs.String("domain", "", "Domain to request (must be under the server tunnel domain; auto-reserved on first use)")
	basicAuth := fs.String("basic-auth", "", "Require HTTP basic auth on the public URL (user:pass)")
	var allowCIDR stringSliceFlag
	fs.Var(&allowCIDR, "allow-cidr", "Allow client IPs matching CIDR or IP (repeatable)")
	var denyCIDR stringSliceFlag
	fs.Var(&denyCIDR, "deny-cidr", "Deny client IPs matching CIDR or IP (repeatable)")
	var allowMethod stringSliceFlag
	fs.Var(&allowMethod, "allow-method", "Allow HTTP method(s) (repeatable)")
	var allowPath stringSliceFlag
	fs.Var(&allowPath, "allow-path", "Allow exact request path(s) (repeatable, must start with /)")
	var allowPathPrefix stringSliceFlag
	fs.Var(&allowPathPrefix, "allow-path-prefix", "Allow request path prefix(es) (repeatable, must start with /)")
	var requestHeaderAdd stringListFlag
	fs.Var(&requestHeaderAdd, "request-header-add", "Add/override a request header (repeatable, \"Name: value\")")
	var requestHeaderRemove stringListFlag
	fs.Var(&requestHeaderRemove, "request-header-remove", "Remove a request header (repeatable, \"Name\")")
	var responseHeaderAdd stringListFlag
	fs.Var(&responseHeaderAdd, "response-header-add", "Add/override a response header (repeatable, \"Name: value\")")
	var responseHeaderRemove stringListFlag
	fs.Var(&responseHeaderRemove, "response-header-remove", "Remove a response header (repeatable, \"Name\")")
	hostHeader := fs.String("host-header", hostHeaderDefault, "Host header mode: preserve (default), rewrite, or a literal value")
	upstreamTLSSkipVerify := fs.Bool("upstream-tls-skip-verify", false, "Disable certificate verification for HTTPS upstreams")
	inspectEnabled := fs.Bool("inspect", inspectDefault, "Enable local inspector")
	inspectAddr := fs.String("inspect-addr", inspectAddrDefault, "Inspector listen address")
	help := fs.Bool("help", false, "Show help")
	fs.BoolVar(help, "h", false, "Show help")

	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintln(out, "usage: eosrift http [flags] <local-port|local-addr|local-url>")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "flags may appear before or after <local-port|local-addr>")
		fs.PrintDefaults()
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "examples:")
		fmt.Fprintln(out, "  eosrift http 3000")
		fmt.Fprintln(out, "  eosrift http 3000 --domain demo.tunnel.eosrift.com")
		fmt.Fprintln(out, "  eosrift http 3000 --subdomain demo")
		fmt.Fprintln(out, "  eosrift http 3000 --basic-auth user:pass")
		fmt.Fprintln(out, "  eosrift http 3000 --allow-cidr 203.0.113.0/24")
		fmt.Fprintln(out, "  eosrift http 3000 --allow-method GET --allow-path /healthz")
		fmt.Fprintln(out, "  eosrift http 3000 --request-header-add \"X-API-Key: secret\"")
		fmt.Fprintln(out, "  eosrift http 3000 --host-header=rewrite")
		fmt.Fprintln(out, "  eosrift http https://127.0.0.1:8443 --upstream-tls-skip-verify")
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
	if strings.TrimSpace(*basicAuth) != "" && !strings.Contains(*basicAuth, ":") {
		fmt.Fprintln(stderr, "error: --basic-auth must be in the form user:pass")
		return 2
	}
	if err := validateCIDRs("allow_cidr", []string(allowCIDR)); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	if err := validateCIDRs("deny_cidr", []string(denyCIDR)); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	parsedAllowMethods, err := control.ParseHTTPMethodList("allow_method", []string(allowMethod), 0)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	parsedAllowPaths, err := control.ParsePathList("allow_path", []string(allowPath), 0)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	parsedAllowPathPrefixes, err := control.ParsePathList("allow_path_prefix", []string(allowPathPrefix), 0)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	parsedRequestHeaderAdd, err := parseHeaderAddList("request_header_add", []string(requestHeaderAdd))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	parsedRequestHeaderRemove, err := parseHeaderRemoveList("request_header_remove", []string(requestHeaderRemove))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	parsedResponseHeaderAdd, err := parseHeaderAddList("response_header_add", []string(responseHeaderAdd))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	parsedResponseHeaderRemove, err := parseHeaderRemoveList("response_header_remove", []string(responseHeaderRemove))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	upstreamScheme, localAddr, err := parseHTTPUpstreamTarget(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
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
		Authtoken:             *authtoken,
		Subdomain:             *subdomain,
		Domain:                *domain,
		BasicAuth:             *basicAuth,
		AllowMethods:          parsedAllowMethods,
		AllowPaths:            parsedAllowPaths,
		AllowPathPrefixes:     parsedAllowPathPrefixes,
		AllowCIDRs:            []string(allowCIDR),
		DenyCIDRs:             []string(denyCIDR),
		RequestHeaderAdd:      parsedRequestHeaderAdd,
		RequestHeaderRemove:   parsedRequestHeaderRemove,
		ResponseHeaderAdd:     parsedResponseHeaderAdd,
		ResponseHeaderRemove:  parsedResponseHeaderRemove,
		HostHeader:            *hostHeader,
		UpstreamScheme:        upstreamScheme,
		UpstreamTLSSkipVerify: *upstreamTLSSkipVerify,
		Inspector:             store,
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
							Scheme:   upstreamScheme,
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
						if upstreamScheme == "https" && *upstreamTLSSkipVerify {
							clientHTTP.Transport = &http.Transport{
								ForceAttemptHTTP2: false,
								TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
							}
						}
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
