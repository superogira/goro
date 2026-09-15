//go:build !js || !wasm

package ui

// Native builds always render the canvas chat shortcuts window.
func chatShortcutsWebEnabled() bool { return false }

func (w *ChatShortcutsWindow) drainChatShortcutsWebActions(ctx Context) {}

func (w *ChatShortcutsWindow) syncChatShortcutsWeb(ctx Context) {}
