package game

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestStoragePasswordPromptGatesStorageUntilSuccess(t *testing.T) {
	sessionState := &session.Session{
		Storage: session.Storage{Open: true},
	}
	ctx := client.Context{Session: sessionState, ScreenW: 800, ScreenH: 600}
	mode := &WorldMode{}
	promptData := make([]byte, 4)
	binary.LittleEndian.PutUint16(promptData[0:2], network.PacketZCStoragePasswordPrompt)
	binary.LittleEndian.PutUint16(promptData[2:4], uint16(network.StoragePasswordSet))

	if next, stop := mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCStoragePasswordPrompt, Data: promptData}, time.Now()); next != nil || stop {
		t.Fatalf("password prompt changed mode: next=%T stop=%t", next, stop)
	}
	if !sessionState.Storage.PasswordPending || sessionState.Storage.Open {
		t.Fatalf("storage state after prompt = %+v", sessionState.Storage)
	}
	if !mode.ui.storagePassword.IsOpen() {
		t.Fatal("password prompt did not open")
	}

	applyStorageAmount(ctx, network.StorageAmount{Amount: 2, MaxAmount: 300})
	applyStorageItemList(ctx, []network.InventoryItem{{Index: 3, ItemID: 512, Amount: 4}})
	if sessionState.Storage.Open {
		t.Fatal("storage opened while password authentication was pending")
	}
	if sessionState.Storage.Amount != 2 || len(sessionState.Storage.Items) != 1 {
		t.Fatalf("storage data was not retained behind password gate: %+v", sessionState.Storage)
	}

	resultData := make([]byte, 6)
	binary.LittleEndian.PutUint16(resultData[0:2], network.PacketZCStoragePasswordResult)
	binary.LittleEndian.PutUint16(resultData[2:4], uint16(network.StoragePasswordCheckSucceeded))
	if next, stop := mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCStoragePasswordResult, Data: resultData}, time.Now()); next != nil || stop {
		t.Fatalf("password result changed mode: next=%T stop=%t", next, stop)
	}
	if sessionState.Storage.PasswordPending || !sessionState.Storage.Open {
		t.Fatalf("storage state after success = %+v", sessionState.Storage)
	}
	if mode.ui.storagePassword.IsOpen() {
		t.Fatal("password prompt remained open after success")
	}
	if !mode.ui.storageWindow.IsOpen() {
		t.Fatal("storage window did not open after authentication")
	}
}

func TestStoragePasswordFailuresKeepGateClosed(t *testing.T) {
	sessionState := &session.Session{Storage: session.Storage{PasswordPending: true}}
	ctx := client.Context{Session: sessionState, ScreenW: 800, ScreenH: 600}
	mode := &WorldMode{}
	mode.handleStoragePasswordPrompt(ctx, network.StoragePasswordPrompt{State: network.StoragePasswordSet})

	mode.handleStoragePasswordResult(ctx, network.StoragePasswordResult{
		Code:       network.StoragePasswordCheckFailed,
		ErrorCount: 2,
	})
	if !sessionState.Storage.PasswordPending || sessionState.Storage.Open {
		t.Fatalf("failed check released storage gate: %+v", sessionState.Storage)
	}
	if !mode.ui.storagePassword.IsOpen() {
		t.Fatal("password prompt closed after failed check")
	}

	mode.handleStoragePasswordResult(ctx, network.StoragePasswordResult{Code: network.StoragePasswordTooManyFailures})
	if !sessionState.Storage.PasswordPending || sessionState.Storage.Open {
		t.Fatalf("lockout released storage gate: %+v", sessionState.Storage)
	}
}

func TestStoragePasswordChangeRequiresFollowUpCheck(t *testing.T) {
	sessionState := &session.Session{Storage: session.Storage{PasswordPending: true}}
	ctx := client.Context{Session: sessionState, ScreenW: 800, ScreenH: 600}
	mode := &WorldMode{}
	mode.handleStoragePasswordPrompt(ctx, network.StoragePasswordPrompt{State: network.StoragePasswordNotSet})

	mode.handleStoragePasswordResult(ctx, network.StoragePasswordResult{Code: network.StoragePasswordChangeSucceeded})

	if !sessionState.Storage.PasswordPending || sessionState.Storage.Open {
		t.Fatalf("password change bypassed password check: %+v", sessionState.Storage)
	}
	if !mode.ui.storagePassword.IsOpen() {
		t.Fatal("password entry prompt was not shown after password change")
	}
}
