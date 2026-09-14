//go:build js && wasm

package game

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/kivutar/goro/client"
	gameui "github.com/kivutar/goro/ui"
)

// DOM twin of the make-character screen: the page renders the sprite
// preview with hair controls, the name field, the stat steppers and the
// Make/Cancel footer; the game keeps ownership of the create state, the
// stat pair rules and packets. Escape still routes through the global
// phase escape (cancel back to character select).

// charCreateWebEnabled reports whether the page provides the DOM character
// creation layer.
func charCreateWebEnabled() bool {
	return js.Global().Get("goroCharCreateSync").Type() == js.TypeFunction
}

// drainCharCreateWebActions services the layer's "create:" actions: name
// drafts, hair controls, stat bumps, submit and cancel.
func (m *LoginMode) drainCharCreateWebActions(ctx client.Context) {
	for _, action := range gameui.DrainWebActions("create:") {
		m.handleCharCreateWebAction(ctx, action)
	}
}

func (m *LoginMode) handleCharCreateWebAction(ctx client.Context, action string) {
	switch {
	case action == "create:hairprev":
		m.changeCreateHairStyle(-1)
	case action == "create:hairnext":
		m.changeCreateHairStyle(1)
	case action == "create:haircolor":
		m.changeCreateHairColor()
	case action == "create:submit":
		m.submitCharacterCreate(ctx)
	case action == "create:cancel":
		m.cancelCharacterCreate(time.Now())
	case strings.HasPrefix(action, "create:stat:"):
		if stat, err := strconv.Atoi(strings.TrimPrefix(action, "create:stat:")); err == nil {
			if !bumpCreateStat(&m.create.stats, stat) {
				m.status = "stat limit reached"
			}
		}
	case strings.HasPrefix(action, "create:name:"):
		m.create.name = appendCharacterNameInput("", strings.TrimPrefix(action, "create:name:"), charCreateNameMaxBytes)
	}
}

// syncCharCreateWeb pushes the layer state when something changed: the
// draft name, the six stats, the hair style/color, the preview sprite and
// the status line.
func (m *LoginMode) syncCharCreateWeb(ctx client.Context) {
	if !charCreateWebEnabled() {
		return
	}
	gameui.InstallWebActionHooks()
	fade := float64(m.fadeAlpha(time.Now())) / 255
	key := fmt.Sprintf("phase=%d;fade=%.3f;name=%q;stats=%v;hair=%d,%d;st=%q",
		m.phase, fade, m.create.name, m.create.stats, m.create.hairStyle, m.create.hairColor, m.status)
	if key == m.createWebSyncKey {
		return
	}
	m.createWebSyncKey = key
	if m.createWebBG == "" {
		m.createWebBG = m.titleWebBackgroundData(ctx)
		if m.createWebBG == "" {
			return // background not loaded yet; retry next frame
		}
	}
	obj := js.Global().Get("Object").New()
	obj.Set("phase", phaseName(m.phase))
	obj.Set("fade", fade)
	obj.Set("bg", m.createWebBG)
	obj.Set("name", m.create.name)
	obj.Set("slot", m.create.slot)
	obj.Set("hairStyle", m.create.hairStyle)
	obj.Set("hairColor", m.create.hairColor)
	obj.Set("status", m.status)
	stats := js.Global().Get("Array").New(createStatCount)
	for i, value := range m.create.stats {
		stats.SetIndex(i, int(value))
	}
	obj.Set("stats", stats)
	obj.Set("preview", m.charCreatePreviewWebURL(ctx))
	js.Global().Get("goroCharCreateSync").Invoke(obj)
}

// charCreatePreviewWebURL encodes the current create preview (hair style +
// color sprite) as a PNG data URL, memoized per preview key.
func (m *LoginMode) charCreatePreviewWebURL(ctx client.Context) string {
	key := charCreatePreviewKey{sex: ctx.Session.Sex, hairStyle: m.create.hairStyle, hairColor: m.create.hairColor}
	if m.createPreviewWebKey == key && m.createPreviewWebURL != "" {
		return m.createPreviewWebURL
	}
	img := m.characterCreatePreviewImage(ctx)
	if img == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	m.createPreviewWebKey = key
	m.createPreviewWebURL = "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	return m.createPreviewWebURL
}
