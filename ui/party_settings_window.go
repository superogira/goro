package ui

import (
	"strconv"
	"strings"

	"github.com/gogpu/ui/core/checkbox"
	"github.com/gogpu/ui/core/radio"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	partySettingsW       = 286
	partySettingsContent = 132
)

type PartySettingsWindow struct {
	Window
	expShare      uint32
	refuseInvites bool
	// webOpen marks the DOM twin as the active presentation (web build).
	webOpen bool
}

func (w *PartySettingsWindow) Open(ctx Context) {
	w.EnsureWindow(partySettingsW, ROWindowTitleHeight+partySettingsContent+ROWindowFooterHeight)
	w.ctx = ctx
	party := sessionParty(ctx.Session)
	w.expShare = party.ExpShare
	w.refuseInvites = party.RefuseInvites
	if partySetupWebSync(partySetupWebState{open: true, expShare: w.expShare, refuseInvite: w.refuseInvites}) {
		w.webOpen = true
		w.open = true
		return
	}
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

// Close hides whichever presentation is active.
func (w *PartySettingsWindow) Close() {
	if w.webOpen {
		w.webOpen = false
		w.open = false
		partySetupWebSync(partySetupWebState{})
		return
	}
	w.Window.Close()
}

func (w *PartySettingsWindow) Update(ctx Context) bool {
	w.EnsureWindow(partySettingsW, ROWindowTitleHeight+partySettingsContent+ROWindowFooterHeight)
	w.ctx = ctx
	if !w.IsOpen() {
		return false
	}
	if w.webOpen {
		for _, action := range drainSocialWebActions("ps:") {
			if action == "ps:cancel" {
				w.Close()
				return true
			}
			if parts := strings.Split(action, ":"); len(parts) == 4 && parts[0] == "ps" && parts[1] == "ok" {
				w.expShare = parsePartySettingUint32(parts[2])
				w.refuseInvites = parts[3] == "1"
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

func (w *PartySettingsWindow) Rebind(ctx Context) {
	if !w.IsOpen() {
		return
	}
	w.ctx = ctx
	w.RebindContent(ctx, w.widgetTree(ctx))
}

func (w *PartySettingsWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Party Settings"),
		CloseButton(true),
		OnClose(w.Close),
		Size(partySettingsW, ROWindowTitleHeight+partySettingsContent+ROWindowFooterHeight),
		Content(
			primitives.Box(
				rotheme.Label("EXP"),
				rotheme.Radio(
					radio.Items(
						radio.ItemDef{Value: "0", Label: "Each Take"},
						radio.ItemDef{Value: "1", Label: "Even Share"},
					),
					radio.Selected(strconv.Itoa(int(w.expShare))),
					radio.OnChange(func(value string) {
						w.expShare = parsePartySettingUint32(value)
					}),
				),
				rotheme.Checkbox(
					checkbox.Checked(w.refuseInvites),
					checkbox.LabelOpt("Refuse party invites"),
					checkbox.OnToggle(func(enabled bool) {
						w.refuseInvites = enabled
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

func (w *PartySettingsWindow) apply(ctx Context) {
	if w.webOpen {
		defer w.Close()
	}
	if ctx.Session != nil {
		ctx.Session.Party.ExpShare = w.expShare
		ctx.Session.Party.RefuseInvites = w.refuseInvites
	}
	if ctx.Network != nil {
		if err := ctx.Network.SendPartyOption(w.expShare); err != nil {
			glog.Warnf("party settings failed: %v", err)
		}
		if err := ctx.Network.SendPartyInviteConfig(w.refuseInvites); err != nil {
			glog.Warnf("party invite settings failed: %v", err)
		}
	}
	if !w.webOpen {
		w.Window.Close()
	}
}

func parsePartySettingUint32(value string) uint32 {
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0
	}
	return uint32(n)
}

func parsePartySettingUint8(value string) uint8 {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0
	}
	return 1
}
