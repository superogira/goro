package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
)

// HandleKeyPress gives gameplay input handlers a chance to consume a press
// before default UI dispatch. Focused chat, forms and modals retain priority.
func (m *WorldMode) HandleKeyPress(ctx client.Context, code input.KeyCode) {
	if m.uiInputSuspended() || m.ui.KeyboardShortcutsBlocked(ctx) {
		return
	}
	m.botKeyPress(ctx, code)
}
