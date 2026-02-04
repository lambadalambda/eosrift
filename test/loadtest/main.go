package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"eosrift.com/eosrift/internal/client"
	"eosrift.com/eosrift/internal/config"
)

func main() {
	serverURLDefault := getenv("EOSRIFT_SERVER_URL", "http://server:8080")
	authtokenDefault := getenv("EOSRIFT_AUTHTOKEN", "")
	modeDefault := getenv("EOSRIFT_LOAD_MODE", "http")
	requestsDefault := atoi(getenv("EOSRIFT_LOAD_REQUESTS", "2000"), 2000)
	concurrencyDefault := atoi(getenv("EOSRIFT_LOAD_CONCURRENCY", "50"), 50)
	timeoutDefault := parseDuration(getenv("EOSRIFT_LOAD_TIMEOUT", "5s"), 5*time.Second)
	tcpPayloadDefault := atoi(getenv("EOSRIFT_LOAD_TCP_PAYLOAD_BYTES", "1024"), 1024)

	var (
		serverURL     string
		authtoken     string
		mode          string
		requests      int
		concurrency   int
		timeout       time.Duration
		tcpPayloadLen int
	)

	flag.StringVar(&serverURL, "server", serverURLDefault, "Server base URL (http(s)://host[:port])")
	flag.StringVar(&authtoken, "authtoken", authtokenDefault, "Client authtoken")
	flag.StringVar(&mode, "mode", modeDefault, "Load mode: http or tcp")
	flag.IntVar(&requests, "requests", requestsDefault, "Total requests/connections to make")
	flag.IntVar(&concurrency, "concurrency", concurrencyDefault, "Number of concurrent workers")
	flag.DurationVar(&timeout, "timeout", timeoutDefault, "Per-request timeout")
	flag.IntVar(&tcpPayloadLen, "tcp-payload-bytes", tcpPayloadDefault, "TCP payload size to echo (bytes)")
	flag.Parse()

	mode = strings.ToLower(strings.TrimSpace(mode))

	if strings.TrimSpace(authtoken) == "" {
		fatal("missing authtoken (set -authtoken or EOSRIFT_AUTHTOKEN)")
	}
	if requests <= 0 {
		fatal("requests must be > 0")
	}
	if concurrency <= 0 {
		fatal("concurrency must be > 0")
	}
	if concurrency > requests {
		concurrency = requests
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if tcpPayloadLen <= 0 {
		tcpPayloadLen = 1024
	}

	if err := waitForHealth(serverURL, 30*time.Second); err != nil {
		fatal(err.Error())
	}

	controlURL, err := config.ControlURLFromServerAddr(serverURL)
	if err != nil {
		fatal("control url: " + err.Error())
	}

	switch mode {
	case "http":
		if err := runHTTP(serverURL, controlURL, authtoken, requests, concurrency, timeout); err != nil {
			fatal(err.Error())
		}
	case "tcp":
		if err := runTCP(serverURL, controlURL, authtoken, requests, concurrency, timeout, tcpPayloadLen); err != nil {
			fatal(err.Error())
		}
	default:
		fatal("invalid mode: " + mode)
	}
}

func runHTTP(serverURL, controlURL, authtoken string, requests, concurrency int, timeout time.Duration) error {
	upLn, upSrv := startUpstreamHTTP()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = upSrv.Shutdown(ctx)
		_ = upLn.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tunnel, err := client.StartHTTPTunnelWithOptions(ctx, controlURL, upLn.Addr().String(), client.HTTPTunnelOptions{
		Authtoken: authtoken,
	})
	if err != nil {
		return fmt.Errorf("start http tunnel: %w", err)
	}
	defer tunnel.Close()

	publicHost, err := hostFromURL(tunnel.URL)
	if err != nil {
		return fmt.Errorf("parse tunnel url: %w", err)
	}

	fmt.Printf("mode=http requests=%d concurrency=%d timeout=%s\n", requests, concurrency, timeout)
	fmt.Printf("forwarding=%s -> %s\n", tunnel.URL, upLn.Addr().String())

	res := runHTTPLoad(serverURL, publicHost, "/load", requests, concurrency, timeout)
	printSummary(res)
	return nil
}

func runTCP(serverURL, controlURL, authtoken string, requests, concurrency int, timeout time.Duration, payloadLen int) error {
	upLn := startUpstreamEcho()
	defer upLn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tunnel, err := client.StartTCPTunnelWithOptions(ctx, controlURL, upLn.Addr().String(), client.TCPTunnelOptions{
		Authtoken: authtoken,
	})
	if err != nil {
		return fmt.Errorf("start tcp tunnel: %w", err)
	}
	defer tunnel.Close()

	serverHost, err := hostFromServerURL(serverURL)
	if err != nil {
		return err
	}
	remoteAddr := fmt.Sprintf("%s:%d", serverHost, tunnel.RemotePort)

	fmt.Printf("mode=tcp requests=%d concurrency=%d timeout=%s payload_bytes=%d\n", requests, concurrency, timeout, payloadLen)
	fmt.Printf("forwarding=tcp://%s -> %s\n", remoteAddr, upLn.Addr().String())

	res := runTCPLoad(remoteAddr, payloadLen, requests, concurrency, timeout)
	printSummary(res)
	return nil
}

type loadResult struct {
	Mode        string
	Total       int
	OK          int64
	Errors      int64
	Elapsed     time.Duration
	Latencies   []time.Duration
	ErrorSample string
}

func runHTTPLoad(serverURL, publicHost, path string, requests, concurrency int, timeout time.Duration) loadResult {
	res := loadResult{
		Mode:      "http",
		Total:     requests,
		Latencies: make([]time.Duration, requests),
	}

	tr := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        concurrency * 4,
		MaxIdleConnsPerHost: concurrency * 4,
		IdleConnTimeout:     30 * time.Second,
	}
	clientHTTP := &http.Client{Transport: tr}
	defer tr.CloseIdleConnections()

	jobs := make(chan int, requests)
	for i := 0; i < requests; i++ {
		jobs <- i
	}
	close(jobs)

	var (
		okCount   atomic.Int64
		errCount  atomic.Int64
		errOnce   sync.Once
		errSample atomic.Value // string
	)

	started := time.Now()

	var wg sync.WaitGroup
	wg.Add(concurrency)
	for w := 0; w < concurrency; w++ {
		go func() {
			defer wg.Done()
			for i := range jobs {
				t0 := time.Now()

				reqCtx, cancel := context.WithTimeout(context.Background(), timeout)
				req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, strings.TrimRight(serverURL, "/")+path, nil)
				if err == nil {
					req.Host = publicHost
					resp, doErr := clientHTTP.Do(req)
					if doErr == nil {
						_, _ = io.Copy(io.Discard, resp.Body)
						_ = resp.Body.Close()
						if resp.StatusCode == http.StatusOK {
							okCount.Add(1)
						} else {
							errCount.Add(1)
							errOnce.Do(func() { errSample.Store(fmt.Sprintf("status=%d", resp.StatusCode)) })
						}
					} else {
						errCount.Add(1)
						errOnce.Do(func() { errSample.Store(doErr.Error()) })
					}
				} else {
					errCount.Add(1)
					errOnce.Do(func() { errSample.Store(err.Error()) })
				}
				cancel()

				res.Latencies[i] = time.Since(t0)
			}
		}()
	}
	wg.Wait()

	res.Elapsed = time.Since(started)
	res.OK = okCount.Load()
	res.Errors = errCount.Load()
	if v := errSample.Load(); v != nil {
		res.ErrorSample, _ = v.(string)
	}

	return res
}

func runTCPLoad(remoteAddr string, payloadLen, requests, concurrency int, timeout time.Duration) loadResult {
	res := loadResult{
		Mode:      "tcp",
		Total:     requests,
		Latencies: make([]time.Duration, requests),
	}

	payload := bytes.Repeat([]byte("a"), payloadLen)

	jobs := make(chan int, requests)
	for i := 0; i < requests; i++ {
		jobs <- i
	}
	close(jobs)

	var (
		okCount   atomic.Int64
		errCount  atomic.Int64
		errOnce   sync.Once
		errSample atomic.Value // string
	)

	started := time.Now()

	var wg sync.WaitGroup
	wg.Add(concurrency)
	for w := 0; w < concurrency; w++ {
		go func() {
			defer wg.Done()
			buf := make([]byte, payloadLen)

			for i := range jobs {
				t0 := time.Now()

				dl := time.Now().Add(timeout)
				conn, err := net.DialTimeout("tcp", remoteAddr, timeout)
				if err != nil {
					errCount.Add(1)
					errOnce.Do(func() { errSample.Store(err.Error()) })
					res.Latencies[i] = time.Since(t0)
					continue
				}

				_ = conn.SetDeadline(dl)
				if _, err := conn.Write(payload); err != nil {
					_ = conn.Close()
					errCount.Add(1)
					errOnce.Do(func() { errSample.Store(err.Error()) })
					res.Latencies[i] = time.Since(t0)
					continue
				}

				if _, err := io.ReadFull(conn, buf); err != nil {
					_ = conn.Close()
					errCount.Add(1)
					errOnce.Do(func() { errSample.Store(err.Error()) })
					res.Latencies[i] = time.Since(t0)
					continue
				}

				_ = conn.Close()

				if bytes.Equal(buf, payload) {
					okCount.Add(1)
				} else {
					errCount.Add(1)
					errOnce.Do(func() { errSample.Store("echo mismatch") })
				}

				res.Latencies[i] = time.Since(t0)
			}
		}()
	}
	wg.Wait()

	res.Elapsed = time.Since(started)
	res.OK = okCount.Load()
	res.Errors = errCount.Load()
	if v := errSample.Load(); v != nil {
		res.ErrorSample, _ = v.(string)
	}

	return res
}

func printSummary(res loadResult) {
	rps := float64(res.Total) / res.Elapsed.Seconds()
	if math.IsNaN(rps) || math.IsInf(rps, 0) {
		rps = 0
	}

	durs := make([]time.Duration, 0, len(res.Latencies))
	for _, d := range res.Latencies {
		if d > 0 {
			durs = append(durs, d)
		}
	}
	sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })

	avg := time.Duration(0)
	for _, d := range durs {
		avg += d
	}
	if len(durs) > 0 {
		avg /= time.Duration(len(durs))
	}

	fmt.Printf("\nresults mode=%s total=%d ok=%d errors=%d elapsed=%s rps=%.1f\n", res.Mode, res.Total, res.OK, res.Errors, res.Elapsed.Truncate(time.Millisecond), rps)
	if res.ErrorSample != "" {
		fmt.Printf("error_sample=%q\n", res.ErrorSample)
	}

	if len(durs) == 0 {
		fmt.Printf("latency: no samples\n")
		return
	}

	fmt.Printf("latency avg=%s p50=%s p95=%s p99=%s max=%s\n",
		avg.Truncate(time.Microsecond),
		percentile(durs, 0.50).Truncate(time.Microsecond),
		percentile(durs, 0.95).Truncate(time.Microsecond),
		percentile(durs, 0.99).Truncate(time.Microsecond),
		durs[len(durs)-1].Truncate(time.Microsecond),
	)
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}

	i := int(math.Ceil(float64(len(sorted))*p)) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(sorted) {
		i = len(sorted) - 1
	}
	return sorted[i]
}

func startUpstreamHTTP() (net.Listener, *http.Server) {
	upstream := http.NewServeMux()
	upstream.HandleFunc("/load", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fatal("listen upstream: " + err.Error())
	}

	srv := &http.Server{
		Handler:           upstream,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = srv.Serve(ln) }()

	return ln, srv
}

func startUpstreamEcho() net.Listener {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fatal("listen echo: " + err.Error())
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()

	return ln
}

func hostFromURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		return "", fmt.Errorf("missing host in url: %q", raw)
	}
	return u.Host, nil
}

func hostFromServerURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("parse server url: %w", err)
	}

	host := u.Host
	if host == "" {
		return "", fmt.Errorf("missing host in server url: %q", raw)
	}

	if h, _, err := net.SplitHostPort(host); err == nil {
		return h, nil
	}
	return host, nil
}

func waitForHealth(serverURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	u := strings.TrimRight(strings.TrimSpace(serverURL), "/") + "/healthz"

	clientHTTP := &http.Client{Timeout: 2 * time.Second}

	for {
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err == nil {
			resp, err := clientHTTP.Do(req)
			if err == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("server not healthy at %s within %s", u, timeout)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func atoi(s string, fallback int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(2)
}
