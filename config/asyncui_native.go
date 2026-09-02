//go:build !js || !wasm

package config

// defaultAsyncUI enables the off-thread UI rasterizer on native builds.
func defaultAsyncUI() bool { return true }
