//go:build !js || !wasm

package network

import (
	"context"
	"net"
	"time"
)

// dialGameServer connects to a game server over raw TCP. Hostnames resolve
// through the preferred DNS server first (see dns.go).
func dialGameServer(ctx context.Context, address string, port int) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 5 * time.Second}
	return dialer.DialContext(ctx, "tcp", resolveDialTarget(ctx, address, port))
}
