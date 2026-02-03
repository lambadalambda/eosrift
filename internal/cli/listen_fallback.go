package cli

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"syscall"
)

func listenTCPWithPortFallback(addr string, maxPort int) (net.Listener, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return net.Listen("tcp", addr)
	}

	startPort, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port %q: %w", portStr, err)
	}

	if maxPort <= 0 {
		maxPort = startPort
	}
	if startPort > maxPort {
		return net.Listen("tcp", addr)
	}

	for port := startPort; port <= maxPort; port++ {
		ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err == nil {
			return ln, nil
		}
		if !errors.Is(err, syscall.EADDRINUSE) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("no free port available from %d to %d", startPort, maxPort)
}
