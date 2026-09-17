package ui

import (
	"strings"

	"github.com/gogpu/ui/core/checkbox"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	friendSettingsW        = 286
	friendSettingsContentH = 112
)

type FriendSettingsWindow struct {
	Window
	settings session.WhisperSettings
	// webOpen marks the DOM twin as the active presentation (web build).
	webOpen bool
}

func (w *FriendSettingsWindow) Open(ctx Context) {
	w.EnsureWindow(friendSettingsW, ROWindowTitleHeight+friendSettingsContentH+ROWindowFooterHeight)
	w.ctx = ctx
	w.settings = friendSettings(ctx.Session)
	if friendSetupWebSync(friendSetupWebState{
		open: true, openStranger: w.settings.OpenStrangers, openFriends: w.settings.OpenFriends, alert: w.settings.Alert,
	}) {
		w.webOpen = true
		w.open = true
		return
	}
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

// Close hides whichever presentation is active.
func (w *FriendSettingsWindow) Close() {
	if w.webOpen {
		w.webOpen = false
		w.open = false
		friendSetupWebSync(friendSetupWebState{})
		return
	}
	w.Window.Close()
}

func (w *FriendSettingsWindow) Update(ctx Context) bool {
	w.EnsureWindow(friendSettingsW, ROWindowTitleHeight+friendSettingsContentH+ROWindowFooterHeight)
	w.ctx = ctx
	if !w.IsOpen() {
		return false
	}
	if w.webOpen {
		for _, action := range drainSocialWebActions("fs:") {
			if action == "fs:cancel" {
				w.Close()
				return true
			}
			if parts := strings.Split(action, ":"); len(parts) == 5 && parts[0] == "fs" && parts[1] == "ok" {
				w.settings.OpenStrangers = parts[2] == "1"
				w.settings.OpenFriends = parts[3] == "1"
				w.settings.Alert = parts[4] == "1"
				w.apply(ctx)
				return true
			}
		}
		return true
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *FriendSettingsWindow) Rebind(ctx Context) {
	if !w.IsOpen() {
		return
	}
	w.ctx = ctx
	w.RebindContent(ctx, w.widgetTree(ctx))
}

func (w *FriendSettingsWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Friend Setup"),
		CloseButton(true),
		OnClose(w.Close),
		Size(friendSettingsW, ROWindowTitleHeight+friendSettingsContentH+ROWindowFooterHeight),
		Content(
			primitives.Box(
				rotheme.Checkbox(
					checkbox.Checked(w.settings.OpenStrangers),
					checkbox.LabelOpt("1:1 Chat from Strangers"),
					checkbox.OnToggle(func(enabled bool) {
						w.settings.OpenStrangers = enabled
						w.settings.Configured = true
					}),
				),
				rotheme.Checkbox(
					checkbox.Checked(w.settings.OpenFriends),
					checkbox.LabelOpt("1:1 Chat from Friends"),
					checkbox.OnToggle(func(enabled bool) {
						w.settings.OpenFriends = enabled
						w.settings.Configured = true
					}),
				),
				rotheme.Checkbox(
					checkbox.Checked(w.settings.Alert),
					checkbox.LabelOpt("1:1 Chat Alert"),
					checkbox.OnToggle(func(enabled bool) {
						w.settings.Alert = enabled
						w.settings.Configured = true
					}),
				),
			).
				Padding(14).
				Gap(8),
		),
		Footer(
			primitives.Expanded(primitives.Box()),
			rotheme.Button("OK", func() {
				w.apply(ctx)
			}),
			rotheme.Button("Cancel", w.Close),
		),
	)
}

func (w *FriendSettingsWindow) apply(ctx Context) {
	if ctx.Session != nil {
		w.settings.Configured = true
		ctx.Session.Whisper = w.settings
	}
	if w.webOpen {
		w.Close()
		return
	}
	w.Window.Close()
}

func friendSettings(s *session.Session) session.WhisperSettings {
	if s == nil || !s.Whisper.Configured {
		return session.DefaultWhisperSettings()
	}
	return s.Whisper
}
