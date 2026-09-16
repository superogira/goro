package ui

import (
	"github.com/kivutar/goro/input"
	"strconv"
	"strings"

	"github.com/gogpu/ui/core/radio"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	chatRoomCreateW        = 306
	chatRoomCreateContentH = 180
)

type ChatRoomCreateWindowAction struct {
	Title    string
	Password string
	Limit    uint16
	Public   bool
}

type ChatRoomCreateWindow struct {
	Window
	title         string
	password      string
	limit         string
	public        bool
	titleField    *textfield.Widget
	passwordField *textfield.Widget
	limitField    *textfield.Widget
	action        ChatRoomCreateWindowAction
	// webOpen marks the DOM-panel twin as the active presentation (web
	// build only); the canvas window is untouched in that mode.
	webOpen bool
}

func (w *ChatRoomCreateWindow) Open(ctx Context) {
	w.ctx = ctx
	if chatRoomCreateWebSync(true) {
		// The DOM twin owns the presentation; the form values live in
		// the page, which resets them on every open.
		w.webOpen = true
		w.open = true
		return
	}
	w.EnsureWindow(chatRoomCreateW, ROWindowTitleHeight+chatRoomCreateContentH+ROWindowFooterHeight)
	w.title = ""
	w.password = ""
	w.limit = "20"
	w.public = true
	w.titleField = nil
	w.passwordField = nil
	w.limitField = nil
	w.action = ChatRoomCreateWindowAction{}
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.focusTitle()
	w.Publish(ctx)
}

// Close hides whichever presentation is active.
func (w *ChatRoomCreateWindow) Close() {
	if w.webOpen {
		w.webOpen = false
		w.open = false
		chatRoomCreateWebSync(false)
		return
	}
	w.Window.Close()
}

func (w *ChatRoomCreateWindow) Update(ctx Context) bool {
	w.ctx = ctx
	if !w.IsOpen() {
		return false
	}
	if w.webOpen {
		return w.updateWeb(ctx)
	}
	w.EnsureWindow(chatRoomCreateW, ROWindowTitleHeight+chatRoomCreateContentH+ROWindowFooterHeight)
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
func (w *ChatRoomCreateWindow) updateWeb(ctx Context) bool {
	action, ok, cancelled := drainChatRoomCreateWebActions()
	if cancelled {
		w.Close()
		return true
	}
	if ok {
		if action.Title == "" {
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

func (w *ChatRoomCreateWindow) Rebind(ctx Context) {
	if w.webOpen || !w.IsOpen() {
		return
	}
	w.ctx = ctx
	w.titleField = nil
	w.passwordField = nil
	w.limitField = nil
	w.SetContent(w.widgetTree(ctx))
	w.focusTitle()
	w.Publish(ctx)
}

func (w *ChatRoomCreateWindow) PopAction() ChatRoomCreateWindowAction {
	action := w.action
	w.action = ChatRoomCreateWindowAction{}
	return action
}

func (w *ChatRoomCreateWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Make a Room"),
		CloseButton(true),
		OnClose(w.Close),
		Size(chatRoomCreateW, ROWindowTitleHeight+chatRoomCreateContentH+ROWindowFooterHeight),
		Content(
			primitives.Box(
				rotheme.Label("Title"),
				primitives.Box(w.titleInput(ctx)).
					Height(24).
					CrossAlign(primitives.CrossAxisStretch),
				primitives.HBox(
					primitives.Box(
						rotheme.Label("Limit"),
						primitives.Box(w.limitInput(ctx)).
							Height(24).
							CrossAlign(primitives.CrossAxisStretch),
					).
						Width(72).
						Gap(4),
					primitives.Box(
						rotheme.Label("Type"),
						primitives.Box(
							rotheme.Radio(
								radio.Items(
									radio.ItemDef{Value: "public", Label: "Public"},
									radio.ItemDef{Value: "private", Label: "Private"},
								),
								radio.DirectionOpt(radio.Horizontal),
								radio.Selected(chatRoomTypeValue(w.public)),
								radio.OnChange(func(value string) {
									w.public = value != "private"
								}),
							),
						).Height(24),
					).
						Gap(4),
				).
					Gap(12),
				rotheme.Label("Password"),
				primitives.Box(w.passwordInput(ctx)).
					Height(24).
					CrossAlign(primitives.CrossAxisStretch),
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

func (w *ChatRoomCreateWindow) titleInput(ctx Context) *textfield.Widget {
	if w.titleField != nil {
		return w.titleField
	}
	w.titleField = rotheme.TextField(
		w.title,
		textfield.TypeText,
		func(value string) {
			w.title = value
		},
		func(string) {
			w.submit(ctx)
		},
		textfield.MaxLength(32),
		textfield.Placeholder("Room Title"),
	)
	return w.titleField
}

func (w *ChatRoomCreateWindow) passwordInput(ctx Context) *textfield.Widget {
	if w.passwordField != nil {
		return w.passwordField
	}
	w.passwordField = rotheme.TextField(
		w.password,
		textfield.TypeText,
		func(value string) {
			w.password = value
		},
		func(string) {
			w.submit(ctx)
		},
		textfield.MaxLength(8),
		textfield.Placeholder("Optional"),
	)
	return w.passwordField
}

func (w *ChatRoomCreateWindow) limitInput(ctx Context) *textfield.Widget {
	if w.limitField != nil {
		return w.limitField
	}
	w.limitField = rotheme.TextField(
		w.limit,
		textfield.TypeNumber,
		func(value string) {
			w.limit = value
		},
		func(string) {
			w.submit(ctx)
		},
		textfield.MaxLength(2),
	)
	return w.limitField
}

func (w *ChatRoomCreateWindow) submit(ctx Context) {
	title := strings.TrimSpace(w.title)
	if title == "" {
		return
	}
	w.action = ChatRoomCreateWindowAction{
		Title:    title,
		Password: strings.TrimSpace(w.password),
		Limit:    chatRoomLimit(w.limit),
		Public:   w.public,
	}
	w.Close()
	w.Publish(ctx)
}

func (w *ChatRoomCreateWindow) submitFromFocusedEnter(ctx Context) bool {
	if ctx.Input == nil {
		return false
	}
	focused := w.titleField != nil && w.titleField.IsFocused() ||
		w.passwordField != nil && w.passwordField.IsFocused() ||
		w.limitField != nil && w.limitField.IsFocused()
	if !focused || !ctx.Input.JustPressed(input.KeyEnter) {
		return false
	}
	w.submit(ctx)
	return true
}

func (w *ChatRoomCreateWindow) focusTitle() {
	if w.titleField != nil {
		w.titleField.SetFocused(true)
	}
}

func chatRoomTypeValue(public bool) string {
	if public {
		return "public"
	}
	return "private"
}

func chatRoomLimit(value string) uint16 {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 20
	}
	if n < 2 {
		return 2
	}
	if n > 20 {
		return 20
	}
	return uint16(n)
}
