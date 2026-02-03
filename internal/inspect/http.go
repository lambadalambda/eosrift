package inspect

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type listRequestsResponse struct {
	Requests []Entry `json:"requests"`
}

type HandlerOptions struct {
	Replay func(ctx context.Context, entry Entry) (ReplayResult, error)
}

type ReplayResult struct {
	StatusCode int `json:"status_code,omitempty"`
}

type replayResponse struct {
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

func Handler(store *Store, opts HandlerOptions) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(inspectorIndexHTML))
	})

	mux.HandleFunc("/api/requests", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(listRequestsResponse{
			Requests: store.List(),
		})
	})

	mux.HandleFunc("/api/requests/", func(w http.ResponseWriter, r *http.Request) {
		// POST /api/requests/<id>/replay
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		rest := strings.TrimPrefix(r.URL.Path, "/api/requests/")
		parts := strings.Split(rest, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] != "replay" {
			http.NotFound(w, r)
			return
		}

		entry, ok := store.Get(parts[0])
		if !ok {
			http.NotFound(w, r)
			return
		}

		if opts.Replay == nil {
			http.Error(w, "replay not configured", http.StatusNotImplemented)
			return
		}

		res, err := opts.Replay(r.Context(), entry)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(replayResponse{
				Error: err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(replayResponse{
			StatusCode: res.StatusCode,
		})
	})

	return mux
}

const inspectorIndexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width,initial-scale=1" />
  <title>EosRift Inspector</title>
  <style>
    :root {
      --bg: #0b1020;
      --panel: #0f172a;
      --border: #1f2937;
      --text: #e5e7eb;
      --muted: #9ca3af;
      --accent: #60a5fa;
      --good: #10b981;
      --warn: #f59e0b;
      --bad: #ef4444;
    }

    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial, "Apple Color Emoji", "Segoe UI Emoji";
      background: var(--bg);
      color: var(--text);
    }
    header {
      position: sticky;
      top: 0;
      z-index: 10;
      background: rgba(11, 16, 32, 0.92);
      backdrop-filter: blur(8px);
      border-bottom: 1px solid var(--border);
      padding: 12px 16px;
      display: flex;
      justify-content: space-between;
      gap: 12px;
      align-items: center;
    }
    header .title {
      font-weight: 650;
      letter-spacing: 0.2px;
    }
    header .subtitle {
      color: var(--muted);
      font-size: 12px;
      margin-top: 2px;
    }
    header .right {
      display: flex;
      align-items: center;
      gap: 10px;
      color: var(--muted);
      font-size: 12px;
      white-space: nowrap;
    }
    header code {
      color: var(--text);
      background: rgba(255,255,255,0.06);
      padding: 2px 6px;
      border-radius: 6px;
      border: 1px solid rgba(255,255,255,0.08);
    }

    .layout {
      display: grid;
      grid-template-columns: 520px 1fr;
      min-height: calc(100vh - 56px);
    }

    .list {
      border-right: 1px solid var(--border);
      background: rgba(15, 23, 42, 0.18);
      overflow-y: auto;
      overflow-x: hidden;
    }
    .row {
      padding: 10px 14px;
      border-bottom: 1px solid rgba(255,255,255,0.06);
      cursor: pointer;
      display: grid;
      grid-template-columns: 54px 1fr auto;
      gap: 12px;
      align-items: center;
    }
    .row:hover { background: rgba(96,165,250,0.08); }
    .row.selected { background: rgba(96,165,250,0.14); }
    .method {
      font-weight: 700;
      font-size: 11px;
      letter-spacing: 0.3px;
      color: var(--accent);
    }
    .pathwrap {
      min-width: 0;
    }
    .path {
      font-size: 13px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .meta {
      margin-top: 2px;
      font-size: 11px;
      color: var(--muted);
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .status {
      flex-shrink: 0;
      font-size: 12px;
      font-weight: 700;
      padding: 3px 10px;
      border-radius: 999px;
      border: 1px solid rgba(255,255,255,0.1);
      background: rgba(255,255,255,0.04);
      text-align: center;
      min-width: 42px;
    }

    .detail {
      padding: 16px;
      overflow: auto;
    }
    .card {
      border: 1px solid var(--border);
      background: rgba(15, 23, 42, 0.35);
      border-radius: 12px;
      padding: 14px;
    }
    .card h2 {
      margin: 0 0 8px 0;
      font-size: 14px;
      font-weight: 650;
      letter-spacing: 0.2px;
    }
    .kv {
      display: grid;
      grid-template-columns: 90px 1fr;
      gap: 10px 16px;
      align-items: baseline;
      font-size: 13px;
    }
    .k { color: var(--muted); flex-shrink: 0; }
    .v { color: var(--text); overflow-wrap: anywhere; word-break: break-all; min-width: 0; }
    .actions {
      margin-top: 12px;
      display: flex;
      gap: 10px;
      align-items: center;
    }
    button {
      cursor: pointer;
      border: 1px solid rgba(255,255,255,0.14);
      background: rgba(96,165,250,0.14);
      color: var(--text);
      padding: 8px 10px;
      border-radius: 10px;
      font-weight: 650;
      font-size: 13px;
    }
    button:hover { background: rgba(96,165,250,0.22); }
    button:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }
    .hint {
      color: var(--muted);
      font-size: 12px;
    }
    pre {
      margin: 10px 0 0 0;
      padding: 10px;
      border-radius: 10px;
      border: 1px solid rgba(255,255,255,0.1);
      background: rgba(0,0,0,0.22);
      overflow: auto;
      font-size: 12px;
      line-height: 1.4;
    }
    a { color: var(--accent); text-decoration: none; }
    a:hover { text-decoration: underline; }

    @media (max-width: 900px) {
      .layout { grid-template-columns: 1fr; }
      .list { border-right: none; border-bottom: 1px solid var(--border); }
    }
  </style>
</head>
<body>
  <header>
    <div>
      <div class="title">EosRift Inspector</div>
      <div class="subtitle">Local request history • refreshes every second</div>
    </div>
    <div class="right">
      API <code>/api/requests</code>
    </div>
  </header>

  <div class="layout">
    <div class="list" id="list"></div>
    <div class="detail">
      <div class="card" id="card">
        <h2>Request</h2>
        <div class="hint">No request selected yet.</div>
      </div>
    </div>
  </div>

  <script>
    const state = {
      requests: [],
      selectedId: null,
      lastError: null,
    };

    function esc(value) {
      return String(value)
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;")
        .replaceAll('"', "&quot;")
        .replaceAll("'", "&#39;");
    }

    function statusColor(statusCode) {
      if (!statusCode) return "rgba(255,255,255,0.18)";
      if (statusCode >= 200 && statusCode < 300) return "rgba(16,185,129,0.35)";
      if (statusCode >= 300 && statusCode < 400) return "rgba(96,165,250,0.35)";
      if (statusCode >= 400 && statusCode < 500) return "rgba(245,158,11,0.35)";
      return "rgba(239,68,68,0.35)";
    }

    function formatStartedAt(ts) {
      try {
        return new Date(ts).toLocaleString();
      } catch {
        return String(ts || "");
      }
    }

    function toHeadersText(headers) {
      if (!headers) return "";
      const lines = [];
      const keys = Object.keys(headers).sort((a, b) => a.localeCompare(b));
      for (const k of keys) {
        const v = headers[k];
        if (Array.isArray(v)) {
          for (const vv of v) lines.push(k + ": " + vv);
        } else {
          lines.push(k + ": " + v);
        }
      }
      return lines.join("\n");
    }

    function renderList() {
      const list = document.getElementById("list");
      const items = state.requests;

      if (!items || items.length === 0) {
        list.innerHTML = '<div class="row" style="cursor:default;grid-template-columns:1fr;"><div class="hint">No requests yet. Make some traffic through your tunnel.</div></div>';
        return;
      }

      list.innerHTML = items.map((r) => {
        const selected = r.id === state.selectedId ? "selected" : "";
        const host = r.host ? r.host : "";
        const status = r.status_code ? String(r.status_code) : "";
        const started = r.started_at ? formatStartedAt(r.started_at) : "";
        return (
          '<div class="row ' + selected + '" data-id="' + esc(r.id) + '">' +
            '<div class="method">' + esc(r.method || "") + '</div>' +
            '<div class="pathwrap">' +
              '<div class="path">' + esc(r.path || "") + '</div>' +
              '<div class="meta">' + esc(host) + ' • ' + esc(started) + '</div>' +
            '</div>' +
            '<div class="status" style="background:' + statusColor(r.status_code) + ';">' + esc(status) + '</div>' +
          '</div>'
        );
      }).join("");

      for (const el of list.querySelectorAll(".row[data-id]")) {
        el.addEventListener("click", () => {
          const id = el.getAttribute("data-id");
          state.selectedId = id;
          renderList();
          renderDetails();
        });
      }
    }

    function renderDetails() {
      const card = document.getElementById("card");
      const entry = state.requests.find((r) => r.id === state.selectedId);

      if (!entry) {
        card.innerHTML = '<h2>Request</h2><div class="hint">No request selected.</div>';
        return;
      }

      const url = (entry.host ? ("https://" + entry.host) : "") + (entry.path || "");
      const status = entry.status_code ? String(entry.status_code) : "";
      const started = entry.started_at ? formatStartedAt(entry.started_at) : "";
      const dur = entry.duration_ms ? (String(entry.duration_ms) + "ms") : "";
      const bytes = (entry.bytes_in || 0) + " in / " + (entry.bytes_out || 0) + " out";

      card.innerHTML =
        '<h2>Request</h2>' +
        '<div class="kv">' +
          '<div class="k">Method</div><div class="v">' + esc(entry.method || "") + '</div>' +
          '<div class="k">Status</div><div class="v">' + esc(status) + '</div>' +
          '<div class="k">URL</div><div class="v"><a href="' + esc(url) + '" target="_blank" rel="noreferrer">' + esc(url) + '</a></div>' +
          '<div class="k">Tunnel</div><div class="v">' + esc(entry.tunnel_id || "") + '</div>' +
          '<div class="k">Started</div><div class="v">' + esc(started) + '</div>' +
          '<div class="k">Duration</div><div class="v">' + esc(dur) + '</div>' +
          '<div class="k">Bytes</div><div class="v">' + esc(bytes) + '</div>' +
        '</div>' +
        '<div class="actions">' +
          '<button id="replay">Replay</button>' +
          '<div class="hint" id="replayResult"></div>' +
        '</div>' +
        '<pre>' + esc(toHeadersText(entry.request_headers)) + '</pre>';

      const btn = document.getElementById("replay");
      const out = document.getElementById("replayResult");
      btn.addEventListener("click", async () => {
        btn.disabled = true;
        out.textContent = "Replaying…";
        try {
          const r = await fetch("/api/requests/" + encodeURIComponent(entry.id) + "/replay", { method: "POST" });
          const body = await r.json().catch(() => null);
          if (!r.ok) {
            out.textContent = (body && body.error) ? ("Error: " + body.error) : ("Error: HTTP " + r.status);
            return;
          }
          if (body && body.status_code) out.textContent = "Replayed (upstream status " + body.status_code + ")";
          else out.textContent = "Replayed";
        } catch (e) {
          out.textContent = "Error: " + String(e);
        } finally {
          btn.disabled = false;
        }
      });
    }

    async function refresh() {
      try {
        const r = await fetch("/api/requests", { cache: "no-store" });
        const body = await r.json();
        state.requests = (body && body.requests) ? body.requests : [];
        if (!state.selectedId && state.requests.length > 0) state.selectedId = state.requests[0].id;
        if (state.selectedId && !state.requests.find((x) => x.id === state.selectedId) && state.requests.length > 0) {
          state.selectedId = state.requests[0].id;
        }
        renderList();
        renderDetails();
      } catch (e) {
        state.lastError = String(e);
        renderList();
        renderDetails();
      }
    }

    refresh();
    setInterval(refresh, 1000);
  </script>
</body>
</html>
`
