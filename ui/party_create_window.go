package ui

import (
	"github.com/kivutar/goro/input"
	"strings"

	"github.com/gogpu/ui/core/radio"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	partyCreateW        = 306
	partyCreateContentH = 158
)

type PartyCreateWindowAction struct {
	Name         string
	ItemPickup   uint8
	ItemDivision uint8
}

type PartyCreateWindow struct {
	Window
	name         string
	itemPickup   uint8
	itemDivision uint8
	nameField    *textfield.Widget
	action       PartyCreateWindowAction
	// webOpen marks the DOM-panel twin as the active presentation (web
	// build only); the canvas window is untouched in that mode.
	webOpen bool
}

func (w *PartyCreateWindow) Open(ctx Context) {
	w.ctx = ctx
	if partyCreateWebSync(true) {
		// The DOM twin owns the presentation; the form values live in the
		// page, which resets them on every open.
		w.webOpen = true
		w.open = true
		return
	}
	w.EnsureWindow(partyCreateW, ROWindowTitleHeight+partyCreateContentH+ROWindowFooterHeight)
	w.name = ""
	w.itemPickup = 0
	w.itemDivision = 0
	w.nameField = nil
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.focusName()
	w.Publish(ctx)
}

// Close hides whichever presentation is active.
func (w *PartyCreateWindow) Close() {
	if w.webOpen {
		w.webOpen = false
		w.open = false
		partyCreateWebSync(false)
		return
	}
	w.Window.Close()
}

func (w *PartyCreateWindow) Update(ctx Context) bool {
	w.ctx = ctx
	if !w.IsOpen() {
		return false
	}
	if w.webOpen {
		return w.updateWeb(ctx)
	}
	w.EnsureWindow(partyCreateW, ROWindowTitleHeight+partyCreateContentH+ROWindowFooterHeight)
	if w.submitFromFocusedEnter(ctx) {
		w.Publish(ctx)
		return true
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

// updateWeb services the DOM form: submissions arrive through the action
// queue; Escape (which reaches the game only when no page input is
// focused) cancels. Stays consumed while open so game shortcuts yield,
// exactly like the canvas twin.
func (w *PartyCreateWindow) updateWeb(ctx Context) bool {
	action, ok, cancelled := drainPartyCreateWebActions()
	if cancelled {
		w.Close()
		return true
	}
	if ok {
		if action.Name == "" {
			return true
		}
		w.action = action
		w.Close()
		return true
	}
	if ctx.Input != nil && ctx.Input.JustPressed(input.KeyEscape) {
		w.Close()
		return true
	}
	return true
}

func (w *PartyCreateWindow) Rebind(ctx Context) {
	if w.webOpen || !w.IsOpen() {
		return
	}
	w.ctx = ctx
	w.nameField = nil
	content := w.widgetTree(ctx)
	w.focusName()
	w.RebindContent(ctx, content)
}

func (w *PartyCreateWindow) PopAction() PartyCreateWindowAction {
	action := w.action
	w.action = PartyCreateWindowAction{}
	return action
}

func (w *PartyCreateWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Create Party"),
		CloseButton(true),
		OnClose(w.Close),
		Size(partyCreateW, ROWindowTitleHeight+partyCreateContentH+ROWindowFooterHeight),
		Content(
			primitives.Box(
				rotheme.Label("Party Name"),
				primitives.Box(w.nameInput(ctx)).
					Height(24).
					CrossAlign(primitives.CrossAxisStretch),
				rotheme.Label("Item Pickup"),
				rotheme.Radio(
					radio.Items(
						radio.ItemDef{Value: "0", Label: "Each Take"},
						radio.ItemDef{Value: "1", Label: "Party Share"},
					),
					radio.Selected(partyRuleValue(w.itemPickup)),
					radio.OnChange(func(value string) {
						w.itemPickup = parsePartySettingUint8(value)
					}),
				),
				rotheme.Label("Item Sharing"),
				rotheme.Radio(
					radio.Items(
						radio.ItemDef{Value: "0", Label: "Individual"},
						radio.ItemDef{Value: "1", Label: "Shared"},
					),
					radio.Selected(partyRuleValue(w.itemDivision)),
					radio.OnChange(func(value string) {
						w.itemDivision = parsePartySettingUint8(value)
					}),
				),
			).
				Padding(14).
				Gap(6),
		),
		Footer(
			primitives.Expanded(primitives.Box()),
			rotheme.Button("OK", func() {
				w.submit(ctx)
			}),
			rotheme.Button("Cancel", w.Close),
		),
	)
}

func (w *PartyCreateWindow) nameInput(ctx Context) *textfield.Widget {
	if w.nameField != nil {
		return w.nameField
	}
	w.nameField = rotheme.TextField(
		w.name,
		textfield.TypeText,
		func(value string) {
			w.name = value
		},
		func(string) {
			w.submit(ctx)
		},
		textfield.MaxLength(23),
		textfield.Placeholder("Party Name"),
	)
	return w.nameField
}

func (w *PartyCreateWindow) submit(ctx Context) {
	name := strings.TrimSpace(w.name)
	if name == "" {
		return
	}
	w.action = PartyCreateWindowAction{Name: name, ItemPickup: w.itemPickup, ItemDivision: w.itemDivision}
	w.Close()
	w.Publish(ctx)
}

func (w *PartyCreateWindow) submitFromFocusedEnter(ctx Context) bool {
	if ctx.Input == nil || w.nameField == nil || !w.nameField.IsFocused() {
		return false
	}
	if !ctx.Input.JustPressed(input.KeyEnter) {
		return false
	}
	w.submit(ctx)
	return true
}

func (w *PartyCreateWindow) focusName() {
	if w.nameField != nil {
		w.nameField.SetFocused(true)
	}
}

func partyRuleValue(value uint8) string {
	if value != 0 {
		return "1"
	}
	return "0"
}
