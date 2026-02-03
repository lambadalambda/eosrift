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
}

type CreateHTTPTunnelResponse struct {
	Type string `json:"type"` // "http"

	ID  string `json:"id,omitempty"`
	URL string `json:"url,omitempty"`

	Error string `json:"error,omitempty"`
}
