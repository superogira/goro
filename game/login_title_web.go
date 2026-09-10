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
// as a one-time PNG data URL, the current phase, and the fade-cover alpha
// (the layer's opacity is 1-alpha). Only changed values are sent, so the
// steady state costs nothing per frame.
func (m *LoginMode) syncTitleWeb(ctx client.Context, alpha float64) {
	if !titleWebEnabled() {
		return
	}
	gameui.InstallWebActionHooks()
	phase := "account"
	switch m.phase {
	case loginPhaseCharacter:
		phase = "character"
	case loginPhaseCreate:
		phase = "create"
	}
	fade := strconv.FormatFloat(alpha, 'f', 3, 64)
	if m.titleWebBG != "" && m.titleWebPhase == phase && m.titleWebFade == fade {
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
	m.titleWebPhase = phase
	m.titleWebFade = fade
	js.Global().Get("goroTitleSync").Invoke(obj)
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
