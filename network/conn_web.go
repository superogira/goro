//go:build js && wasm

package network

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"syscall/js"
	"time"
)

// webSocketDefaultBaseURL is the bridge endpoint on the page origin. nginx
// proxies /wsro/ to the webbridge process; destinations are loopback-only
// there.
const webSocketDefaultBaseURL = "/wsro/connect"

// webSocketBaseURL returns the bridge endpoint the game connects to.
//
// Resolution order:
//  1. ?ws=... page URL parameter (absolute ws:// or wss:// URL, or a path)
//  2. window.goroWSBase defined by the host page
//  3. same-origin relative path (nginx proxies /wsro/ to the bridge)
//
// Deployments that serve the page over HTTPS must use a wss:// endpoint or
// the same-origin path behind a TLS proxy — browsers block ws:// from
// secure pages (mixed content).
func webSocketBaseURL() string {
	if value := js.Global().Get("location").Get("search").String(); value != "" {
		if query, err := url.ParseQuery(strings.TrimPrefix(value, "?")); err == nil {
			if override := strings.TrimSpace(query.Get("ws")); override != "" {
				return override
			}
		}
	}
	if base := js.Global().Get("goroWSBase"); base.Type() == js.TypeString && base.String() != "" {
		return base.String()
	}
	return webSocketDefaultBaseURL
}

// dialGameServer connects to a game server through the WebSocket bridge.
func dialGameServer(_ context.Context, address string, port int) (net.Conn, error) {
	wsURL := fmt.Sprintf("%s?addr=%s", webSocketBaseURL(), encodeAddr(address, port))
	conn, err := newWebSocketConn(wsURL)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func encodeAddr(address string, port int) string {
	host := address
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return net.JoinHostPort(host, fmt.Sprintf("%d", port))
}

// wsAddr is a placeholder net.Addr for bridged connections.
type wsAddr struct{}

func (wsAddr) Network() string { return "ws" }
func (wsAddr) String() string  { return "websocket-bridge" }

// webSocketConn implements net.Conn over the browser WebSocket API.
// Incoming frames are buffered; Read polls the buffer so the wasm runtime
// keeps servicing the JS event loop (same pattern as the HTTP fetch bridge).
type webSocketConn struct {
	ws     js.Value
	closed bool

	mu       sync.Mutex
	inbox    [][]byte
	notify   chan struct{}
	readDead time.Time
}

func newWebSocketConn(url string) (*webSocketConn, error) {
	socket := js.Global().Get("WebSocket").New(url)
	socket.Set("binaryType", "arraybuffer")

	conn := &webSocketConn{
		ws:     socket,
		notify: make(chan struct{}, 1),
	}

	opened := make(chan error, 1)
	onOpen := js.FuncOf(func(this js.Value, args []js.Value) any {
		opened <- nil
		return nil
	})
	onError := js.FuncOf(func(this js.Value, args []js.Value) any {
		opened <- fmt.Errorf("websocket error")
		return nil
	})
	socket.Call("addEventListener", "open", onOpen)
	socket.Call("addEventListener", "error", onError)
	defer onOpen.Release()
	defer onError.Release()

	deadline := time.Now().Add(10 * time.Second)
	for {
		select {
		case err := <-opened:
			if err != nil {
				return nil, err
			}
			conn.attachHandlers()
			return conn, nil
		default:
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("websocket connect %s: timed out", url)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func (c *webSocketConn) attachHandlers() {
	onMessage := js.FuncOf(func(this js.Value, args []js.Value) any {
		payload := args[0].Get("data")
		var data []byte
		if payload.Type() == js.TypeString {
			// Text frames arrive as strings; treat each code unit as a byte.
			data = []byte(payload.String())
		} else {
			size := payload.Get("byteLength").Int()
			data = make([]byte, size)
			js.CopyBytesToGo(data, js.Global().Get("Uint8Array").New(payload))
		}

		c.mu.Lock()
		c.inbox = append(c.inbox, data)
		c.mu.Unlock()
		select {
		case c.notify <- struct{}{}:
		default:
		}
		return nil
	})
	onClose := js.FuncOf(func(this js.Value, args []js.Value) any {
		c.mu.Lock()
		c.closed = true
		c.mu.Unlock()
		select {
		case c.notify <- struct{}{}:
		default:
		}
		return nil
	})
	c.ws.Call("addEventListener", "message", onMessage)
	c.ws.Call("addEventListener", "close", onClose)
	// Handlers are released together with the socket in Close.
	c.ws.Set("_goroHandlers", js.ValueOf([]any{onMessage, onClose}))
}

func (c *webSocketConn) Read(b []byte) (int, error) {
	for {
		c.mu.Lock()
		if len(c.inbox) > 0 {
			frame := c.inbox[0]
			n := copy(b, frame)
			if n < len(frame) {
				c.inbox[0] = frame[n:]
			} else {
				c.inbox = c.inbox[1:]
			}
			c.mu.Unlock()
			return n, nil
		}
		closed := c.closed
		readDeadline := c.readDead
		c.mu.Unlock()

		if closed {
			return 0, fmt.Errorf("websocket closed")
		}
		if !readDeadline.IsZero() && time.Now().After(readDeadline) {
			return 0, fmt.Errorf("websocket read deadline exceeded")
		}

		select {
		case <-c.notify:
		default:
			time.Sleep(2 * time.Millisecond)
		}
	}
}

func (c *webSocketConn) Write(b []byte) (int, error) {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return 0, fmt.Errorf("websocket closed")
	}
	view := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(view, b)
	c.ws.Call("send", view)
	return len(b), nil
}

func (c *webSocketConn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()
	c.ws.Call("close")
	return nil
}

func (c *webSocketConn) LocalAddr() net.Addr                { return wsAddr{} }
func (c *webSocketConn) RemoteAddr() net.Addr               { return wsAddr{} }
func (c *webSocketConn) SetDeadline(t time.Time) error      { return c.SetReadDeadline(t) }
func (c *webSocketConn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	c.readDead = t
	c.mu.Unlock()
	select {
	case c.notify <- struct{}{}:
	default:
	}
	return nil
}
func (c *webSocketConn) SetWriteDeadline(time.Time) error { return nil }
