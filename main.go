package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kivutar/goro/app"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/render"

	goui "github.com/gogpu/ui"
	"log/slog"
)

func main() {
	cfg, err := config.LoadConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	closeLog, err := glog.Configure(cfg.Log)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer closeLog()
	// The UI toolkit logs frame-level detail at Info (layout triggers per
	// window open); only warnings and above are worth the console on web.
	goui.SetLogger(slog.New(slog.NewTextHandler(glog.Writer(), &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})))

	// Boot-time self update: fetch version.txt from the update server and,
	// when it advertises a newer build, install it and re-exec — running
	// next to the game loop so the on-screen progress overlay renders.
	// Failures are non-fatal; the game boots with whatever it has.
	if cfg.Update.Enabled && !cfg.Headless && cfg.Update.BaseURL != "" {
		app.StartSelfUpdate(cfg.Update.BaseURL)
	}

	game, err := app.New(cfg)
	if err != nil {
		glog.Fatalf("%v", err)
	}

	if cfg.Headless {
		err = render.RunHeadless(context.Background(), game, cfg.Window)
	} else {
		err = render.Run(game, cfg.Window, cfg.Render)
	}
	if err != nil {
		glog.Fatalf("%v", err)
	}
}
