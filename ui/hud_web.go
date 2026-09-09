//go:build js && wasm

package ui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"strconv"
	"strings"
	"sync"
	"syscall/js"

	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

// The HUD and basic menu render as page DOM on the web build — the same
// reason the dormant chat log does: browser text is crisp at every device
// scale and zoom level, while canvas-rastered text blurs on Windows
// display scaling. DOM composites on the browser's own threads, so the
// wasm canvas pays nothing for their updates.

// hudWebActionQueue marshals DOM interactions (button taps, close, tab)
// onto the game goroutine.
var hudWebActionQueue = struct {
	sync.Mutex
	actions []string
}{}

var hudWebHooksInstalled bool

// hudWebInstallHooks exposes the page hooks: goroHudSync pushes state
// (called from the game goroutine) and goroMenuAction/goroHudAction feed
// taps back (called from DOM events).
func hudWebInstallHooks() {
	if hudWebHooksInstalled {
		return
	}
	hudWebHooksInstalled = true
	js.Global().Set("goroHudAction", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) >= 1 && args[0].Type() == js.TypeString {
			hudWebActionQueue.Lock()
			hudWebActionQueue.actions = append(hudWebActionQueue.actions, "hud:"+args[0].String())
			hudWebActionQueue.Unlock()
		}
		return nil
	}))
	js.Global().Set("goroMenuAction", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) >= 1 && args[0].Type() == js.TypeString {
			hudWebActionQueue.Lock()
			hudWebActionQueue.actions = append(hudWebActionQueue.actions, "menu:"+args[0].String())
			hudWebActionQueue.Unlock()
		}
		return nil
	}))
	js.Global().Set("goroStatsAction", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) >= 1 && args[0].Type() == js.TypeString {
			hudWebActionQueue.Lock()
			hudWebActionQueue.actions = append(hudWebActionQueue.actions, "stats:"+args[0].String())
			hudWebActionQueue.Unlock()
		}
		return nil
	}))
	js.Global().Set("goroCamAction", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) >= 1 && args[0].Type() == js.TypeString {
			hudWebActionQueue.Lock()
			hudWebActionQueue.actions = append(hudWebActionQueue.actions, "cam:"+args[0].String())
			hudWebActionQueue.Unlock()
		}
		return nil
	}))
	js.Global().Set("goroHotbarAction", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) >= 1 && args[0].Type() == js.TypeString {
			hudWebActionQueue.Lock()
			hudWebActionQueue.actions = append(hudWebActionQueue.actions, "hot:"+args[0].String())
			hudWebActionQueue.Unlock()
		}
		return nil
	}))
}

// hudWebDrainActions takes queued DOM interactions whose action starts
// with the given prefix ("hud:" or "menu:") and leaves the rest queued.
// Consumers run at different points of the frame loop, so a consumer must
// never take — let alone discard — another consumer's actions.
func hudWebDrainActions(prefix string) []string {
	hudWebActionQueue.Lock()
	defer hudWebActionQueue.Unlock()
	if len(hudWebActionQueue.actions) == 0 {
		return nil
	}
	var out []string
	kept := hudWebActionQueue.actions[:0]
	for _, action := range hudWebActionQueue.actions {
		if strings.HasPrefix(action, prefix) {
			out = append(out, action)
		} else {
			kept = append(kept, action)
		}
	}
	hudWebActionQueue.actions = kept
	return out
}

// hudWebSync pushes the full HUD state and both windows' visibility to the
// page. Values arrive pre-formatted so the page only ever interpolates.
func hudWebSync(fields [15]string) {
	sync := js.Global().Get("goroHudSync")
	if sync.Type() != js.TypeFunction {
		return
	}
	obj := js.Global().Get("Object").New()
	names := []string{
		"open", "name", "job",
		"hp", "maxHp", "sp", "maxSp",
		"baseLv", "baseExp", "nextBaseExp",
		"jobLv", "jobExp", "nextJobExp",
		"weight", "zeny",
	}
	for i, name := range names {
		if i == 0 {
			obj.Set(name, fields[0] == "1")
			continue
		}
		obj.Set(name, fields[i])
	}
	sync.Invoke(obj)
}

// statsWebSync pushes the status window state to the page.
func statsWebSync(open bool, rows [6]statRow, derived [][2]string, points int, canIncrease [6]bool, guild string) {
	fn := js.Global().Get("goroStatsSync")
	if fn.Type() != js.TypeFunction {
		return
	}
	obj := js.Global().Get("Object").New()
	obj.Set("open", open)
	rowArr := js.Global().Get("Array").New(len(rows))
	for i, row := range rows {
		entry := js.Global().Get("Object").New()
		entry.Set("label", row.label)
		entry.Set("value", formatStatValue(row.value, row.bonus))
		entry.Set("cost", row.cost)
		entry.Set("canInc", canIncrease[i])
		rowArr.SetIndex(i, entry)
	}
	obj.Set("rows", rowArr)
	derArr := js.Global().Get("Array").New(len(derived))
	for i, pair := range derived {
		entry := js.Global().Get("Object").New()
		entry.Set("label", pair[0])
		entry.Set("value", pair[1])
		derArr.SetIndex(i, entry)
	}
	obj.Set("derived", derArr)
	obj.Set("points", points)
	obj.Set("guild", guild)
	fn.Invoke(obj)
}

// statsWebEnabled reports whether the page provides the DOM status window.
func statsWebEnabled() bool {
	return js.Global().Get("goroStatsSync").Type() == js.TypeFunction
}

// pickupWebEnabled reports whether the page provides the DOM pickup toast.
func pickupWebEnabled() bool {
	return js.Global().Get("goroPickupShow").Type() == js.TypeFunction
}

// pickupWebShow raises the pickup toast on the page; the page times it out.
// The item's icon travels beside the text as a cached data URL.
func pickupWebShow(text string, icon string) {
	fn := js.Global().Get("goroPickupShow")
	if fn.Type() != js.TypeFunction {
		return
	}
	fn.Invoke(text, icon)
}

// pickupWebIcon resolves and caches the item icon data URL.
func pickupWebIcon(manager *res.Manager, item session.InventoryItem) string {
	if manager == nil || item.ItemID == 0 {
		return ""
	}
	key := fmt.Sprintf("pickup:%d:%t", item.ItemID, item.Identified)
	if url, ok := hotbarWebIconCache[key]; ok {
		return url
	}
	resourceName, ok := manager.ItemResourceName(int(item.ItemID), item.Identified)
	if !ok {
		return ""
	}
	img, _, err := res.LoadImage(manager, res.ItemIconTextureCandidates(resourceName))
	if err != nil {
		return ""
	}
	url := hotbarWebIcon(key, img)
	return url
}

// DrainCameraActions takes queued camera-button presses; it runs even when
// the rest of the UI chain is gated. The cam: prefix is stripped so
// consumers match bare action names.
func DrainCameraActions() []string {
	var out []string
	for _, action := range hudWebDrainActions("cam:") {
		out = append(out, strings.TrimPrefix(action, "cam:"))
	}
	return out
}

// hotbarWebEnabled reports whether the page provides the DOM hotbar.
func hotbarWebEnabled() bool {
	return js.Global().Get("goroHotbarSync").Type() == js.TypeFunction
}

// hotbarWebIconCache memoizes PNG data URLs per icon key.
var hotbarWebIconCache = map[string]string{}

// hotbarWebIcon encodes an icon image as a cached data URL.
func hotbarWebIcon(key string, img image.Image) string {
	if img == nil {
		return ""
	}
	if url, ok := hotbarWebIconCache[key]; ok {
		return url
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	url := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	hotbarWebIconCache[key] = url
	return url
}

// hotbarWebSync pushes the hotbar layout and slot contents to the page.
// The slot rects stay in Go (slotAt) so canvas-originated drag-drops keep
// landing; the DOM only renders what Go lays out.
func (b *ShortcutBar) hotbarWebSync(ctx Context) {
	fn := js.Global().Get("goroHotbarSync")
	if fn.Type() != js.TypeFunction {
		return
	}
	rows := b.visibleRowCount()
	x, y := b.bounds(ctx)
	obj := js.Global().Get("Object").New()
	obj.Set("x", x)
	obj.Set("y", y)
	obj.Set("rows", rows)
	slots := js.Global().Get("Array").New(shortcutTotalSlots)
	for i := 0; i < shortcutTotalSlots; i++ {
		entry := js.Global().Get("Object").New()
		icon, label := "", ""
		switch b.slots[i].kind {
		case shortcutItem:
			item := session.InventoryItem{ItemID: b.slots[i].itemID, Index: b.slots[i].itemIndex, Identified: b.slots[i].identified, Amount: 1}
			if live, ok := inventoryItemForShortcut(ctx.Session, b.slots[i].itemIndex, b.slots[i].itemID); ok {
				item = live
			}
			icon = hotbarWebIcon(fmt.Sprintf("item:%d:%t", item.ItemID, item.Identified), b.itemIconImage(ctx.Resources, item))
			if item.Amount > 1 {
				label = strconv.Itoa(int(item.Amount))
			}
		case shortcutSkill:
			skill, _ := skillForShortcut(ctx.Session, b.slots[i])
			if skill.ID == 0 {
				skill = session.Skill{ID: b.slots[i].skillID, Level: b.slots[i].skillLevel}
			}
			if b.assets != nil {
				icon = hotbarWebIcon(fmt.Sprintf("skill:%d", skill.ID), b.assets.SkillIconImage(ctx.Resources, skill, 24))
			}
			if skill.Level > 0 {
				label = "Lv" + strconv.Itoa(maxInt(1, skill.Level))
			}
		}
		entry.Set("icon", icon)
		entry.Set("label", label)
		slots.SetIndex(i, entry)
	}
	obj.Set("slots", slots)
	fn.Invoke(obj)
}

// hotbarWebKey summarizes slot contents for change detection.
func (b *ShortcutBar) hotbarWebKey(ctx Context) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "rows=%d|", b.visibleRowCount())
	for i := range b.slots {
		s := &b.slots[i]
		label := ""
		if s.kind == shortcutItem {
			if live, ok := inventoryItemForShortcut(ctx.Session, s.itemIndex, s.itemID); ok && live.Amount > 1 {
				label = strconv.Itoa(int(live.Amount))
			}
		} else if s.kind == shortcutSkill {
			if skill, _ := skillForShortcut(ctx.Session, *s); skill.Level > 0 {
				label = strconv.Itoa(skill.Level)
			}
		}
		fmt.Fprintf(&sb, "%d:%d:%d:%d:%s;", i, s.kind, s.itemID, s.skillID, label)
	}
	return sb.String()
}

// menuWebSync reports the menu window's visibility.
func menuWebSync(open bool) {
	sync := js.Global().Get("goroHudSync")
	if sync.Type() != js.TypeFunction {
		return
	}
	obj := js.Global().Get("Object").New()
	obj.Set("menuOpen", open)
	sync.Invoke(obj)
}

// hudWebEnabled reports whether the page provides the DOM HUD.
func hudWebEnabled() bool {
	return js.Global().Get("goroHudSync").Type() == js.TypeFunction
}

// hudFormatNumber groups thousands with commas, matching the canvas HUD.
func hudFormatNumber(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	text := strconv.FormatInt(value, 10)
	if len(text) <= 3 {
		return sign + text
	}
	var b strings.Builder
	prefix := len(text) % 3
	if prefix == 0 {
		prefix = 3
	}
	b.WriteString(text[:prefix])
	for i := prefix; i < len(text); i += 3 {
		b.WriteByte(',')
		b.WriteString(text[i : i+3])
	}
	return sign + b.String()
}
