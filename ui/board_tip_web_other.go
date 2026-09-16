//go:build !js || !wasm

package ui

// Native stubs: no DOM tooltip layer exists.

func BoardTipWebAvailable() bool { return false }

func BoardTipWebSync(text string, x, y int) {}
