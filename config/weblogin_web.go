//go:build js && wasm

package config

import (
	"net/url"
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
}
