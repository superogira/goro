//go:build !js || !wasm

package network

import (
	"context"
	"fmt"
	"net"
	"time"
)

// dialGameServer connects to a game server over raw TCP.
func dialGameServer(ctx context.Context, address string, port int) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 5 * time.Second}
	return dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", address, port))
}
