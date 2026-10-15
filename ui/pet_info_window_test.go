package ui

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/kivutar/goro/network"
)

func TestPetStatusRefreshPreservesRenameFocusAndDraft(t *testing.T) {
	ctx, _, app := newWindowInstanceTest()
	var w PetInfoWindow
	property := network.PetProperty{Name: "Poring", Job: 1002, Fullness: 80}
	w.OpenInfo(ctx, property)
	field := w.nameField
	wc := app.Window().Context()
	wc.RequestFocus(field)
	field.Event(wc, event.NewKeyEvent(event.KeyPress, event.KeyA, 0, event.ModCtrl))
	for _, r := range "Fluffy" {
		field.Event(wc, event.NewKeyEvent(event.KeyPress, event.KeyUnknown, r, event.ModNone))
	}
	property.Fullness--
	w.OpenInfo(ctx, property)
	if w.nameField != field || wc.FocusedWidget() != field || !w.KeyboardShortcutsBlocked() {
		t.Fatal("status refresh replaced the focused name editor")
	}
	if field.Text() != "Fluffy" || w.name != "Fluffy" || w.property.Fullness != 79 {
		t.Fatal("status refresh lost the rename draft or did not update hunger")
	}
	wc.ReleaseFocus(field)
	w.OpenInfo(ctx, property)
	if field.Text() != property.Name || w.KeyboardShortcutsBlocked() {
		t.Fatal("unfocused editor did not return to the server name")
	}
}

func TestPetRenameEditorResetReleasesFocus(t *testing.T) {
	for _, change := range []string{"rename confirmed", "name changed", "pet changed", "reopened"} {
		t.Run(change, func(t *testing.T) {
			ctx, _, app := newWindowInstanceTest()
			var w PetInfoWindow
			property := network.PetProperty{Name: "Poring", Job: 1002}
			w.OpenInfo(ctx, property)
			field := w.nameField
			wc := app.Window().Context()
			wc.RequestFocus(field)
			field.SetText("Fluffy")
			switch change {
			case "rename confirmed":
				property.Name, property.Modified = "Fluffy", true
			case "name changed":
				property.Name = "Drops"
			case "pet changed":
				property.Job = 1113
			case "reopened":
				w.Close()
			}
			w.OpenInfo(ctx, property)
			if w.nameField == field || field.IsFocused() || wc.FocusedWidget() == field || w.KeyboardShortcutsBlocked() {
				t.Fatal("reset kept the old name editor's focus")
			}
			if w.name != property.Name {
				t.Fatal("reset did not adopt the server name")
			}
			if property.Modified && w.nameField != nil {
				t.Fatal("renamed pet still has an editor")
			}
		})
	}
}
