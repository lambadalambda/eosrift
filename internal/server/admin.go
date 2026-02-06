package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxAdminBodyBytes = 64 * 1024

func serveAdminAPI(w http.ResponseWriter, r *http.Request, store AdminStore) {
	if store == nil {
		http.NotFound(w, r)
		return
	}

	resource := strings.TrimPrefix(r.URL.Path, "/api/admin/")
	resource = strings.TrimSpace(resource)
	if resource == "" || resource == "/" {
		http.NotFound(w, r)
		return
	}

	switch {
	case resource == "summary":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		serveAdminSummary(w, r, store)
	case resource == "tokens":
		switch r.Method {
		case http.MethodGet:
			serveAdminListTokens(w, r, store)
		case http.MethodPost:
			serveAdminCreateToken(w, r, store)
		default:
			methodNotAllowed(w)
		}
	case strings.HasPrefix(resource, "tokens/"):
		if r.Method != http.MethodDelete {
			methodNotAllowed(w)
			return
		}
		serveAdminRevokeToken(w, r, store, strings.TrimPrefix(resource, "tokens/"))
	case resource == "subdomains":
		switch r.Method {
		case http.MethodGet:
			serveAdminListSubdomains(w, r, store)
		case http.MethodPost:
			serveAdminReserveSubdomain(w, r, store)
		default:
			methodNotAllowed(w)
		}
	case strings.HasPrefix(resource, "subdomains/"):
		if r.Method != http.MethodDelete {
			methodNotAllowed(w)
			return
		}
		serveAdminUnreserveSubdomain(w, r, store, strings.TrimPrefix(resource, "subdomains/"))
	case resource == "tcp-ports":
		switch r.Method {
		case http.MethodGet:
			serveAdminListTCPPorts(w, r, store)
		case http.MethodPost:
			serveAdminReserveTCPPort(w, r, store)
		default:
			methodNotAllowed(w)
		}
	case strings.HasPrefix(resource, "tcp-ports/"):
		if r.Method != http.MethodDelete {
			methodNotAllowed(w)
			return
		}
		serveAdminUnreserveTCPPort(w, r, store, strings.TrimPrefix(resource, "tcp-ports/"))
	default:
		http.NotFound(w, r)
	}
}

func serveAdminSummary(w http.ResponseWriter, r *http.Request, store AdminStore) {
	tokens, err := store.ListTokens(r.Context())
	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, "failed to list tokens")
		return
	}
	subdomains, err := store.ListReservedSubdomains(r.Context())
	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, "failed to list subdomains")
		return
	}
	ports, err := store.ListReservedTCPPorts(r.Context())
	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, "failed to list tcp ports")
		return
	}

	active := 0
	revoked := 0
	for _, t := range tokens {
		if t.RevokedAt == nil {
			active++
		} else {
			revoked++
		}
	}

	writeAdminJSON(w, http.StatusOK, map[string]any{
		"active_tokens":      active,
		"revoked_tokens":     revoked,
		"reserved_subdomain": len(subdomains),
		"reserved_tcp_ports": len(ports),
		"time":               time.Now().UTC().Format(time.RFC3339),
	})
}

func serveAdminListTokens(w http.ResponseWriter, r *http.Request, store AdminStore) {
	records, err := store.ListTokens(r.Context())
	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, "failed to list tokens")
		return
	}

	items := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		status := "active"
		var revokedAt any
		if rec.RevokedAt != nil {
			status = "revoked"
			revokedAt = rec.RevokedAt.UTC().Format(time.RFC3339)
		}
		items = append(items, map[string]any{
			"id":         rec.ID,
			"label":      rec.Label,
			"prefix":     rec.Prefix,
			"status":     status,
			"created_at": rec.CreatedAt.UTC().Format(time.RFC3339),
			"revoked_at": revokedAt,
		})
	}

	writeAdminJSON(w, http.StatusOK, map[string]any{"tokens": items})
}

func serveAdminCreateToken(w http.ResponseWriter, r *http.Request, store AdminStore) {
	var req struct {
		Label string `json:"label"`
	}
	if err := decodeAdminJSON(r, &req); err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rec, plain, err := store.CreateToken(r.Context(), strings.TrimSpace(req.Label))
	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	writeAdminJSON(w, http.StatusCreated, map[string]any{
		"id":         rec.ID,
		"label":      rec.Label,
		"prefix":     rec.Prefix,
		"created_at": rec.CreatedAt.UTC().Format(time.RFC3339),
		"token":      plain,
	})
}

func serveAdminRevokeToken(w http.ResponseWriter, r *http.Request, store AdminStore, rawID string) {
	id, err := strconv.ParseInt(strings.TrimSpace(rawID), 10, 64)
	if err != nil || id <= 0 {
		writeAdminError(w, http.StatusBadRequest, "invalid token id")
		return
	}
	if err := store.RevokeToken(r.Context(), id); err != nil {
		writeAdminError(w, http.StatusInternalServerError, "failed to revoke token")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func serveAdminListSubdomains(w http.ResponseWriter, r *http.Request, store AdminStore) {
	records, err := store.ListReservedSubdomains(r.Context())
	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, "failed to list subdomains")
		return
	}

	items := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		items = append(items, map[string]any{
			"subdomain":    rec.Subdomain,
			"token_id":     rec.TokenID,
			"token_prefix": rec.TokenPrefix,
			"created_at":   rec.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	writeAdminJSON(w, http.StatusOK, map[string]any{"subdomains": items})
}

func serveAdminReserveSubdomain(w http.ResponseWriter, r *http.Request, store AdminStore) {
	var req struct {
		TokenID   int64  `json:"token_id"`
		Subdomain string `json:"subdomain"`
	}
	if err := decodeAdminJSON(r, &req); err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Subdomain = strings.TrimSpace(req.Subdomain)
	if req.TokenID <= 0 || req.Subdomain == "" {
		writeAdminError(w, http.StatusBadRequest, "token_id and subdomain are required")
		return
	}
	if err := store.ReserveSubdomain(r.Context(), req.TokenID, req.Subdomain); err != nil {
		writeAdminError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeAdminJSON(w, http.StatusCreated, map[string]any{"subdomain": req.Subdomain, "token_id": req.TokenID})
}

func serveAdminUnreserveSubdomain(w http.ResponseWriter, r *http.Request, store AdminStore, raw string) {
	subdomain, err := url.PathUnescape(strings.TrimSpace(raw))
	if err != nil || subdomain == "" {
		writeAdminError(w, http.StatusBadRequest, "invalid subdomain")
		return
	}
	if err := store.UnreserveSubdomain(r.Context(), subdomain); err != nil {
		writeAdminError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func serveAdminListTCPPorts(w http.ResponseWriter, r *http.Request, store AdminStore) {
	records, err := store.ListReservedTCPPorts(r.Context())
	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, "failed to list tcp ports")
		return
	}

	items := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		items = append(items, map[string]any{
			"port":         rec.Port,
			"token_id":     rec.TokenID,
			"token_prefix": rec.TokenPrefix,
			"created_at":   rec.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	writeAdminJSON(w, http.StatusOK, map[string]any{"ports": items})
}

func serveAdminReserveTCPPort(w http.ResponseWriter, r *http.Request, store AdminStore) {
	var req struct {
		TokenID int64 `json:"token_id"`
		Port    int   `json:"port"`
	}
	if err := decodeAdminJSON(r, &req); err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TokenID <= 0 || req.Port <= 0 {
		writeAdminError(w, http.StatusBadRequest, "token_id and port are required")
		return
	}
	if err := store.ReserveTCPPort(r.Context(), req.TokenID, req.Port); err != nil {
		writeAdminError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeAdminJSON(w, http.StatusCreated, map[string]any{"port": req.Port, "token_id": req.TokenID})
}

func serveAdminUnreserveTCPPort(w http.ResponseWriter, r *http.Request, store AdminStore, raw string) {
	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || port <= 0 {
		writeAdminError(w, http.StatusBadRequest, "invalid port")
		return
	}
	if err := store.UnreserveTCPPort(r.Context(), port); err != nil {
		writeAdminError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func serveAdminIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write([]byte(adminIndexHTML))
}

func serveAdminStyle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write([]byte(adminStyleCSS))
}

func serveAdminApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write([]byte(adminAppJS))
}

func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func decodeAdminJSON(r *http.Request, out any) error {
	if r == nil || r.Body == nil {
		return errors.New("missing body")
	}
	defer r.Body.Close()

	dec := json.NewDecoder(io.LimitReader(r.Body, maxAdminBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return errors.New("extra data")
	}
	return nil
}

func writeAdminJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeAdminError(w http.ResponseWriter, status int, msg string) {
	writeAdminJSON(w, status, map[string]string{"error": msg})
}

const adminIndexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Eosrift Admin</title>
  <link rel="stylesheet" href="/admin/style.css">
</head>
<body>
  <main class="layout">
    <header class="hero">
      <h1>Eosrift Admin</h1>
      <p>Manage tokens and reservations for your server.</p>
    </header>

    <section class="panel auth">
      <h2>Admin Token</h2>
      <div class="row">
        <input id="adminToken" type="password" placeholder="Paste EOSRIFT_ADMIN_TOKEN">
        <button id="saveTokenBtn" type="button">Save Token</button>
        <button id="refreshBtn" type="button">Refresh</button>
      </div>
      <p class="hint">Token is stored in this browser only (localStorage).</p>
      <p id="statusLine" class="status"></p>
    </section>

    <section class="stats" id="summaryGrid">
      <article class="stat"><h3>Active Tokens</h3><p id="activeTokens">-</p></article>
      <article class="stat"><h3>Revoked Tokens</h3><p id="revokedTokens">-</p></article>
      <article class="stat"><h3>Subdomain Reservations</h3><p id="subdomainCount">-</p></article>
      <article class="stat"><h3>TCP Port Reservations</h3><p id="tcpCount">-</p></article>
    </section>

    <section class="grid">
      <article class="panel">
        <h2>Authtokens</h2>
        <form id="createTokenForm" class="row">
          <input id="tokenLabel" type="text" placeholder="Label (optional)">
          <button type="submit">Create Token</button>
        </form>
        <pre id="createTokenOut" class="mono hidden"></pre>
        <table>
          <thead><tr><th>ID</th><th>Prefix</th><th>Label</th><th>Status</th><th></th></tr></thead>
          <tbody id="tokensBody"></tbody>
        </table>
      </article>

      <article class="panel">
        <h2>Reserved Subdomains</h2>
        <form id="reserveSubdomainForm" class="row">
          <input id="subdomainTokenId" type="number" min="1" placeholder="Token ID">
          <input id="subdomainName" type="text" placeholder="subdomain">
          <button type="submit">Reserve</button>
        </form>
        <table>
          <thead><tr><th>Subdomain</th><th>Token ID</th><th>Prefix</th><th></th></tr></thead>
          <tbody id="subdomainsBody"></tbody>
        </table>
      </article>

      <article class="panel">
        <h2>Reserved TCP Ports</h2>
        <form id="reservePortForm" class="row">
          <input id="portTokenId" type="number" min="1" placeholder="Token ID">
          <input id="portNumber" type="number" min="1" max="65535" placeholder="Port">
          <button type="submit">Reserve</button>
        </form>
        <table>
          <thead><tr><th>Port</th><th>Token ID</th><th>Prefix</th><th></th></tr></thead>
          <tbody id="portsBody"></tbody>
        </table>
      </article>
    </section>
  </main>
  <script src="/admin/app.js"></script>
</body>
</html>
`

const adminStyleCSS = `:root {
  --bg: #07111a;
  --bg-soft: #0f1d29;
  --bg-card: #132538;
  --border: #27445e;
  --text: #e7f3ff;
  --muted: #8fb1d1;
  --accent: #2cb3ff;
  --danger: #f06464;
  --ok: #4fd38c;
}

* { box-sizing: border-box; }

body {
  margin: 0;
  font-family: "Inter", "Segoe UI", sans-serif;
  background:
    radial-gradient(circle at 15% 10%, #12314a 0%, transparent 45%),
    radial-gradient(circle at 85% 0%, #10283d 0%, transparent 40%),
    var(--bg);
  color: var(--text);
}

.layout {
  max-width: 1280px;
  margin: 0 auto;
  padding: 24px;
}

.hero h1 {
  margin: 0 0 8px;
  font-size: 2rem;
}

.hero p {
  margin: 0 0 18px;
  color: var(--muted);
}

.panel {
  background: linear-gradient(180deg, #173049, var(--bg-card));
  border: 1px solid var(--border);
  border-radius: 14px;
  padding: 14px;
}

.auth {
  margin-bottom: 16px;
}

.row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 10px;
  margin: 0 0 16px;
}

.stat {
  background: rgba(13, 27, 40, 0.7);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 12px;
}

.stat h3 {
  margin: 0;
  font-size: 0.85rem;
  color: var(--muted);
}

.stat p {
  margin: 8px 0 0;
  font-size: 1.35rem;
  font-weight: 700;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
  gap: 12px;
}

h2 {
  margin: 0 0 12px;
  font-size: 1rem;
}

input, button {
  border-radius: 8px;
  border: 1px solid var(--border);
  font-size: 0.95rem;
  padding: 8px 10px;
}

input {
  background: #0f202f;
  color: var(--text);
  flex: 1;
  min-width: 120px;
}

button {
  background: #123c59;
  color: var(--text);
  cursor: pointer;
}

button:hover { filter: brightness(1.1); }

button.danger {
  background: #4b2430;
  border-color: #7a3344;
}

table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 12px;
  font-size: 0.9rem;
}

th, td {
  border-bottom: 1px solid var(--border);
  text-align: left;
  padding: 7px 6px;
}

th { color: var(--muted); font-weight: 600; }

.hint {
  color: var(--muted);
  font-size: 0.85rem;
  margin: 8px 0 0;
}

.status {
  margin: 10px 0 0;
  min-height: 1.2em;
}

.status.ok { color: var(--ok); }
.status.err { color: var(--danger); }

.mono {
  margin: 10px 0 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: #0c1a27;
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 10px;
  overflow: auto;
}

.hidden { display: none; }

@media (max-width: 720px) {
  .layout { padding: 14px; }
  .grid { grid-template-columns: 1fr; }
}
`

const adminAppJS = `(() => {
  const storageKey = "eosrift_admin_token";
  const tokenInput = document.getElementById("adminToken");
  const statusLine = document.getElementById("statusLine");
  const createTokenOut = document.getElementById("createTokenOut");

  const activeTokens = document.getElementById("activeTokens");
  const revokedTokens = document.getElementById("revokedTokens");
  const subdomainCount = document.getElementById("subdomainCount");
  const tcpCount = document.getElementById("tcpCount");

  const tokensBody = document.getElementById("tokensBody");
  const subdomainsBody = document.getElementById("subdomainsBody");
  const portsBody = document.getElementById("portsBody");

  let adminToken = localStorage.getItem(storageKey) || "";
  tokenInput.value = adminToken;

  function setStatus(msg, ok) {
    statusLine.textContent = msg;
    statusLine.className = "status " + (ok ? "ok" : "err");
  }

  async function api(path, opts = {}) {
    const headers = Object.assign({}, opts.headers || {});
    if (adminToken) headers["Authorization"] = "Bearer " + adminToken;
    if (opts.body && !headers["Content-Type"]) {
      headers["Content-Type"] = "application/json";
    }

    const res = await fetch("/api/admin/" + path, Object.assign({}, opts, { headers }));
    if (res.status === 204) return null;
    let body = null;
    try {
      body = await res.json();
    } catch (_) {
      body = null;
    }
    if (!res.ok) {
      const message = body && body.error ? body.error : ("request failed: " + res.status);
      throw new Error(message);
    }
    return body;
  }

  function rowButton(label, cls, onClick) {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.textContent = label;
    if (cls) btn.classList.add(cls);
    btn.addEventListener("click", onClick);
    return btn;
  }

  function clearChildren(el) {
    while (el.firstChild) el.removeChild(el.firstChild);
  }

  async function loadSummary() {
    const data = await api("summary");
    activeTokens.textContent = String(data.active_tokens);
    revokedTokens.textContent = String(data.revoked_tokens);
    subdomainCount.textContent = String(data.reserved_subdomain);
    tcpCount.textContent = String(data.reserved_tcp_ports);
  }

  async function loadTokens() {
    const data = await api("tokens");
    clearChildren(tokensBody);
    data.tokens.forEach((token) => {
      const tr = document.createElement("tr");
      const fields = [token.id, token.prefix, token.label || "-", token.status];
      fields.forEach((value) => {
        const td = document.createElement("td");
        td.textContent = String(value);
        tr.appendChild(td);
      });

      const action = document.createElement("td");
      if (token.status === "active") {
        action.appendChild(rowButton("Revoke", "danger", async () => {
          await api("tokens/" + token.id, { method: "DELETE" });
          await refreshAll();
        }));
      }
      tr.appendChild(action);
      tokensBody.appendChild(tr);
    });
  }

  async function loadSubdomains() {
    const data = await api("subdomains");
    clearChildren(subdomainsBody);
    data.subdomains.forEach((item) => {
      const tr = document.createElement("tr");
      [item.subdomain, item.token_id, item.token_prefix].forEach((value) => {
        const td = document.createElement("td");
        td.textContent = String(value);
        tr.appendChild(td);
      });

      const action = document.createElement("td");
      action.appendChild(rowButton("Remove", "danger", async () => {
        await api("subdomains/" + encodeURIComponent(item.subdomain), { method: "DELETE" });
        await refreshAll();
      }));
      tr.appendChild(action);
      subdomainsBody.appendChild(tr);
    });
  }

  async function loadPorts() {
    const data = await api("tcp-ports");
    clearChildren(portsBody);
    data.ports.forEach((item) => {
      const tr = document.createElement("tr");
      [item.port, item.token_id, item.token_prefix].forEach((value) => {
        const td = document.createElement("td");
        td.textContent = String(value);
        tr.appendChild(td);
      });

      const action = document.createElement("td");
      action.appendChild(rowButton("Remove", "danger", async () => {
        await api("tcp-ports/" + item.port, { method: "DELETE" });
        await refreshAll();
      }));
      tr.appendChild(action);
      portsBody.appendChild(tr);
    });
  }

  async function refreshAll() {
    if (!adminToken) {
      setStatus("Set the admin token to start.", false);
      return;
    }
    try {
      await Promise.all([loadSummary(), loadTokens(), loadSubdomains(), loadPorts()]);
      setStatus("Connected.", true);
    } catch (err) {
      setStatus(err.message, false);
    }
  }

  document.getElementById("saveTokenBtn").addEventListener("click", async () => {
    adminToken = tokenInput.value.trim();
    localStorage.setItem(storageKey, adminToken);
    await refreshAll();
  });

  document.getElementById("refreshBtn").addEventListener("click", async () => {
    await refreshAll();
  });

  document.getElementById("createTokenForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const label = document.getElementById("tokenLabel").value.trim();
      const data = await api("tokens", { method: "POST", body: JSON.stringify({ label }) });
      createTokenOut.classList.remove("hidden");
      createTokenOut.textContent = "id: " + data.id + "\nprefix: " + data.prefix + "\ntoken: " + data.token;
      document.getElementById("tokenLabel").value = "";
      await refreshAll();
    } catch (err) {
      setStatus(err.message, false);
    }
  });

  document.getElementById("reserveSubdomainForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const tokenID = Number(document.getElementById("subdomainTokenId").value);
      const subdomain = document.getElementById("subdomainName").value.trim();
      await api("subdomains", {
        method: "POST",
        body: JSON.stringify({ token_id: tokenID, subdomain })
      });
      document.getElementById("subdomainName").value = "";
      await refreshAll();
    } catch (err) {
      setStatus(err.message, false);
    }
  });

  document.getElementById("reservePortForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const tokenID = Number(document.getElementById("portTokenId").value);
      const port = Number(document.getElementById("portNumber").value);
      await api("tcp-ports", {
        method: "POST",
        body: JSON.stringify({ token_id: tokenID, port })
      });
      document.getElementById("portNumber").value = "";
      await refreshAll();
    } catch (err) {
      setStatus(err.message, false);
    }
  });

  refreshAll();
})();
`
