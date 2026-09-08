package network

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

func TestMapKeepaliveRunsWithoutGameUpdates(t *testing.T) {
	client := NewClient(20080910, false)
	client.mapKeepaliveInterval = 10 * time.Millisecond
	local, remote := net.Pipe()
	defer remote.Close()

	sendCh := make(chan outboundPacket, sendQueueSize)
	client.mu.Lock()
	client.conn = local
	client.sendCh = sendCh
	client.mu.Unlock()
	go client.writeLoop(local, sendCh)
	defer client.Close()

	enter := BuildMapServerEnterPacketForClientDate(MapServerEnter{
		AccountID:  1,
		CharID:     2,
		AuthCode:   3,
		ClientTick: 4,
		Sex:        1,
	}, client.clientDate)
	if err := client.SendMapServerEnter(1, 2, 3, 4, 1); err != nil {
		t.Fatalf("SendMapServerEnter returned error: %v", err)
	}

	if err := remote.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	gotEnter := make([]byte, len(enter))
	if _, err := io.ReadFull(remote, gotEnter); err != nil {
		t.Fatalf("reading map enter packet: %v", err)
	}
	if !bytes.Equal(gotEnter, enter) {
		t.Fatalf("map enter packet = % x, want % x", gotEnter, enter)
	}

	tickLen := len(BuildTickSendPacketForClientDate(0, client.clientDate))
	tick := make([]byte, tickLen)
	if _, err := io.ReadFull(remote, tick); err != nil {
		t.Fatalf("reading autonomous map keepalive: %v", err)
	}
	wantOpcode := binary.LittleEndian.Uint16(BuildTickSendPacketForClientDate(0, client.clientDate)[:2])
	if got := binary.LittleEndian.Uint16(tick[:2]); got != wantOpcode {
		t.Fatalf("keepalive opcode = 0x%04x, want 0x%04x", got, wantOpcode)
	}
}

func TestMapKeepaliveStopsWithConnection(t *testing.T) {
	client := NewClient(20080910, false)
	client.mapKeepaliveInterval = time.Hour
	local, remote := net.Pipe()
	defer remote.Close()

	sendCh := make(chan outboundPacket, sendQueueSize)
	client.mu.Lock()
	client.conn = local
	client.sendCh = sendCh
	client.mu.Unlock()
	go client.writeLoop(local, sendCh)

	if err := client.SendMapServerEnter(1, 2, 3, 4, 1); err != nil {
		t.Fatalf("SendMapServerEnter returned error: %v", err)
	}
	client.Close()

	client.mu.Lock()
	defer client.mu.Unlock()
	if client.mapKeepaliveStop != nil {
		t.Fatal("map keepalive remained active after connection close")
	}
}
