package login

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
)

func TestLoginRefuseMessage(t *testing.T) {
	if got := loginRefuseMessage(nil, network.AccountRefuseLogin{ErrorCode: 1}); got != "Incorrect Password." {
		t.Fatalf("wrong-password message = %q", got)
	}
	if got := loginRefuseMessage(nil, network.AccountRefuseLogin{
		ErrorCode:   6,
		UnblockTime: "2026-08-29 21:40:00",
	}); got != "You are prohibited from logging in until 2026-08-29 21:40:00." {
		t.Fatalf("temporary-ban message = %q", got)
	}
}

func TestAccountRefusalReturnsToLoginWithoutQuitting(t *testing.T) {
	quit := false
	mode := New(Bindings{})
	mode.loginPending = true
	ctx := client.Context{
		ScreenW: 1280,
		ScreenH: 720,
		RequestQuit: func() {
			quit = true
		},
	}

	mode.applyAccountRefuseLogin(ctx, network.AccountRefuseLogin{ErrorCode: 1})

	if mode.loginPending {
		t.Fatal("login attempt remained pending after refusal")
	}
	if !mode.disconnectDialog.IsOpen() {
		t.Fatal("login refusal did not open an error dialog")
	}
	if mode.status != "Incorrect Password." {
		t.Fatalf("status = %q", mode.status)
	}

	mode.disconnectDialog.Confirm(ctx)
	if quit {
		t.Fatal("acknowledging a login refusal quit the client")
	}
	if mode.status != "enter account credentials" {
		t.Fatalf("status after acknowledgement = %q", mode.status)
	}
}
