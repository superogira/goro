package network

import (
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"testing/synctest"
	"time"
)

// Attach an in-memory transport so synctest can exercise production timeout
// durations without wall-clock sleeps or game/render updates.
func attachMapTimeoutTestConnection(client *Client) (net.Conn, net.Conn) {
	local, remote := net.Pipe()
	sendCh := make(chan outboundPacket, sendQueueSize)
	client.mu.Lock()
	client.conn = local
	client.sendCh = sendCh
	client.mu.Unlock()
	go client.readLoop(local)
	go client.writeLoop(local, sendCh)
	return local, remote
}

func TestMapConnectionTimesOutWithoutReplies(t *testing.T) {
	for _, blockedWrites := range []bool{false, true} {
		name := "writes accepted but no replies"
		if blockedWrites {
			name = "writes also blocked"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				client := NewClient(20080910, false)
				_, remote := attachMapTimeoutTestConnection(client)
				defer remote.Close()
				defer client.Close()
				if !blockedWrites {
					go io.Copy(io.Discard, remote)
				}
				if err := client.SendMapServerEnter(1, 2, 3, 4, 1); err != nil {
					t.Fatal(err)
				}
				synctest.Wait()
				time.Sleep(mapReadTimeout - time.Second)
				synctest.Wait()
				if errs := client.DrainErrors(); len(errs) != 0 {
					t.Fatalf("disconnected before timeout: %v", errs)
				}
				time.Sleep(time.Second)
				synctest.Wait()

				errs := client.DrainErrors()
				if len(errs) != 1 || !errors.Is(errs[0], os.ErrDeadlineExceeded) {
					t.Fatalf("errors = %v, want exactly one read timeout", errs)
				}
				if client.Status() != "offline" {
					t.Fatalf("status = %q, want offline", client.Status())
				}
				if client.mapKeepaliveStop != nil || client.sendCh != nil {
					t.Fatal("timed-out connection retained its keepalive or send queue")
				}
				if err := client.SendWalkToXY(10, 20); err == nil {
					t.Fatal("timed-out connection still accepted movement packets")
				}
			})
		})
	}
}

func TestMapConnectionIncomingDataExtendsTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := NewClient(20080910, false)
		_, remote := attachMapTimeoutTestConnection(client)
		defer remote.Close()
		defer client.Close()
		go io.Copy(io.Discard, remote)
		if err := client.SendMapServerEnter(1, 2, 3, 4, 1); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()

		// Even fragmented server traffic refreshes liveness. No game update
		// is needed to process heartbeats or keep an idle connection alive.
		tickReply := []byte{0x7f, 0x00, 1, 0, 0, 0}
		for i := 0; i < 4; i++ {
			for _, fragment := range [][]byte{tickReply[:2], tickReply[2:]} {
				time.Sleep(mapReadTimeout / 2)
				if _, err := remote.Write(fragment); err != nil {
					t.Fatalf("healthy connection closed: %v", err)
				}
				synctest.Wait()
			}
		}
		if errs := client.DrainErrors(); len(errs) != 0 {
			t.Fatalf("live connection reported errors: %v", errs)
		}
		if packets := client.DrainPackets(); len(packets) != 4 {
			t.Fatalf("received %d packets, want 4 heartbeat replies", len(packets))
		}

		// Once replies stop, outgoing keepalives must not extend the deadline.
		time.Sleep(mapReadTimeout + time.Second)
		synctest.Wait()
		errs := client.DrainErrors()
		if len(errs) != 1 || !errors.Is(errs[0], os.ErrDeadlineExceeded) {
			t.Fatalf("errors after replies stopped = %v, want read timeout", errs)
		}
	})
}

func TestMapTimeoutIsScopedToItsConnection(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := NewClient(20080910, false)
		old, remote := attachMapTimeoutTestConnection(client)
		defer remote.Close()
		defer client.Close()
		go io.Copy(io.Discard, remote)

		// Login/character selection can legitimately remain quiet.
		time.Sleep(2 * mapReadTimeout)
		synctest.Wait()
		if errs := client.DrainErrors(); len(errs) != 0 {
			t.Fatalf("non-map connection timed out: %v", errs)
		}
		if err := client.SendMapServerEnter(1, 2, 3, 4, 1); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		time.Sleep(mapReadTimeout / 2)
		client.Close()
		synctest.Wait()

		current, replacement := attachMapTimeoutTestConnection(client)
		defer replacement.Close()
		go io.Copy(io.Discard, replacement)
		// An old read timeout arriving after a reconnect must neither close
		// the new connection nor report a disconnect to the UI.
		client.clearConn(old, os.ErrDeadlineExceeded)
		if err := client.refreshMapReadDeadline(old); err != nil {
			t.Fatalf("old reader affected replacement: %v", err)
		}
		time.Sleep(2 * mapReadTimeout)
		synctest.Wait()
		if client.conn != current {
			t.Fatal("replacement connection was closed by old timeout")
		}
		if errs := client.DrainErrors(); len(errs) != 0 {
			t.Fatalf("intentional close/reconnect produced errors: %v", errs)
		}

		// A new map connection must acquire its own deadline too.
		if err := client.SendMapServerEnter(1, 2, 3, 4, 1); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		time.Sleep(mapReadTimeout + time.Second)
		synctest.Wait()
		if errs := client.DrainErrors(); len(errs) != 1 || !errors.Is(errs[0], os.ErrDeadlineExceeded) {
			t.Fatalf("replacement map connection errors = %v, want read timeout", errs)
		}
	})
}
