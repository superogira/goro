package game

import (
	"image/color"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
)

func TestServerProgressLifecycle(t *testing.T) {
	now := time.Unix(100, 0)
	mode := NewWorldMode()
	mode.startServerProgress(client.Context{}, network.ProgressBar{Color: 0x3366CC, Duration: 2 * time.Second}, now)
	if mode.serverProgress.color != (color.RGBA{R: 0x33, G: 0x66, B: 0xCC, A: 0xFF}) {
		t.Fatalf("progress color = %#v", mode.serverProgress.color)
	}
	if !mode.updateServerProgress(client.Context{}, now.Add(time.Second)) {
		t.Fatal("active progress did not block actions")
	}
	if mode.updateServerProgress(client.Context{}, now.Add(2*time.Second)) {
		t.Fatal("completed progress still blocks actions")
	}
	if !mode.serverProgress.started.IsZero() {
		t.Fatal("completed progress was not cleared")
	}
}

func TestBlackServerProgressUsesDefaultGreen(t *testing.T) {
	if got := progressBarColor(0); got != (color.RGBA{G: 255, A: 255}) {
		t.Fatalf("default progress color = %#v", got)
	}
}

func TestServerProgressCancellationConsumesInput(t *testing.T) {
	now := time.Unix(100, 0)
	state := input.NewState()
	state.SetMouseButton(input.MouseButtonLeft, true)
	mode := NewWorldMode()
	mode.startServerProgress(client.Context{}, network.ProgressBar{Duration: time.Minute}, now)

	if !mode.updateServerProgress(client.Context{Input: state}, now.Add(time.Second)) {
		t.Fatal("cancellation input was not consumed")
	}
	if !mode.serverProgress.started.IsZero() {
		t.Fatal("cancellation did not clear progress")
	}
}

func TestServerProgressCompletionSendsAcknowledgement(t *testing.T) {
	networkClient, serverConn := newBotTestConnection(t, 20080910)
	now := time.Unix(100, 0)
	mode := NewWorldMode()
	ctx := client.Context{Network: networkClient}
	mode.startServerProgress(ctx, network.ProgressBar{Duration: time.Second}, now)

	if mode.updateServerProgress(ctx, now.Add(time.Second)) {
		t.Fatal("completed progress still blocks actions")
	}
	readBotTestPackets(t, serverConn, network.BuildProgressBarDonePacket())
}

func TestServerProgressCancellationSendsAcknowledgement(t *testing.T) {
	networkClient, serverConn := newBotTestConnection(t, 20080910)
	now := time.Unix(100, 0)
	mode := NewWorldMode()
	ctx := client.Context{Network: networkClient}
	mode.startServerProgress(ctx, network.ProgressBar{Duration: time.Minute}, now)

	packet := network.Packet{
		ID:   network.PacketZCProgressCancel,
		Data: []byte{0xF2, 0x02},
	}
	if next, stop := mode.handleNetworkPacket(ctx, packet, now.Add(time.Second)); next != nil || stop {
		t.Fatalf("progress cancellation changed mode: next=%T stop=%t", next, stop)
	}
	if !mode.serverProgress.started.IsZero() {
		t.Fatal("server cancellation did not clear progress")
	}
	readBotTestPackets(t, serverConn, network.BuildProgressBarDonePacket())
}
