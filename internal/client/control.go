package client

import (
	"context"

	"github.com/hashicorp/yamux"
	"nhooyr.io/websocket"
)

func dialControl(ctx context.Context, controlURL string) (*websocket.Conn, *yamux.Session, error) {
	ws, _, err := websocket.Dial(ctx, controlURL, &websocket.DialOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		return nil, nil, err
	}

	netConn := websocket.NetConn(ctx, ws, websocket.MessageBinary)

	session, err := yamux.Client(netConn, nil)
	if err != nil {
		_ = ws.Close(websocket.StatusInternalError, "yamux error")
		return nil, nil, err
	}

	return ws, session, nil
}

