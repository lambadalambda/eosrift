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

type HeaderKV struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type CreateHTTPTunnelRequest struct {
	Type      string `json:"type"` // "http"
	Authtoken string `json:"authtoken,omitempty"`
	Subdomain string `json:"subdomain,omitempty"`
	Domain    string `json:"domain,omitempty"`

	BasicAuth string `json:"basic_auth,omitempty"` // "user:pass"

	// Optional allowlist-style filtering on the server edge.
	AllowMethod     []string `json:"allow_method,omitempty"`
	AllowPath       []string `json:"allow_path,omitempty"`
	AllowPathPrefix []string `json:"allow_path_prefix,omitempty"`

	// CIDR-based edge access control. If allow_cidr is non-empty, the server
	// denies any request whose client IP does not match at least one entry.
	// deny_cidr always takes precedence.
	AllowCIDR []string `json:"allow_cidr,omitempty"`
	DenyCIDR  []string `json:"deny_cidr,omitempty"`

	// Header transforms applied at the server edge (before proxying to the
	// client/upstream). These are applied in the following order:
	// - request_header_remove
	// - request_header_add
	//
	// Response transforms are applied after receiving a response from the
	// upstream, before sending it to the public client:
	// - response_header_remove
	// - response_header_add
	RequestHeaderAdd    []HeaderKV `json:"request_header_add,omitempty"`
	RequestHeaderRemove []string   `json:"request_header_remove,omitempty"`
	ResponseHeaderAdd   []HeaderKV `json:"response_header_add,omitempty"`
	ResponseHeaderRemove []string  `json:"response_header_remove,omitempty"`
}

type CreateHTTPTunnelResponse struct {
	Type string `json:"type"` // "http"

	ID  string `json:"id,omitempty"`
	URL string `json:"url,omitempty"`

	Error string `json:"error,omitempty"`
}
