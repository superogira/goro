// Command webbridge pipes WebSocket connections to local TCP servers so the
// browser build of goro can reach the game servers (browsers cannot open raw
// TCP sockets). Destinations are restricted to loopback addresses unless
// -allow-private is set, which also permits RFC1918 LAN ranges.
package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

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
		websocket.Handler(pipeConn).ServeHTTP(w, r)
	})

	scope := "loopback destinations only"
	if *allowPrivate {
		scope = "loopback and RFC1918 LAN destinations"
	}
	log.Printf("webbridge listening on ws://%s/connect?addr=host:port (%s)", *addr, scope)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}

func pipeConn(ws *websocket.Conn) {
	target := ws.Request().URL.Query().Get("addr")
	conn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		log.Printf("bridge dial %s: %v", target, err)
		ws.Close()
		return
	}
	defer conn.Close()
	defer ws.Close()

	log.Printf("bridge %s <-> %s", ws.Request().RemoteAddr, target)
	ws.PayloadType = websocket.BinaryFrame
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(conn, ws)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(ws, conn)
		done <- struct{}{}
	}()
	<-done
	log.Printf("bridge closed %s", target)
}
