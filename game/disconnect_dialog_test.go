package game

import (
	"errors"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
)

func TestDisconnectMessageForBanCode15UsesNeutralMessage(t *testing.T) {
	got := disconnectMessageForBanCode(nil, 15)
	want := "Disconnected from Server!"
	if got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}

func TestDisconnectMessageForUnknownBanCodeUsesGeneric(t *testing.T) {
	got := disconnectMessageForBanCode(nil, 250)
	if got != "Disconnected from Server!" {
		t.Fatalf("message = %q, want %q", got, "Disconnected from Server!")
	}
}

func TestConnectionFailedMessageFallback(t *testing.T) {
	got := disconnectMessageText(nil, disconnectMessage{1, "Failed to Connect to Server."})
	if got != "Failed to Connect to Server." {
		t.Fatalf("message = %q, want %q", got, "Failed to Connect to Server.")
	}
}

func TestHandleDisconnectPacketOpensAlertAndOKQuits(t *testing.T) {
	var modal gameui.ConfirmModal
	quit := false
	ctx := client.Context{
		ScreenW:     800,
		ScreenH:     600,
		RequestQuit: func() { quit = true },
	}
	packet := network.Packet{
		ID:   network.PacketSCNotifyBan,
		Data: []byte{0x81, 0x00, 15},
	}

	if !handleDisconnectPacket(ctx, &modal, packet, nil) {
		t.Fatal("disconnect packet was not handled")
	}
	if !modal.IsOpen() {
		t.Fatal("disconnect dialog did not open")
	}

	modal.Confirm(ctx)
	if !quit {
		t.Fatal("OK did not request client quit")
	}
}

func TestHandleNetworkDisconnectErrorsIgnoresFrameErrors(t *testing.T) {
	var modal gameui.ConfirmModal
	handled := handleNetworkDisconnectErrors(client.Context{}, &modal, []error{
		network.FrameError{Err: errors.New("bad frame")},
	}, nil)
	if handled {
		t.Fatal("frame error opened disconnect dialog")
	}
	if modal.IsOpen() {
		t.Fatal("disconnect dialog opened for frame error")
	}
}

func TestHandleNetworkDisconnectErrorsOpensAlert(t *testing.T) {
	var modal gameui.ConfirmModal
	handled := handleNetworkDisconnectErrors(client.Context{ScreenW: 800, ScreenH: 600}, &modal, []error{
		network.ErrDisconnected,
	}, nil)
	if !handled {
		t.Fatal("disconnect error was not handled")
	}
	if !modal.IsOpen() {
		t.Fatal("disconnect dialog did not open")
	}
}
