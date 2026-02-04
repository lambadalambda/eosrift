package control

// Control protocol messages are sent over a dedicated yamux stream.

type CreateTCPTunnelRequest struct {
	Type       string `json:"type"` // "tcp"
	Authtoken  string `json:"authtoken,omitempty"`
	RemotePort int    `json:"remote_port"` // 0 = auto-allocate
}

type CreateTCPTunnelResponse struct {
	Type       string `json:"type"`        // "tcp"
	RemotePort int    `json:"remote_port"` // allocated
	Error      string `json:"error,omitempty"`
}

type CreateHTTPTunnelRequest struct {
	Type      string `json:"type"` // "http"
	Authtoken string `json:"authtoken,omitempty"`
	Subdomain string `json:"subdomain,omitempty"`
	Domain    string `json:"domain,omitempty"`

	BasicAuth string `json:"basic_auth,omitempty"` // "user:pass"

	// CIDR-based edge access control. If allow_cidr is non-empty, the server
	// denies any request whose client IP does not match at least one entry.
	// deny_cidr always takes precedence.
	AllowCIDR []string `json:"allow_cidr,omitempty"`
	DenyCIDR  []string `json:"deny_cidr,omitempty"`
}

type CreateHTTPTunnelResponse struct {
	Type string `json:"type"` // "http"

	ID  string `json:"id,omitempty"`
	URL string `json:"url,omitempty"`

	Error string `json:"error,omitempty"`
}
