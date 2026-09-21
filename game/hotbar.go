package game

import (
	"fmt"
	"image/color"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
)

// The handheld hotbar: nine slots holding items or skills. X on an item in
// the inventory's Item/Equip tabs (or on a skill in the skill window) fills
// the next slot, wrapping 1→9→1 and overwriting onward. L2/R2 step the
// active slot (works whether the bar is shown or hidden), B on the plain
// screen uses the active slot, and the MENU entry toggles the bar view.
// While hidden, the active slot's icon rides next to the vitals HUD.

const (
	hotbarSlotCount   = 9
	hotbarCellSize    = 44
	hotbarIconSize    = 28
	hotbarWindowPad   = 6
	hotbarRightMargin = 10
)

type hotbarEntryKind uint8

const (
	hotbarEmpty hotbarEntryKind = iota
	hotbarItem
	hotbarSkill
)

type hotbarEntry struct {
	kind      hotbarEntryKind
	itemID    uint16
	itemIndex uint16 // inventory index captured at add time
	skillID   uint16
}

type hotbar struct {
	entries [hotbarSlotCount]hotbarEntry
	active  int
	fill    int
	shown   bool
}

// addItem fills the next slot (wrapping, overwriting) with the given
// inventory stack. A successful add also shows the bar: the fill happens
// from inside the inventory window, and without the bar popping into view
// there is no way to tell the press landed.
func (h *hotbar) addItem(item session.InventoryItem) {
	if item.ItemID == 0 {
		return
	}
	h.entries[h.fill] = hotbarEntry{kind: hotbarItem, itemID: item.ItemID, itemIndex: item.Index}
	h.fill = (h.fill + 1) % hotbarSlotCount
	h.shown = true
}

// addSkill fills the next slot with the given skill (and shows the bar,
// like addItem).
func (h *hotbar) addSkill(skillID uint16) {
	if skillID == 0 {
		return
	}
	h.entries[h.fill] = hotbarEntry{kind: hotbarSkill, skillID: skillID}
	h.fill = (h.fill + 1) % hotbarSlotCount
	h.shown = true
}

// cycle moves the active slot (dir < 0 = L2/left, dir > 0 = R2/right),
// skipping nothing — every slot is addressable, empty or not.
func (h *hotbar) cycle(dir int) {
	if dir == 0 {
		return
	}
	h.active = ((h.active+dir)%hotbarSlotCount + hotbarSlotCount) % hotbarSlotCount
}

// activeEntry reports the currently selected slot.
func (h *hotbar) activeEntry() hotbarEntry {
	return h.entries[h.active]
}

// resolveItem finds the live inventory stack behind an item slot: by the
// index captured at add time first, then by item ID when stacks shifted.
func (h *hotbar) resolveItem(s *session.Session, entry hotbarEntry) (session.InventoryItem, int, bool) {
	if entry.kind != hotbarItem || s == nil {
		return session.InventoryItem{}, -1, false
	}
	for i := range s.Inventory.Items {
		if s.Inventory.Items[i].Index == entry.itemIndex && s.Inventory.Items[i].ItemID == entry.itemID {
			return s.Inventory.Items[i], i, true
		}
	}
	for i := range s.Inventory.Items {
		if s.Inventory.Items[i].ItemID == entry.itemID {
			return s.Inventory.Items[i], i, true
		}
	}
	return session.InventoryItem{}, -1, false
}

var (
	hotbarPanelColor  = color.RGBA{R: 14, G: 18, B: 24, A: 190}
	hotbarBorderColor = color.RGBA{R: 160, G: 175, B: 195, A: 90}
	hotbarActiveColor = color.RGBA{R: 214, G: 178, B: 92, A: 255}
	hotbarEmptyColor  = color.RGBA{R: 60, G: 66, B: 78, A: 120}
	hotbarTextColor   = color.RGBA{R: 244, G: 248, B: 252, A: 255}
	hotbarOutline     = color.RGBA{R: 10, G: 12, B: 16, A: 210}
)

// draw renders the hotbar as a vertical column docked to the top-right
// corner — the top, because the bottom-right corner belongs to the FPS/
// memory stack and a vertically centered bar ran into it. Icons reuse
// the WorldMode's cached item/skill painters, so nothing here allocates
// per frame.
func (h *hotbar) draw(screen *render.Frame, ctx client.Context, m *WorldMode) {
	if screen == nil || !h.shown {
		return
	}
	bounds := screen.Bounds()
	width := hotbarCellSize + hotbarWindowPad*2
	height := hotbarSlotCount*hotbarCellSize + hotbarWindowPad*2
	x := bounds.Dx() - width - hotbarRightMargin
	y := hotbarRightMargin
	render.DrawRect(screen, float64(x), float64(y), float64(width), float64(height), hotbarPanelColor)
	render.DrawRect(screen, float64(x), float64(y), 1, float64(height), hotbarBorderColor)
	render.DrawRect(screen, float64(x+width-1), float64(y), 1, float64(height), hotbarBorderColor)

	for i := 0; i < hotbarSlotCount; i++ {
		h.drawCell(screen, ctx, m, x+hotbarWindowPad, y+hotbarWindowPad+i*hotbarCellSize, i)
	}
}

// drawClosedBadge renders the active slot's icon next to the vitals HUD
// while the bar itself is hidden.
func (h *hotbar) drawClosedBadge(screen *render.Frame, ctx client.Context, m *WorldMode) {
	if screen == nil || h.shown {
		return
	}
	entry := h.activeEntry()
	if entry.kind == hotbarEmpty {
		return
	}
	x := 10 + miniHUDWidthOf() + 6
	y := 10
	const badge = hotbarCellSize - 4
	render.DrawRect(screen, float64(x-2), float64(y-2), badge, badge, hotbarPanelColor)
	render.DrawRect(screen, float64(x-2), float64(y-2), badge, 1, hotbarActiveColor)
	render.DrawRect(screen, float64(x-2), float64(y+badge-2), badge, 1, hotbarActiveColor)
	h.drawIcon(screen, ctx, m, entry, x+badge/2-2, y+badge/2-2)
	render.DrawUIOutlinedTextAt(screen, fmt.Sprintf("%d", h.active+1), float64(x)-1, float64(y)-2, hotbarTextColor, hotbarOutline)
	if item, _, ok := h.resolveItem(ctx.Session, entry); ok && entry.kind == hotbarItem && item.Amount > 1 {
		h.drawAmount(screen, float64(x)+float64(badge)/2-2, float64(y)+float64(badge)-14, item.Amount)
	}
}

func (h *hotbar) drawCell(screen *render.Frame, ctx client.Context, m *WorldMode, x, y, slot int) {
	entry := h.entries[slot]
	cellColor := hotbarEmptyColor
	if entry.kind != hotbarEmpty {
		cellColor = color.RGBA{R: 40, G: 46, B: 58, A: 190}
	}
	render.DrawRect(screen, float64(x)+1, float64(y)+1, hotbarCellSize-2, hotbarCellSize-2, cellColor)
	if slot == h.active {
		render.DrawRect(screen, float64(x)+1, float64(y)+1, hotbarCellSize-2, 2, hotbarActiveColor)
		render.DrawRect(screen, float64(x)+1, float64(y+hotbarCellSize)-3, hotbarCellSize-2, 2, hotbarActiveColor)
		render.DrawRect(screen, float64(x)+1, float64(y)+1, 2, hotbarCellSize-2, hotbarActiveColor)
		render.DrawRect(screen, float64(x+hotbarCellSize)-3, float64(y)+1, 2, hotbarCellSize-2, hotbarActiveColor)
	}
	if entry.kind == hotbarEmpty {
		return
	}
	h.drawIcon(screen, ctx, m, entry, x+hotbarCellSize/2, y+hotbarCellSize/2)
	render.DrawUIOutlinedTextAt(screen, fmt.Sprintf("%d", slot+1), float64(x)+3, float64(y)+2, color.RGBA{R: 190, G: 200, B: 215, A: 220}, hotbarOutline)
	if item, _, ok := h.resolveItem(ctx.Session, entry); ok && entry.kind == hotbarItem && item.Amount > 1 {
		h.drawAmount(screen, float64(x)+float64(hotbarCellSize)/2, float64(y+hotbarCellSize)-13, item.Amount)
	}
}

// drawAmount renders the stack count centered at the bottom edge of a
// cell; stacks of one stay unlabeled, like the inventory grid.
func (h *hotbar) drawAmount(screen *render.Frame, centerX, y float64, amount int) {
	render.DrawCenteredUIOutlinedTextAt(screen, hotbarAmountText(amount), centerX, y,
		color.RGBA{R: 250, G: 250, B: 255, A: 255}, hotbarOutline)
}

// hotbarAmountText compacts large stacks for the narrow cell ("30000" ->
// "30k"); the cell is 44px wide and the 12px font fits roughly five
// digits.
func hotbarAmountText(amount int) string {
	if amount >= 10000 {
		return fmt.Sprintf("%dk", amount/1000)
	}
	return fmt.Sprintf("%d", amount)
}

// drawIcon renders the entry's icon centered on (cx, cy) — the cell's
// middle for the bar, the badge's middle when hidden.
func (h *hotbar) drawIcon(screen *render.Frame, ctx client.Context, m *WorldMode, entry hotbarEntry, cx, cy int) {
	switch entry.kind {
	case hotbarItem:
		if item, _, ok := h.resolveItem(ctx.Session, entry); ok {
			m.drawInventoryItemIcon(screen, ctx.Resources, item, cx-inventoryIconSize/2, cy-inventoryIconSize/2)
			return
		}
		// Stale slot: dim placeholder.
		render.DrawRect(screen, float64(cx-hotbarIconSize/2), float64(cy-hotbarIconSize/2), hotbarIconSize, hotbarIconSize, hotbarEmptyColor)
	case hotbarSkill:
		for _, skill := range ctx.Session.Skills.List {
			if skill.ID == entry.skillID {
				m.drawSkillIcon(screen, ctx.Resources, skill, cx-hotbarIconSize/2, cy-hotbarIconSize/2, hotbarIconSize)
				return
			}
		}
		render.DrawRect(screen, float64(cx-hotbarIconSize/2), float64(cy-hotbarIconSize/2), hotbarIconSize, hotbarIconSize, hotbarEmptyColor)
	}
}

// miniHUDWidthOf mirrors the ui package's mini HUD panel width so the
// closed badge sits beside it without an import cycle.
func miniHUDWidthOf() int {
	return 168
}

// useActive triggers the active slot on the plain screen (B button).
// Every dead end explains itself with a notice — a silent no-op reads as
// "the hotbar is broken" on the handheld.
func (m *WorldMode) useHotbarActive(ctx client.Context) {
	entry := m.ui.hotbar.activeEntry()
	switch entry.kind {
	case hotbarEmpty:
		render.ShowScreenNotice(fmt.Sprintf("Hotbar slot %d is empty", m.ui.hotbar.active+1))
	case hotbarItem:
		if _, listIndex, ok := m.ui.hotbar.resolveItem(ctx.Session, entry); ok {
			m.ui.inventoryBag.UseByIndex(ctx, listIndex)
		} else {
			render.ShowScreenNotice("Hotbar: item no longer in inventory")
		}
	case hotbarSkill:
		for _, skill := range ctx.Session.Skills.List {
			if skill.ID == entry.skillID {
				if err := m.UseShortcutSkill(ctx, skill); err != nil {
					m.ui.console.AddErrorMessage("%s", err.Error())
				}
				return
			}
		}
		render.ShowScreenNotice("Hotbar: skill not learned")
	}
}

// drawHotbar draws both hotbar presentations; called from DrawUIOverlay.
func (m *WorldMode) drawHotbar(screen *render.Frame, ctx client.Context, now time.Time) {
	m.ui.hotbar.draw(screen, ctx, m)
	m.ui.hotbar.drawClosedBadge(screen, ctx, m)
}

// syncFromSession fills the nine slots from the server-saved shortcut
// list's first row (the 27-slot protocol list is three rows of nine; the
// handheld bar is row one). Saved item slots carry no inventory index —
// resolveItem's ID fallback covers that.
func (h *hotbar) syncFromSession(s *session.Session) {
	if s == nil {
		return
	}
	for i := 0; i < hotbarSlotCount; i++ {
		h.entries[i] = hotbarEntry{}
		if i >= len(s.Hotkeys.Slots) {
			continue
		}
		slot := s.Hotkeys.Slots[i]
		switch slot.Type {
		case network.HotkeyTypeItem:
			if slot.ID != 0 {
				h.entries[i] = hotbarEntry{kind: hotbarItem, itemID: uint16(slot.ID)}
			}
		case network.HotkeyTypeSkill:
			if slot.ID != 0 {
				h.entries[i] = hotbarEntry{kind: hotbarSkill, skillID: uint16(slot.ID)}
			}
		}
	}
}

// pushHotbarSlot persists one bar slot to the server's saved shortcut list
// (CZ_SHORTCUT_KEY_CHANGE — rAthena keeps it in the hotkey table, so the
// setup survives relogin) and mirrors it into the session so the desktop
// shortcut bar agrees.
func (m *WorldMode) pushHotbarSlot(ctx client.Context, slot int) {
	if slot < 0 || slot >= hotbarSlotCount {
		return
	}
	entry := m.ui.hotbar.entries[slot]
	saved := network.HotkeySlot{} // type 0 + id 0 = empty
	switch entry.kind {
	case hotbarItem:
		saved = network.HotkeySlot{Type: network.HotkeyTypeItem, ID: uint32(entry.itemID)}
	case hotbarSkill:
		level := uint16(0)
		if ctx.Session != nil {
			for _, skill := range ctx.Session.Skills.List {
				if skill.ID == entry.skillID {
					level = uint16(maxInt(0, int(skill.Level)))
					break
				}
			}
		}
		saved = network.HotkeySlot{Type: network.HotkeyTypeSkill, ID: uint32(entry.skillID), Level: level}
	}
	if ctx.Network != nil {
		if err := ctx.Network.SendHotkey(uint16(slot), saved); err != nil {
			glog.Warnf("hotbar save failed slot=%d: %v", slot+1, err)
		}
	}
	if ctx.Session != nil {
		s := ctx.Session
		if len(s.Hotkeys.Slots) <= slot {
			next := make([]session.HotkeySlot, slot+1)
			copy(next, s.Hotkeys.Slots)
			s.Hotkeys.Slots = next
		}
		s.Hotkeys.Slots[slot] = session.HotkeySlot{Type: saved.Type, ID: saved.ID, Level: saved.Level}
		s.Hotkeys.Loaded = true
		s.Hotkeys.Version++
	}
}

// rightInsetOf reports the right-edge strip a drawn bar occupies, so
// overlays (the map thumbnail) can shift out from under it.
func (h *hotbar) rightInsetOf() int {
	if !h.shown {
		return 0
	}
	return hotbarCellSize + hotbarWindowPad*2 + hotbarRightMargin
}
