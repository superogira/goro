//go:build js && wasm

package config

import (
	"net/url"
	"strconv"
	"strings"
	"syscall/js"
)

// applyWebLoginQuery seeds login credentials from the page URL so a browser
// session can auto-login: index.html?u=test&p=test&auto=1
// mute=1 disables audio: oto's WASM driver blocks startup until the browser
// autoplay policy grants a user gesture, which automated sessions cannot
// provide reliably.
func applyWebLoginQuery(cfg *Config) {
	search := js.Global().Get("location").Get("search").String()
	query, err := url.ParseQuery(strings.TrimPrefix(search, "?"))
	if err != nil {
		return
	}
	if value := query.Get("u"); value != "" {
		cfg.Login.Username = value
	}
	if value := query.Get("p"); value != "" {
		cfg.Login.Password = value
	}
	if query.Get("auto") == "1" {
		cfg.Login.AutoLogin = true
	}
	// char-slot picks the character automatically after login (0-based);
	// combined with auto=1 the session runs from title screen into the map
	// with no taps — useful on tablets and for automated checks.
	if value := query.Get("char-slot"); value != "" {
		if n, err := strconv.Atoi(value); err == nil {
			cfg.Login.CharSlot = n
		}
	}
	if query.Get("mute") == "1" {
		cfg.Audio.Disabled = true
	}
	if query.Get("stats") == "1" {
		cfg.Render.Stats = true
	}
	if query.Get("nobg") == "1" {
		cfg.Login.DebugNoBackground = true
	}
	if query.Get("nowin") == "1" {
		cfg.Login.DebugNoWindow = true
	}
	if query.Get("nocursor") == "1" {
		cfg.Login.DebugNoCursor = true
	}
	if query.Get("noui") == "1" {
		cfg.Login.DebugNoUI = true
	}
	// fps=1 shows the engine's own FPS counter, which is drawn straight to
	// the frame and therefore stays visible with noui=1 (the widget FPS HUD
	// is part of the suppressed UI layer).
	if query.Get("fps") == "1" {
		cfg.Render.FPS = true
	}
}
