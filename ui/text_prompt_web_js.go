//go:build js && wasm

package ui

import (
	"strings"
	"syscall/js"
)

// DOM twin of the text prompt window: the page renders the small
// title/label/input/OK dialog in the family panel styling, and actions
// come back through the shared "tprompt:" queue. One panel serves every
// prompt caller — party invitations, guild reasons, the character
// delete email and skill ground messages.

// textPromptWebSync pushes the prompt state to the page. It reports
// whether the page provides the DOM panel; when false the caller falls
// back to the canvas window.
func textPromptWebSync(title, label, placeholder string, maxLength int, open bool) bool {
	sync := js.Global().Get("goroTextPromptSync")
	if sync.Type() != js.TypeFunction {
		return false
	}
	hudWebInstallHooks()
	obj := js.Global().Get("Object").New()
	obj.Set("open", open)
	obj.Set("title", title)
	obj.Set("label", label)
	obj.Set("placeholder", placeholder)
	obj.Set("maxLength", maxLength)
	sync.Invoke(obj)
	return true
}

// drainTextPromptWebActions services the panel's "tprompt:" actions.
func drainTextPromptWebActions() (submit string, cancelled bool) {
	for _, action := range hudWebDrainActions("tprompt:") {
		switch {
		case action == "tprompt:cancel":
			cancelled = true
		case strings.HasPrefix(action, "tprompt:submit:"):
			submit = strings.TrimPrefix(action, "tprompt:submit:")
		}
	}
	return submit, cancelled
}
