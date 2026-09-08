//go:build !js || !wasm

package ui

// ConsoleMessage is defined in console.go; the native build keeps the whole
// chat log in the canvas, where a synchronous raster has no per-message
// network fetch to hide and desktop CPUs absorb the cost.

// consoleWebLogEnabled reports whether the dormant chat log renders as page
// DOM instead of canvas widgets. Native builds always use the canvas.
func consoleWebLogEnabled() bool { return false }

// consoleWebSync is the native no-op twin of the web build's DOM bridge.
func consoleWebSync(active bool, lines []ConsoleMessage) {}
