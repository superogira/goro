//go:build js && wasm

package game

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/draw"
	"image/png"
	"strconv"
	"syscall/js"

	"github.com/kivutar/goro/client"
	gameui "github.com/kivutar/goro/ui"
	xdraw "golang.org/x/image/draw"
)

// titleWebEnabled reports whether the page provides the DOM title layer.
func titleWebEnabled() bool {
	return js.Global().Get("goroTitleSync").Type() == js.TypeFunction
}

// skipCanvasTitleBackground hides the canvas title background while the
// DOM layer fully covers it (account phase, no fade in flight) — the
// full-screen blit every frame is exactly the load the DOM layer removes.
func (m *LoginMode) skipCanvasTitleBackground() bool {
	return titleWebEnabled() && m.phase == loginPhaseAccount &&
		m.fade.phase == loginFadeNone && !m.fade.enterWorld
}

// syncTitleWeb pushes the title layer state to the page: the background
// as a one-time PNG data URL, the current phase, the fade-cover alpha
// (the layer's opacity is 1-alpha), the status line, and the login-phase
// modal alert (refused login / lost connection) that the skipped canvas
// can no longer show. Only changed values are sent, so the steady state
// costs nothing per frame. It also drains the layer's own "title:"
// actions here — the modals swallow Update input while open, so this is
// the one path that always runs.
func (m *LoginMode) syncTitleWeb(ctx client.Context, alpha float64) {
	if !titleWebEnabled() {
		return
	}
	gameui.InstallWebActionHooks()
	for _, action := range gameui.DrainWebActions("title:") {
		m.handleTitleWebAction(ctx, action)
	}
	phase := "account"
	switch m.phase {
	case loginPhaseCharacter:
		phase = "character"
	case loginPhaseCreate:
		phase = "create"
	}
	fade := strconv.FormatFloat(alpha, 'f', 3, 64)
	alert := m.titleWebAlertState()
	if m.titleWebBG != "" && m.titleWebPhase == phase && m.titleWebFade == fade &&
		m.titleWebStatus == m.status && m.titleWebAlert == alert {
		return
	}
	if m.titleWebBG == "" {
		m.titleWebBG = m.titleWebBackgroundData(ctx)
		if m.titleWebBG == "" {
			return // background not loaded yet; retry next frame
		}
	}
	obj := js.Global().Get("Object").New()
	obj.Set("phase", phase)
	obj.Set("fade", alpha)
	obj.Set("bg", m.titleWebBG)
	obj.Set("status", m.status)
	if alert != "" {
		title, message, okOnly := m.titleWebAlertContent()
		alertObj := js.Global().Get("Object").New()
		alertObj.Set("title", title)
		alertObj.Set("message", message)
		alertObj.Set("okOnly", okOnly)
		obj.Set("alert", alertObj)
	} else {
		obj.Set("alert", nil)
	}
	m.titleWebPhase = phase
	m.titleWebFade = fade
	m.titleWebStatus = m.status
	m.titleWebAlert = alert
	js.Global().Get("goroTitleSync").Invoke(obj)
}

// titleWebAlertState returns a signature of the open login-phase modal
// ("" when none), and titleWebAlertContent its pieces. While the DOM
// layer covers the account phase the canvas is not drawn, so a canvas
// modal there would be invisible yet still swallow all input — the DOM
// alert is the only visible face of it.
func (m *LoginMode) titleWebAlertState() string {
	if m.disconnectDialog.IsOpen() {
		title, message, okOnly := m.alertDialogContent(&m.disconnectDialog)
		return title + "\x00" + message + "\x00" + strconv.FormatBool(okOnly)
	}
	if m.quitConfirm.IsOpen() {
		title, message, okOnly := m.alertDialogContent(&m.quitConfirm)
		return title + "\x00" + message + "\x00" + strconv.FormatBool(okOnly)
	}
	return ""
}

func (m *LoginMode) titleWebAlertContent() (string, string, bool) {
	if m.disconnectDialog.IsOpen() {
		return m.alertDialogContent(&m.disconnectDialog)
	}
	return m.alertDialogContent(&m.quitConfirm)
}

func (m *LoginMode) alertDialogContent(dialog *gameui.ConfirmModal) (string, string, bool) {
	return dialog.DialogTitle(), dialog.DialogMessage(), dialog.DialogOKOnly()
}

func (m *LoginMode) handleTitleWebAction(ctx client.Context, action string) {
	if action != "title:alertok" && action != "title:alertcancel" {
		return
	}
	target := &m.disconnectDialog
	if !target.IsOpen() {
		target = &m.quitConfirm
	}
	if !target.IsOpen() {
		return
	}
	if action == "title:alertok" || target.DialogOKOnly() {
		target.Confirm(ctx)
		return
	}
	target.Cancel(ctx)
}

// titleWebBackgroundData composes the current title background (single
// image or the 12-tile grid) into one PNG data URL for the page's <img>.
func (m *LoginMode) titleWebBackgroundData(ctx client.Context) string {
	width, height := ctx.ScreenSize()
	if width <= 0 || height <= 0 || !m.bgLoaded {
		return ""
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), image.Black, image.Point{}, draw.Src)
	switch {
	case m.background != nil:
		scaleDraw(dst, m.background.RGBA())
	case len(m.bgTiles) == 12:
		cellW := float64(width) / 4
		cellH := float64(height) / 3
		for i, tile := range m.bgTiles {
			if tile == nil {
				continue
			}
			cell := image.Rect(
				int(float64(i%4)*cellW), int(float64(i/4)*cellH),
				int(float64(i%4+1)*cellW), int(float64(i/4+1)*cellH))
			xdraw.ApproxBiLinear.Scale(dst, cell, tile.RGBA(), tile.Bounds(), xdraw.Over, nil)
		}
	default:
		return ""
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func scaleDraw(dst *image.RGBA, src *image.RGBA) {
	b := src.Bounds()
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
}
