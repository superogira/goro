//go:build !js || !wasm

package gogpu

// paceBrowserFrame is a no-op on native platforms where VSync-present blocks
// and paces the loop naturally.
func paceBrowserFrame() {}
