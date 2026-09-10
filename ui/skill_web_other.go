//go:build !js || !wasm

package ui

// skillWebEnabled reports whether the page provides the DOM skill window.
// Native builds always render the canvas window.
func skillWebEnabled() bool { return false }

// webSync is a no-op without a page; the canvas window owns the UI.
func (w *SkillWindow) webSync(ctx Context) {}

// handleSkillWebAction never runs without a page feeding skill: actions.
func (w *SkillWindow) handleSkillWebAction(ctx Context, actions GameActions, action string) {}
