package npc

import (
	"image/color"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
)

func TestServerProgressLifecycle(t *testing.T) {
	now := time.Unix(100, 0)
	mode := newTestSystem()
	mode.StartProgress(network.ProgressBar{Color: 0x3366CC, Duration: 2 * time.Second}, now)
	if mode.Progress.Color != (color.RGBA{R: 0x33, G: 0x66, B: 0xCC, A: 0xFF}) {
		t.Fatalf("progress color = %#v", mode.Progress.Color)
	}
	if !mode.UpdateProgress(client.Context{}, now.Add(time.Second)) {
		t.Fatal("active progress did not block actions")
	}
	if mode.UpdateProgress(client.Context{}, now.Add(2*time.Second)) {
		t.Fatal("completed progress still blocks actions")
	}
	if !mode.Progress.Started.IsZero() {
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
	mode := newTestSystem()
	mode.StartProgress(network.ProgressBar{Duration: time.Minute}, now)

	if !mode.UpdateProgress(client.Context{Input: state}, now.Add(time.Second)) {
		t.Fatal("cancellation input was not consumed")
	}
	if !mode.Progress.Started.IsZero() {
		t.Fatal("cancellation did not clear progress")
	}
}

func TestServerProgressCompletionSendsAcknowledgement(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	now := time.Unix(100, 0)
	mode := newTestSystem()
	ctx := client.Context{Network: networkClient}
	mode.StartProgress(network.ProgressBar{Duration: time.Second}, now)

	if mode.UpdateProgress(ctx, now.Add(time.Second)) {
		t.Fatal("completed progress still blocks actions")
	}
	testutil.ReadPackets(t, serverConn, network.BuildProgressBarDonePacket())
}
