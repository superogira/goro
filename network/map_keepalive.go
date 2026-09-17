package network

import (
	"errors"
	"net"
	"time"

	"github.com/kivutar/goro/glog"
)

// Heartbeat replies arrive even when the player is idle. Many servers also
// push their own ZC_NOTIFY_TIME every ~30s, so the deadline must comfortably
// exceed both the reply interval and a server-driven ping interval before a
// silent map connection is treated as disconnected.
const mapReadTimeout = 35 * time.Second

// startMapKeepalive ties the map-server heartbeat to the connection rather
// than the game or render loop. A reconnect stops the old loop before the new
// connection can receive any of its packets.
func (c *Client) startMapKeepalive(conn net.Conn) {
	c.mu.Lock()
	if conn == nil || c.conn != conn {
		c.mu.Unlock()
		return
	}
	c.stopMapKeepaliveLocked()
	stop := make(chan struct{})
	c.mapKeepaliveStop = stop
	interval := c.mapKeepaliveInterval
	c.mu.Unlock()

	if err := c.refreshMapReadDeadline(conn); err != nil {
		c.clearConn(conn, err)
		return
	}
	if interval <= 0 {
		interval = defaultMapKeepaliveInterval
	}
	go c.runMapKeepalive(conn, stop, interval)
}

// Only map connections expect regular replies. Refresh on receipt, never on
// send: successful writes can merely be queued in TCP while the route is dead.
func (c *Client) refreshMapReadDeadline(conn net.Conn) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != conn || c.mapKeepaliveStop == nil {
		return nil
	}
	return conn.SetReadDeadline(time.Now().Add(mapReadTimeout))
}

func (c *Client) runMapKeepalive(conn net.Conn, stop <-chan struct{}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			packet := BuildTickSendPacketForClientDate(uint32(time.Now().UnixMilli()), c.clientDate)
			if _, err := c.enqueue(packet, conn); err != nil {
				if errors.Is(err, ErrDisconnected) {
					return
				}
				glog.Warnf("map keepalive failed opcode=0x%04X len=%d client_date=%d: %v", ID(packet), len(packet), c.clientDate, err)
			} else {
				// Always log the tick (info, not trace): field reports of
				// "enters map then disconnects ~20s later" need to prove
				// which opcode went out and that the server never answered.
				glog.Infof("sent map tick opcode=0x%04X len=%d client_date=%d", ID(packet), len(packet), c.clientDate)
			}
		case <-stop:
			return
		}
	}
}

// stopMapKeepaliveLocked must be called with c.mu held.
func (c *Client) stopMapKeepaliveLocked() {
	if c.mapKeepaliveStop == nil {
		return
	}
	close(c.mapKeepaliveStop)
	c.mapKeepaliveStop = nil
}
