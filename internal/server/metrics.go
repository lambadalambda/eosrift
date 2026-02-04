package server

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type metrics struct {
	startedAt time.Time
	now       func() time.Time

	activeControl atomic.Int64
	activeHTTP    atomic.Int64
	activeTCP     atomic.Int64

	totalHTTP atomic.Int64
	totalTCP  atomic.Int64
}

func newMetrics(now func() time.Time) *metrics {
	if now == nil {
		now = time.Now
	}
	return &metrics{
		startedAt: now(),
		now:       now,
	}
}

func (m *metrics) trackControlConn() func() {
	m.activeControl.Add(1)
	return func() { m.activeControl.Add(-1) }
}

func (m *metrics) trackHTTPTunnel() func() {
	m.totalHTTP.Add(1)
	m.activeHTTP.Add(1)
	return func() { m.activeHTTP.Add(-1) }
}

func (m *metrics) trackTCPTunnel() func() {
	m.totalTCP.Add(1)
	m.activeTCP.Add(1)
	return func() { m.activeTCP.Add(-1) }
}

func (m *metrics) writePrometheus(w http.ResponseWriter) {
	// Prometheus text format v0.0.4 (minimal).
	// See: https://prometheus.io/docs/instrumenting/exposition_formats/
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	now := m.now()
	uptime := int64(now.Sub(m.startedAt).Seconds())

	writeGauge := func(name, help string, value int64) {
		_, _ = fmt.Fprintf(w, "# HELP %s %s\n", name, help)
		_, _ = fmt.Fprintf(w, "# TYPE %s gauge\n", name)
		_, _ = fmt.Fprintf(w, "%s %d\n", name, value)
	}
	writeCounter := func(name, help string, value int64) {
		_, _ = fmt.Fprintf(w, "# HELP %s %s\n", name, help)
		_, _ = fmt.Fprintf(w, "# TYPE %s counter\n", name)
		_, _ = fmt.Fprintf(w, "%s %d\n", name, value)
	}

	writeGauge("eosrift_uptime_seconds", "Process uptime in seconds.", uptime)
	writeGauge("eosrift_active_control_connections", "Active /control websocket connections.", m.activeControl.Load())
	writeGauge("eosrift_active_http_tunnels", "Active HTTP tunnels.", m.activeHTTP.Load())
	writeGauge("eosrift_active_tcp_tunnels", "Active TCP tunnels.", m.activeTCP.Load())

	writeCounter("eosrift_http_tunnels_total", "Total HTTP tunnels created.", m.totalHTTP.Load())
	writeCounter("eosrift_tcp_tunnels_total", "Total TCP tunnels created.", m.totalTCP.Load())
}

func metricsHandler(token string, m *metrics) http.HandlerFunc {
	token = strings.TrimSpace(token)

	return func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			http.NotFound(w, r)
			return
		}

		ok := false

		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(auth, "Bearer ") && strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")) == token {
			ok = true
		}
		if !ok && strings.TrimSpace(r.URL.Query().Get("token")) == token {
			ok = true
		}
		if !ok {
			http.NotFound(w, r)
			return
		}

		m.writePrometheus(w)
	}
}
