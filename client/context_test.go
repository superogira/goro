package client_test

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
)

func TestContextScreenSizeUsesLayoutSizeWhenAvailable(t *testing.T) {
	ctx := client.Context{
		Config:  config.Config{Window: config.WindowConfig{Width: 1024, Height: 768}},
		ScreenW: 1280,
		ScreenH: 720,
	}

	width, height := ctx.ScreenSize()
	if width != 1280 || height != 720 {
		t.Fatalf("screen size = %dx%d, want 1280x720", width, height)
	}
}

func TestContextScreenSizeFallsBackToConfig(t *testing.T) {
	ctx := client.Context{
		Config: config.Config{Window: config.WindowConfig{Width: 1024, Height: 768}},
	}

	width, height := ctx.ScreenSize()
	if width != 1024 || height != 768 {
		t.Fatalf("screen size = %dx%d, want 1024x768", width, height)
	}
}
