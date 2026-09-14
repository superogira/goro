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
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

// DOM twin of the character select screen: the page renders the slot row,
// info table, footer buttons and the delete confirm/email modals; the game
// keeps ownership of selection, slots, previews and packets. Keyboard input
// (arrows, Enter, Escape) keeps working through the canvas input path and
// the DOM layer follows the same selection.

// charSelectWebEnabled reports whether the page provides the DOM character
// select layer.
func charSelectWebEnabled() bool {
	return js.Global().Get("goroCharSelectSync").Type() == js.TypeFunction
}

// skipCanvasCharSelectBackground hides the canvas login background while
// the DOM layer fully covers the character phase.
func (m *LoginMode) skipCanvasCharSelectBackground() bool {
	return charSelectWebEnabled() && m.phase == loginPhaseCharacter &&
		m.fade.phase == loginFadeNone && !m.fade.enterWorld
}

// drainCharSelectWebActions services the layer's "char:" actions: slot
// taps, footer buttons, pagination and the delete confirm/email modals.
func (m *LoginMode) drainCharSelectWebActions(ctx client.Context) {
	for _, action := range gameui.DrainWebActions("char:") {
		m.handleCharSelectWebAction(ctx, action)
	}
}

func (m *LoginMode) handleCharSelectWebAction(ctx client.Context, action string) {
	switch {
	case action == "char:prev":
		m.moveToPreviousCharacterSlot()
	case action == "char:next":
		m.moveToNextCharacterSlot()
	case action == "char:make":
		slot := m.selectedSlot
		if _, ok := characterBySlot(ctx.Session.Characters, slot); ok {
			m.status = "character slot occupied"
			return
		}
		m.openCharacterCreate(ctx, slot, time.Now())
	case action == "char:ok":
		m.submitSelectedCharacter(ctx)
	case action == "char:cancel":
		m.cancelCharacterSelect(ctx)
	case action == "char:delete":
		m.openCharacterDeleteConfirm(ctx)
	case action == "char:delok":
		if m.charDeleteWebStep == 1 {
			m.charDeleteWebStep = 2
		}
	case action == "char:delcancel", action == "char:delmailcancel":
		m.charDeleteWebStep = 0
		m.deleteCharID = 0
	case strings.HasPrefix(action, "char:delmail:"):
		if m.charDeleteWebStep != 2 {
			return
		}
		key := strings.TrimPrefix(action, "char:delmail:")
		m.charDeleteWebStep = 0
		m.submitCharacterDelete(ctx, key)
	case strings.HasPrefix(action, "char:select:"):
		if slot, err := strconv.Atoi(strings.TrimPrefix(action, "char:select:")); err == nil {
			m.selectedSlot = clampCharacterSlot(slot, m.maxSlots)
		}
	case strings.HasPrefix(action, "char:activate:"):
		if slot, err := strconv.Atoi(strings.TrimPrefix(action, "char:activate:")); err == nil {
			m.activateCharacterSelectSlot(ctx, slot, time.Now())
		}
	}
}

// syncCharSelectWeb pushes the layer state when something changed: slot
// pages with preview sprites and stats, the selection highlight, the page
// counter, the status line and the delete modal step.
func (m *LoginMode) syncCharSelectWeb(ctx client.Context) {
	if !charSelectWebEnabled() {
		return
	}
	gameui.InstallWebActionHooks()
	fade := float64(m.fadeAlpha(time.Now())) / 255
	page := gameui.CharacterSelectPage(m.selectedSlot)
	key := fmt.Sprintf("sel=%d;max=%d;page=%d;n=%d;st=%q;del=%d;name=%q;fade=%.3f;phase=%d",
		m.selectedSlot, m.maxSlots, page, len(ctx.Session.Characters), m.status,
		m.charDeleteWebStep, m.charDeleteWebName, fade, m.phase)
	for _, character := range ctx.Session.Characters {
		key += fmt.Sprintf(";c%d=%d", character.Slot, character.ID)
	}
	if key == m.charWebSyncKey {
		return
	}
	m.charWebSyncKey = key
	if m.charWebBG == "" {
		m.charWebBG = m.titleWebBackgroundData(ctx)
		if m.charWebBG == "" {
			return // background not loaded yet; retry next frame
		}
	}
	obj := js.Global().Get("Object").New()
	obj.Set("phase", phaseName(m.phase))
	obj.Set("fade", fade)
	obj.Set("bg", m.charWebBG)
	obj.Set("selected", m.selectedSlot)
	obj.Set("page", page)
	obj.Set("pageCount", maxInt(1, (m.maxSlots+2)/3))
	obj.Set("status", m.status)
	obj.Set("deleteStep", m.charDeleteWebStep)
	obj.Set("deleteName", m.charDeleteWebName)

	pageStart := page * 3
	slots := js.Global().Get("Array").New(3)
	for localSlot := 0; localSlot < 3; localSlot++ {
		slot := pageStart + localSlot
		entry := js.Global().Get("Object").New()
		character, ok := characterBySlot(ctx.Session.Characters, slot)
		if ok {
			entry.Set("name", trimRunesForWeb(character.Name, 24))
			entry.Set("job", trimRunesForWeb(db.JobDisplayName(int(character.Job)), 18))
			entry.Set("level", fmt.Sprintf("%d / %d", character.Level, character.JobLevel))
			entry.Set("exp", strconv.FormatUint(uint64(character.Exp), 10))
			entry.Set("hp", fmt.Sprintf("%d / %d", character.HP, character.MaxHP))
			entry.Set("sp", fmt.Sprintf("%d / %d", character.SP, character.MaxSP))
			entry.Set("str", strconv.Itoa(int(character.Str)))
			entry.Set("agi", strconv.Itoa(int(character.Agi)))
			entry.Set("vit", strconv.Itoa(int(character.Vit)))
			entry.Set("int", strconv.Itoa(int(character.Int)))
			entry.Set("dex", strconv.Itoa(int(character.Dex)))
			entry.Set("luk", strconv.Itoa(int(character.Luk)))
			entry.Set("preview", m.charPreviewWebURL(ctx, character))
		}
		slots.SetIndex(localSlot, entry)
	}
	obj.Set("slots", slots)
	js.Global().Get("goroCharSelectSync").Invoke(obj)
}

// charPreviewWebURL encodes the cached 139x144 character preview as a PNG
// data URL, memoized per character.
func (m *LoginMode) charPreviewWebURL(ctx client.Context, character session.Character) string {
	if character.ID == 0 {
		return ""
	}
	if m.charPreviewWebURLs == nil {
		m.charPreviewWebURLs = make(map[uint32]string)
	}
	if url, ok := m.charPreviewWebURLs[character.ID]; ok {
		return url
	}
	img := m.characterPreviewImage(ctx, character)
	if img == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	url := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	m.charPreviewWebURLs[character.ID] = url
	return url
}

func trimRunesForWeb(s string, max int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max])
}

func phaseName(phase loginPhase) string {
	switch phase {
	case loginPhaseCharacter:
		return "character"
	case loginPhaseCreate:
		return "create"
	default:
		return "account"
	}
}
