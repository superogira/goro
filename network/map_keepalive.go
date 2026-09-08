package network

import (
	"errors"
	"net"
	"time"

	"github.com/kivutar/goro/glog"
)

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

	if interval <= 0 {
		interval = defaultMapKeepaliveInterval
	}
	go c.runMapKeepalive(conn, stop, interval)
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
			} else if c.trace {
				glog.Debugf("sent CZ_REQUEST_TIME opcode=0x%04X client_date=%d", ID(packet), c.clientDate)
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
