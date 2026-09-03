package ui

import (
	"testing"

	"github.com/kivutar/goro/client"
)

func TestStoragePasswordWindowChecksExistingPassword(t *testing.T) {
	ctx := client.Context{ScreenW: 800, ScreenH: 600}
	var window StoragePasswordWindow
	window.OpenPrompt(ctx, StoragePasswordModeEnter, "Enter your storage password.")
	window.password = "secret"

	window.submit(ctx)

	action := window.PopAction()
	if action.Kind != StoragePasswordActionCheck || action.Password != "secret" {
		t.Fatalf("action = %+v; want password check", action)
	}
	if !window.waiting {
		t.Fatal("window did not wait for the server result")
	}
}

func TestStoragePasswordWindowSetsAndConfirmsNewPassword(t *testing.T) {
	ctx := client.Context{ScreenW: 800, ScreenH: 600}
	var window StoragePasswordWindow
	window.OpenPrompt(ctx, StoragePasswordModeSet, "Set a storage password.")
	window.password = "secret"
	window.confirmation = "different"

	window.submit(ctx)

	if action := window.PopAction(); action.Kind != StoragePasswordActionNone {
		t.Fatalf("mismatched confirmation produced action %+v", action)
	}
	if window.message != "The passwords do not match." {
		t.Fatalf("message = %q; want mismatch feedback", window.message)
	}

	window.confirmation = "secret"
	window.submit(ctx)
	action := window.PopAction()
	if action.Kind != StoragePasswordActionChange || action.Password != "secret" {
		t.Fatalf("action = %+v; want password change", action)
	}
}

func TestStoragePasswordWindowValidatesLength(t *testing.T) {
	ctx := client.Context{ScreenW: 800, ScreenH: 600}
	var window StoragePasswordWindow
	window.OpenPrompt(ctx, StoragePasswordModeEnter, "")
	window.password = "123"

	window.submit(ctx)

	if action := window.PopAction(); action.Kind != StoragePasswordActionNone {
		t.Fatalf("short password produced action %+v", action)
	}
	if window.message != "Use 4 to 8 characters." {
		t.Fatalf("message = %q; want length feedback", window.message)
	}
}

func TestStoragePasswordWindowCancelProducesAction(t *testing.T) {
	ctx := client.Context{ScreenW: 800, ScreenH: 600}
	var window StoragePasswordWindow
	window.OpenPrompt(ctx, StoragePasswordModeEnter, "")

	window.cancel(ctx)

	if window.IsOpen() {
		t.Fatal("window remained open after cancellation")
	}
	if action := window.PopAction(); action.Kind != StoragePasswordActionCancel {
		t.Fatalf("action = %+v; want cancel", action)
	}
}
