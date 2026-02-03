package control

// Control protocol messages are sent over a dedicated yamux stream.

type CreateTCPTunnelRequest struct {
	Type       string `json:"type"`        // "tcp"
	RemotePort int    `json:"remote_port"` // 0 = auto-allocate
}

type CreateTCPTunnelResponse struct {
	Type       string `json:"type"`        // "tcp"
	RemotePort int    `json:"remote_port"` // allocated
	Error      string `json:"error,omitempty"`
}

