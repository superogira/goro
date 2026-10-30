package npc

import "github.com/kivutar/goro/network"

func (m *System) ApplyDialog(dialog network.NPCDialog) {
	m.bindings.Windows.Dialog.Apply(dialog)
	if !m.bindings.Windows.Dialog.IsOpen() && (dialog.Kind == network.NPCDialogClear || dialog.Kind == network.NPCDialogClose) {
		m.bindings.Windows.Cutin.Clear()
	}
}
