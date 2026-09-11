//go:build !js || !wasm

package render

// webVersionBadgeReady is false on native: the badge draws on canvas.
func webVersionBadgeReady() bool { return false }

// SetWebVersionBadge is a no-op on native builds.
func SetWebVersionBadge(label string) {}
