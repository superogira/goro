//go:build !js || !wasm

package config

// applyWebLoginQuery is a no-op outside the browser.
func applyWebLoginQuery(cfg *Config) {}
