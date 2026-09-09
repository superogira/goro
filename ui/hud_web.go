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
	js.Global().Set("goroNPCAction", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) >= 1 && args[0].Type() == js.TypeString {
			hudWebActionQueue.Lock()
			hudWebActionQueue.actions = append(hudWebActionQueue.actions, "npc:"+args[0].String())
			hudWebActionQueue.Unlock()
		}
		return nil
	}))
	js.Global().Set("goroInvAction", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) >= 1 && args[0].Type() == js.TypeString {
			hudWebActionQueue.Lock()
			hudWebActionQueue.actions = append(hudWebActionQueue.actions, "inv:"+args[0].String())
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

// inventoryWebEnabled reports whether the page provides the DOM inventory.
func inventoryWebEnabled() bool {
	return js.Global().Get("goroInventorySync").Type() == js.TypeFunction
}

// inventoryWebSync pushes the inventory window state: visibility, active
// tab, and the tab's items (icon data URLs cached like the hotbar's).
func inventoryWebSync(open bool, tab int, items []session.InventoryItem, icons []string, names []string, weights []int) {
	fn := js.Global().Get("goroInventorySync")
	if fn.Type() != js.TypeFunction {
		return
	}
	obj := js.Global().Get("Object").New()
	obj.Set("open", open)
	obj.Set("tab", tab)
	arr := js.Global().Get("Array").New(len(items))
	for i, item := range items {
		entry := js.Global().Get("Object").New()
		entry.Set("icon", icons[i])
		entry.Set("name", names[i])
		entry.Set("amount", item.Amount)
		entry.Set("equipped", item.Equipped)
		entry.Set("weight", weights[i])
		entry.Set("index", item.Index)
		entry.Set("stackable", inventoryItemTypeStackable(item.Type))
		entry.Set("maxAmount", inventoryDropMaxAmount(item))
		arr.SetIndex(i, entry)
	}
	obj.Set("items", arr)
	fn.Invoke(obj)
}

// itemInfoWebSync opens the DOM item-info panel for one item: title with
// refine, description lines, and card slots.
func (w *InventoryBagWindow) itemInfoWebSync(ctx Context, item session.InventoryItem) {
	fn := js.Global().Get("goroItemInfoSync")
	if fn.Type() != js.TypeFunction {
		return
	}
	title := inventoryItemDisplayName(ctx.Resources, item)
	if item.Refine > 0 {
		title = fmt.Sprintf("+%d %s", item.Refine, title)
	}
	desc, _ := ctx.Resources.ItemDescription(int(item.ItemID), item.Identified)
	obj := js.Global().Get("Object").New()
	obj.Set("title", title)
	obj.Set("descHTML", itemInfoWebDescHTML(strings.Join(desc, "\n")))
	cards := js.Global().Get("Array").New(0)
	for _, cardID := range item.Cards {
		if cardID == 0 {
			continue
		}
		if name, ok := ctx.Resources.ItemDisplayName(int(cardID), true); ok {
			cards.Call("push", name)
		}
	}
	obj.Set("cards", cards)
	// Card slots footer: the game shows 4 slot frames — filled ones for
	// slots the item has (empty_card_slot icon, or the card's own icon +
	// name when equipped), hatched ones when the item has fewer slots.
	slotCount, _ := ctx.Resources.ItemSlotCount(int(item.ItemID))
	obj.Set("slotCount", slotCount)
	slots := js.Global().Get("Array").New(4)
	for i := 0; i < 4; i++ {
		entry := js.Global().Get("Object").New()
		cardID := uint16(0)
		if i < len(item.Cards) {
			cardID = item.Cards[i]
			if cardID == 0x00ff || cardID == 0x00fe || cardID == 0xff00 {
				cardID = 0
			}
		}
		if i < slotCount {
			if cardID != 0 {
				entry.Set("state", "card")
				if name, ok := ctx.Resources.ItemDisplayName(int(cardID), true); ok {
					entry.Set("name", name)
				}
				if cardRes, ok := ctx.Resources.ItemResourceName(int(cardID), true); ok {
					entry.Set("icon", collectionWebIcon(ctx, cardRes, cardID, true))
				}
			} else {
				entry.Set("state", "empty")
			}
		} else {
			entry.Set("state", "none")
		}
		slots.SetIndex(i, entry)
	}
	obj.Set("slots", slots)
	// The illustration (collection art, the big picture the canvas window
	// showed at 75x100) rides along as a cached data URL.
	if resourceName, ok := ctx.Resources.ItemResourceName(int(item.ItemID), item.Identified); ok {
		obj.Set("icon", collectionWebIcon(ctx, resourceName, item.ItemID, item.Identified))
	} else {
		obj.Set("icon", "")
	}
	fn.Invoke(obj)
}


// npcDialogWebEnabled reports whether the page provides the DOM NPC dialog.
func npcDialogWebEnabled() bool {
	return js.Global().Get("goroNPCDialogSync").Type() == js.TypeFunction
}

// npcDialogWebSync pushes the NPC dialog state: lines with ^RRGGBB runs
// converted to HTML, the current action (next/close/menu/input), and the
// menu options.
func (d *NPCDialog) npcDialogWebSync() {
	fn := js.Global().Get("goroNPCDialogSync")
	if fn.Type() != js.TypeFunction {
		return
	}
	obj := js.Global().Get("Object").New()
	if !d.open {
		obj.Set("open", false)
		fn.Invoke(obj)
		return
	}
	obj.Set("open", true)
	lines := js.Global().Get("Array").New(len(d.lines))
	for i, line := range d.lines {
		lines.SetIndex(i, npcDialogWebLineHTML(line))
	}
	obj.Set("lines", lines)
	switch d.action {
	case npcDialogActionNext:
		obj.Set("action", "next")
	case npcDialogActionClose:
		obj.Set("action", "close")
	case npcDialogActionMenu:
		obj.Set("action", "menu")
		options := js.Global().Get("Array").New(len(d.options))
		for i, opt := range d.options {
			options.SetIndex(i, npcDialogWebLineHTML(opt))
		}
		obj.Set("options", options)
	case npcDialogActionNumberInput:
		obj.Set("action", "number")
	case npcDialogActionStringInput:
		obj.Set("action", "string")
	default:
		obj.Set("action", "none")
	}
	fn.Invoke(obj)
}

// npcDialogWebLineHTML converts one dialog line's ^RRGGBB codes into HTML
// spans. ^000000 resets to the window's base color.
func npcDialogWebLineHTML(text string) string {
	runes := []rune(text)
	var b strings.Builder
	const base = "#2a2622"
	active := false
	for i := 0; i < len(runes); i++ {
		if runes[i] == '^' && i+6 < len(runes) && isHexRunes(runes[i+1:i+7]) {
			code := string(runes[i+1 : i+7])
			if active {
				b.WriteString("</span>")
			}
			color := "#" + code
			if strings.EqualFold(code, "000000") {
				color = base
			}
			b.WriteString(`<span style="color:` + color + `">`)
			active = true
			i += 6
			continue
		}
		b.WriteRune(runes[i])
	}
	if active {
		b.WriteString("</span>")
	}
	return b.String()
}

// hotbarWebEnabled reports whether the page provides the DOM hotbar.
func hotbarWebEnabled() bool {
	return js.Global().Get("goroHotbarSync").Type() == js.TypeFunction
}

// hotbarWebIconCache memoizes PNG data URLs per icon key.
var hotbarWebIconCache = map[string]string{}

// collectionWebIcon resolves the item's collection illustration (the big
// art from data/texture/유저인터페이스/collection) as a cached data URL.
func collectionWebIcon(ctx Context, resourceName string, itemID uint16, identified bool) string {
	key := fmt.Sprintf("collection:%d:%t", itemID, identified)
	if url, ok := hotbarWebIconCache[key]; ok {
		return url
	}
	img, _, err := res.LoadImage(ctx.Resources, res.ItemCollectionTextureCandidates(resourceName))
	if err != nil {
		return ""
	}
	return hotbarWebIcon(key, img)
}

// hotbarWebIconKeyOnly resolves an item icon by resource name and returns
// its cached data URL.
func hotbarWebIconKeyOnly(ctx Context, resourceName string, itemID uint16, identified bool) string {
	key := fmt.Sprintf("item:%d:%t", itemID, identified)
	if url, ok := hotbarWebIconCache[key]; ok {
		return url
	}
	img, _, err := res.LoadImage(ctx.Resources, res.ItemIconTextureCandidates(resourceName))
	if err != nil {
		return ""
	}
	return hotbarWebIcon(key, img)
}

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
		entry.Set("key", shortcutKeyLabels[i])
		slots.SetIndex(i, entry)
	}
	obj.Set("slots", slots)
	fn.Invoke(obj)
}

// hotbarWebKey summarizes slot contents for change detection.
func (b *ShortcutBar) hotbarWebKey(ctx Context) string {
	var sb strings.Builder
	x, y := b.bounds(ctx)
	fmt.Fprintf(&sb, "pos=%d,%d|rows=%d|", x, y, b.visibleRowCount())
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
