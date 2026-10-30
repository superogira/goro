package npc

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestStoragePasswordFailuresKeepGateClosed(t *testing.T) {
	sessionState := &session.Session{Storage: session.Storage{PasswordPending: true}}
	ctx := client.Context{Session: sessionState, ScreenW: 800, ScreenH: 600}
	mode := newTestSystem()
	mode.HandleStoragePasswordPrompt(ctx, network.StoragePasswordPrompt{State: network.StoragePasswordSet})

	mode.HandleStoragePasswordResult(ctx, network.StoragePasswordResult{
		Code:       network.StoragePasswordCheckFailed,
		ErrorCount: 2,
	})
	if !sessionState.Storage.PasswordPending || sessionState.Storage.Open {
		t.Fatalf("failed check released storage gate: %+v", sessionState.Storage)
	}
	if !mode.bindings.Windows.StoragePassword.IsOpen() {
		t.Fatal("password prompt closed after failed check")
	}

	mode.HandleStoragePasswordResult(ctx, network.StoragePasswordResult{Code: network.StoragePasswordTooManyFailures})
	if !sessionState.Storage.PasswordPending || sessionState.Storage.Open {
		t.Fatalf("lockout released storage gate: %+v", sessionState.Storage)
	}
}

func TestStoragePasswordChangeRequiresFollowUpCheck(t *testing.T) {
	sessionState := &session.Session{Storage: session.Storage{PasswordPending: true}}
	ctx := client.Context{Session: sessionState, ScreenW: 800, ScreenH: 600}
	mode := newTestSystem()
	mode.HandleStoragePasswordPrompt(ctx, network.StoragePasswordPrompt{State: network.StoragePasswordNotSet})

	mode.HandleStoragePasswordResult(ctx, network.StoragePasswordResult{Code: network.StoragePasswordChangeSucceeded})

	if !sessionState.Storage.PasswordPending || sessionState.Storage.Open {
		t.Fatalf("password change bypassed password check: %+v", sessionState.Storage)
	}
	if !mode.bindings.Windows.StoragePassword.IsOpen() {
		t.Fatal("password entry prompt was not shown after password change")
	}
}
