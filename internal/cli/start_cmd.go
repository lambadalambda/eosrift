package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"eosrift.com/eosrift/internal/client"
	"eosrift.com/eosrift/internal/config"
	"eosrift.com/eosrift/internal/inspect"
)

func runStart(ctx context.Context, args []string, configPath string, stdout, stderr io.Writer) int {
	cfg, ok, err := config.Load(configPath)
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

	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(stderr)

	serverAddr := fs.String("server", serverDefault, "Server address (https://host, http://host:port, or ws(s)://host/control)")
	authtoken := fs.String("authtoken", authtokenDefault, "Auth token")
	all := fs.Bool("all", false, "Start all tunnels defined in config")
	inspectEnabled := fs.Bool("inspect", inspectDefault, "Enable local inspector (HTTP tunnels)")
	inspectAddr := fs.String("inspect-addr", inspectAddrDefault, "Inspector listen address")
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
		fmt.Fprintln(out, "  eosrift start web db")
		fmt.Fprintln(out, "  eosrift start --all")
		fmt.Fprintln(out, "  eosrift start --all --server https://eosrift.com")
		fmt.Fprintln(out, "  eosrift start --all --inspect=false")
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

	if !ok {
		fmt.Fprintln(stderr, "error: config file not found:", configPath)
		return 1
	}

	if len(cfg.Tunnels) == 0 {
		fmt.Fprintln(stderr, "error: no tunnels defined in config")
		return 1
	}

	var names []string
	if *all {
		names = make([]string, 0, len(cfg.Tunnels))
		for name := range cfg.Tunnels {
			names = append(names, name)
		}
		sort.Strings(names)
	} else {
		names = fs.Args()
	}

	selected := make([]namedTunnel, 0, len(names))
	for _, name := range names {
		t, ok := cfg.Tunnels[name]
		if !ok {
			fmt.Fprintln(stderr, "error: unknown tunnel:", name)
			return 2
		}
		selected = append(selected, namedTunnel{
			Name:   name,
			Tunnel: t,
		})
	}

	if err := validateNamedTunnels(selected); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	controlURL, err := config.ControlURLFromServerAddr(*serverAddr)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	inspectorCfg := inspectorConfig{
		Enabled: *inspectEnabled,
		Addr:    *inspectAddr,
	}

	defaultHostHeader := strings.TrimSpace(cfg.HostHeader)
	if defaultHostHeader == "" {
		defaultHostHeader = "preserve"
	}

	replayMap := replayTargets{}

	var (
		store         *inspect.Store
		inspectorURL  string
		stopInspector func()
	)

	if needsInspector(selected, inspectorCfg.Enabled) {
		store = inspect.NewStore(inspect.StoreConfig{MaxEntries: 200})

		u, stop, err := startInspectorServer(ctx, store, inspectorCfg.Addr, replayMap.ReplayFunc())
		if err != nil {
			fmt.Fprintln(stderr, "warning: inspector disabled:", err)
			store = nil
		} else {
			inspectorURL = u
			stopInspector = stop
		}
	}
	if stopInspector != nil {
		defer stopInspector()
	}

	started, err := startNamedTunnels(ctx, controlURL, *authtoken, defaultHostHeader, selected, inspectorCfg.Enabled, store, &replayMap)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	defer func() {
		for _, t := range started {
			_ = t.Close()
		}
	}()

	printStartSession(stdout, startSessionOutput{
		Version:   version,
		Status:    "online",
		Inspector: inspectorURL,
		Tunnels:   started,
	})

	if err := waitAll(ctx, started); err != nil && !errors.Is(err, context.Canceled) {
		if ctx.Err() != nil {
			return 0
		}
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	return 0
}

type namedTunnel struct {
	Name   string
	Tunnel config.Tunnel
}

func validateNamedTunnels(tunnels []namedTunnel) error {
	for _, t := range tunnels {
		tn := strings.TrimSpace(t.Name)
		if tn == "" {
			return errors.New("tunnel name is empty")
		}

		addr := strings.TrimSpace(t.Tunnel.Addr)
		if addr == "" {
			return fmt.Errorf("tunnel %q: addr is required", t.Name)
		}

		normalizedAddr := addr
		if !strings.Contains(normalizedAddr, ":") {
			normalizedAddr = "127.0.0.1:" + normalizedAddr
		}
		_, port, err := net.SplitHostPort(normalizedAddr)
		if err != nil {
			return fmt.Errorf("tunnel %q: invalid addr %q: %v", t.Name, addr, err)
		}
		portNum, err := strconv.Atoi(port)
		if err != nil || portNum < 1 || portNum > 65535 {
			return fmt.Errorf("tunnel %q: invalid addr %q: invalid port", t.Name, addr)
		}

		proto := strings.ToLower(strings.TrimSpace(t.Tunnel.Proto))
		if proto == "" {
			return fmt.Errorf("tunnel %q: proto is required (http|tcp)", t.Name)
		}

		switch proto {
		case "http":
			domain := strings.TrimSpace(t.Tunnel.Domain)
			subdomain := strings.TrimSpace(t.Tunnel.Subdomain)
			if domain != "" && subdomain != "" {
				return fmt.Errorf("tunnel %q: only one of domain or subdomain may be set", t.Name)
			}
			if t.Tunnel.RemotePort != 0 {
				return fmt.Errorf("tunnel %q: remote_port is only valid for tcp tunnels", t.Name)
			}
		case "tcp":
			if t.Tunnel.RemotePort < 0 {
				return fmt.Errorf("tunnel %q: remote_port must be >= 0", t.Name)
			}
			if strings.TrimSpace(t.Tunnel.Domain) != "" {
				return fmt.Errorf("tunnel %q: domain is only valid for http tunnels", t.Name)
			}
			if strings.TrimSpace(t.Tunnel.Subdomain) != "" {
				return fmt.Errorf("tunnel %q: subdomain is only valid for http tunnels", t.Name)
			}
			if strings.TrimSpace(t.Tunnel.HostHeader) != "" {
				return fmt.Errorf("tunnel %q: host_header is only valid for http tunnels", t.Name)
			}
		default:
			return fmt.Errorf("tunnel %q: unsupported proto %q", t.Name, proto)
		}
	}

	return nil
}

type inspectorConfig struct {
	Enabled bool
	Addr    string
}

type startedTunnel struct {
	Name string

	ForwardingFrom string
	ForwardingTo   string

	wait  func() error
	close func() error
}

func (t startedTunnel) Wait() error  { return t.wait() }
func (t startedTunnel) Close() error { return t.close() }

type startSessionOutput struct {
	Version string
	Status  string

	Inspector string
	Tunnels   []startedTunnel
}

func printStartSession(w io.Writer, out startSessionOutput) {
	color := wantsColor(w)
	st := ansiStyle{enabled: color}

	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintf(w, "%s %s\n\n", st.brand("Eosrift"), st.dim(out.Version))

	const labelWidth = 14
	row := func(label, value string) {
		_, _ = fmt.Fprintf(w, "  %s  %s\n", st.dim(fmt.Sprintf("%-*s", labelWidth, label)), value)
	}

	row("Session Status", st.ok(out.Status))
	_, _ = fmt.Fprintln(w, "")

	if out.Inspector != "" {
		row("Inspector", st.url(out.Inspector))
		_, _ = fmt.Fprintln(w, "")
	}

	for i, t := range out.Tunnels {
		if i > 0 {
			_, _ = fmt.Fprintln(w, "")
		}
		row("Tunnel", st.dim(t.Name))
		row("Forwarding", fmt.Sprintf("%s %s %s", st.url(t.ForwardingFrom), st.dim("→"), st.dim(t.ForwardingTo)))
	}
}

func needsInspector(tunnels []namedTunnel, inspectDefault bool) bool {
	for _, t := range tunnels {
		if !strings.EqualFold(strings.TrimSpace(t.Tunnel.Proto), "http") {
			continue
		}

		inspectEnabled := inspectDefault
		if t.Tunnel.Inspect != nil {
			inspectEnabled = *t.Tunnel.Inspect
		}
		if inspectEnabled {
			return true
		}
	}
	return false
}

func startInspectorServer(ctx context.Context, store *inspect.Store, addr string, replay func(ctx context.Context, entry inspect.Entry) (inspect.ReplayResult, error)) (string, func(), error) {
	ln, err := listenTCPWithPortFallback(addr, 5000)
	if err != nil {
		return "", nil, err
	}

	inspectorURL := "http://" + displayHostPort(ln.Addr().String())

	srv := &http.Server{
		Handler: inspect.Handler(store, inspect.HandlerOptions{
			Replay: replay,
		}),
	}

	var once sync.Once
	stop := func() {
		once.Do(func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
			_ = ln.Close()
		})
	}

	go func() {
		<-ctx.Done()
		stop()
	}()

	go func() { _ = srv.Serve(ln) }()

	return inspectorURL, stop, nil
}

func startNamedTunnels(ctx context.Context, controlURL, authtoken, defaultHostHeader string, tunnels []namedTunnel, inspectDefault bool, store *inspect.Store, replayMap *replayTargets) ([]startedTunnel, error) {
	var started []startedTunnel

	for _, t := range tunnels {
		tn := strings.TrimSpace(t.Name)
		if tn == "" {
			return nil, errors.New("tunnel name is empty")
		}

		proto := strings.ToLower(strings.TrimSpace(t.Tunnel.Proto))
		if proto == "" {
			return nil, fmt.Errorf("tunnel %q: proto is required (http|tcp)", t.Name)
		}

		localAddr := strings.TrimSpace(t.Tunnel.Addr)
		if localAddr == "" {
			return nil, fmt.Errorf("tunnel %q: addr is required", t.Name)
		}
		if !strings.Contains(localAddr, ":") {
			localAddr = "127.0.0.1:" + localAddr
		}

		switch proto {
		case "http":
			inspectEnabled := store != nil && inspectDefault
			if t.Tunnel.Inspect != nil {
				inspectEnabled = store != nil && *t.Tunnel.Inspect
			}

			hostHeader := strings.TrimSpace(t.Tunnel.HostHeader)
			if hostHeader == "" {
				hostHeader = strings.TrimSpace(defaultHostHeader)
			}
			if hostHeader == "" {
				hostHeader = "preserve"
			}

			tun, err := client.StartHTTPTunnelWithOptions(ctx, controlURL, localAddr, client.HTTPTunnelOptions{
				Authtoken:  authtoken,
				Domain:     strings.TrimSpace(t.Tunnel.Domain),
				Subdomain:  strings.TrimSpace(t.Tunnel.Subdomain),
				HostHeader: hostHeader,
				Inspector: func() *inspect.Store {
					if inspectEnabled {
						return store
					}
					return nil
				}(),
			})
			if err != nil {
				return nil, fmt.Errorf("tunnel %q: %w", t.Name, err)
			}

			if replayMap != nil {
				replayMap.Set(tun.ID, localAddr)
			}

			started = append(started, startedTunnel{
				Name:           t.Name,
				ForwardingFrom: tun.URL,
				ForwardingTo:   displayHostPort(localAddr),
				wait:           tun.Wait,
				close:          tun.Close,
			})
		case "tcp":
			if t.Tunnel.RemotePort < 0 {
				return nil, fmt.Errorf("tunnel %q: remote_port must be >= 0", t.Name)
			}
			tun, err := client.StartTCPTunnelWithOptions(ctx, controlURL, localAddr, client.TCPTunnelOptions{
				Authtoken:  authtoken,
				RemotePort: t.Tunnel.RemotePort,
			})
			if err != nil {
				return nil, fmt.Errorf("tunnel %q: %w", t.Name, err)
			}

			host := controlHost(controlURL)
			started = append(started, startedTunnel{
				Name:           t.Name,
				ForwardingFrom: fmt.Sprintf("tcp://%s:%d", host, tun.RemotePort),
				ForwardingTo:   displayHostPort(localAddr),
				wait:           tun.Wait,
				close:          tun.Close,
			})
		default:
			return nil, fmt.Errorf("tunnel %q: unsupported proto %q", t.Name, proto)
		}
	}

	return started, nil
}

func waitAll(ctx context.Context, tunnels []startedTunnel) error {
	if len(tunnels) == 0 {
		return nil
	}

	errCh := make(chan error, len(tunnels))
	for _, t := range tunnels {
		t := t
		go func() { errCh <- t.Wait() }()
	}

	var firstErr error
	for i := 0; i < len(tunnels); i++ {
		select {
		case <-ctx.Done():
			for _, t := range tunnels {
				_ = t.Close()
			}
			return ctx.Err()
		case err := <-errCh:
			if firstErr == nil {
				firstErr = err
				for _, t := range tunnels {
					_ = t.Close()
				}
			}
		}
	}

	return firstErr
}

type replayTargets struct {
	mu sync.RWMutex
	m  map[string]string
}

func (t *replayTargets) Set(tunnelID, localAddr string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.m == nil {
		t.m = make(map[string]string)
	}
	t.m[tunnelID] = localAddr
}

func (t *replayTargets) get(tunnelID string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.m == nil {
		return "", false
	}
	v, ok := t.m[tunnelID]
	return v, ok
}

func (t *replayTargets) ReplayFunc() func(ctx context.Context, entry inspect.Entry) (inspect.ReplayResult, error) {
	return func(ctx context.Context, entry inspect.Entry) (inspect.ReplayResult, error) {
		localAddr, ok := t.get(entry.TunnelID)
		if !ok || localAddr == "" {
			return inspect.ReplayResult{}, errors.New("unknown tunnel")
		}

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
	}
}
