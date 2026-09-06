package main

import (
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

	game, err := app.New(cfg)
	if err != nil {
		glog.Fatalf("%v", err)
	}

	if err := render.Run(game, cfg.Window, cfg.Render); err != nil {
		glog.Fatalf("%v", err)
	}
}
