// Command webbridge pipes WebSocket connections to local TCP servers so the
// browser build of goro can reach the game servers (browsers cannot open raw
// TCP sockets). Destinations are restricted to loopback addresses unless
// -allow-private is set, which also permits RFC1918 LAN ranges.
//
// The bridge pings every WebSocket on a short interval: reverse proxies in
// front of the bridge (nginx Proxy Manager & friends) cut WebSockets that
// stay silent for their proxy_read_timeout (60s by default), which players
// experienced as periodic disconnects whenever the game went quiet. Browsers
// answer pings automatically, so this needs no client cooperation.
package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// keepaliveInterval must stay well below the common 60s proxy_read_timeout.
const keepaliveInterval = 25 * time.Second

var upgrader = websocket.Upgrader{
	ReadBufferSize:  32 << 10,
	WriteBufferSize: 32 << 10,
	// Same-origin policy, matching the previous x/net/websocket behavior:
	// the game page is served from the same host as the bridge. Requests
	// without an Origin header (curl, health checks) are allowed through.
	// Only hostnames are compared: reverse proxies commonly strip the port
	// from the forwarded Host header.
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		originHost, _, err := net.SplitHostPort(u.Host)
		if err != nil {
			originHost = u.Host
		}
		reqHost, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			reqHost = r.Host
		}
		return strings.EqualFold(originHost, reqHost)
	},
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8124", "listen address")
	allowPrivate := flag.Bool("allow-private", false, "also permit RFC1918 LAN destinations (loopback is always allowed)")
	flag.Parse()

	destinationAllowed := func(ip net.IP) bool {
		return ip.IsLoopback() || (*allowPrivate && ip.IsPrivate())
	}

	http.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		target := r.URL.Query().Get("addr")
		host, _, err := net.SplitHostPort(target)
		if err != nil {
			http.Error(w, "invalid addr", http.StatusBadRequest)
			return
		}
		ip := net.ParseIP(host)
		if ip == nil || !destinationAllowed(ip) {
			http.Error(w, "destination must be a loopback address", http.StatusForbidden)
			return
		}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade %s: %v", r.RemoteAddr, err)
			return
		}
		pipeConn(ws, target)
	})

	scope := "loopback destinations only"
	if *allowPrivate {
		scope = "loopback and RFC1918 LAN destinations"
	}
	log.Printf("webbridge listening on ws://%s/connect?addr=host:port (%s, keepalive %s)", *addr, scope, keepaliveInterval)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}

// wsNetConn adapts a gorilla WebSocket into a byte pipe for io.Copy. Each
// WebSocket message becomes one or more Read results; control frames (the
// browser's pongs included) are consumed by the library and never surface.
type wsNetConn struct {
	ws      *websocket.Conn
	pending []byte // unread tail of the current message (single Read caller)
	msgs    chan []byte
	errCh   chan error
	done    chan struct{}
	closeOnce sync.Once
}

func newWSNetConn(ws *websocket.Conn) *wsNetConn {
	c := &wsNetConn{
		ws:    ws,
		msgs:  make(chan []byte, 64),
		errCh: make(chan error, 1),
		done:  make(chan struct{}),
	}
	go c.readLoop()
	return c
}

// shutdown closes done exactly once: the read loop's error path and the
// external Close/keepalive failure path race for it.
func (c *wsNetConn) shutdown() {
	c.closeOnce.Do(func() { close(c.done) })
}

func (c *wsNetConn) readLoop() {
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			select {
			case c.errCh <- err:
			default:
			}
			c.shutdown()
			return
		}
		select {
		case c.msgs <- data:
		case <-c.done:
			return
		}
	}
}

func (c *wsNetConn) Read(b []byte) (int, error) {
	for len(c.pending) == 0 {
		select {
		case err := <-c.errCh:
			return 0, err
		case data := <-c.msgs:
			c.pending = data
		case <-c.done:
			return 0, io.EOF
		}
	}
	n := copy(b, c.pending)
	c.pending = c.pending[n:]
	return n, nil
}

func (c *wsNetConn) Write(b []byte) (int, error) {
	if err := c.ws.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *wsNetConn) Close() error {
	c.shutdown()
	return c.ws.Close()
}

func pipeConn(ws *websocket.Conn, target string) {
	conn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		log.Printf("bridge dial %s: %v", target, err)
		ws.Close()
		return
	}
	defer conn.Close()

	log.Printf("bridge %s <-> %s", ws.RemoteAddr(), target)
	defer log.Printf("bridge closed %s", target)

	wsc := newWSNetConn(ws)
	defer wsc.Close()

	// Keepalive pings: they keep reverse proxies from cutting the WebSocket
	// during quiet stretches, and a failed ping write detects dead links.
	// WriteControl is safe to call concurrently with the data pumps. There
	// is deliberately no read deadline: the game can legitimately stay
	// silent for minutes, and the pings alone prove liveness.
	go func() {
		ticker := time.NewTicker(keepaliveInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
					wsc.Close()
					return
				}
			case <-wsc.done:
				return
			}
		}
	}()

	done := make(chan struct{}, 2)
	go func() {
		io.Copy(conn, wsc)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(wsc, conn)
		done <- struct{}{}
	}()
	<-done
}
