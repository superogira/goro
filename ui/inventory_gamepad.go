package ui

import (
	"github.com/kivutar/goro/session"
)

// Handheld controls for the inventory bag, driven from game/gamepad.go the
// same way the stats window is: the d-pad moves a cell selection through the
// current tab's grid, A activates the selected item (the double-click path:
// use, equip, or start card composition), Y opens its description window,
// L1/R1 cycle the Item/Equip/Etc tabs.

// GamepadTab cycles the tab (dir < 0 = L1/previous, dir > 0 = R1/next) and
// restarts the selection, mirroring what a tab click does.
func (w *InventoryBagWindow) GamepadTab(ctx Context, dir int) {
	if !w.IsOpen() {
		return
	}
	w.hideTooltip()
	count := len(inventoryBagTabs)
	tab := w.tab
	if dir < 0 {
		tab--
	} else if dir > 0 {
		tab++
	}
	w.tab = ((tab % count) + count) % count
	w.ensureScrollSignal().Set(0)
	w.lastClickItem = 0
	w.gamepadSelected = 0
	w.refresh(ctx, w.itemInfo)
}

// GamepadNavigate moves the cell selection: dx steps columns, dy steps rows
// (the grid is row-major). The selection is clamped to the current tab's
// items and scrolled into view.
func (w *InventoryBagWindow) GamepadNavigate(ctx Context, dx, dy int) {
	if !w.IsOpen() || (dx == 0 && dy == 0) {
		return
	}
	items := w.tabItems(ctx.Session)
	if len(items) == 0 {
		return
	}
	next := w.gamepadSelected + dx + dy*inventoryBagCols
	if next < 0 {
		next = 0
	}
	if next >= len(items) {
		next = len(items) - 1
	}
	if next == w.gamepadSelected {
		return
	}
	w.gamepadSelected = next
	w.gamepadScrollToSelection()
	w.refresh(ctx, w.itemInfo)
}

// gamepadScrollToSelection keeps the selected cell inside the visible rows.
func (w *InventoryBagWindow) gamepadScrollToSelection() {
	top := float32((w.gamepadSelected / inventoryBagCols) * inventoryBagCell)
	bottom := top + inventoryBagCell
	scroll := w.ensureScrollSignal()
	value := scroll.Get()
	switch {
	case top < value:
		scroll.Set(top)
	case bottom > value+inventoryBagViewH:
		scroll.Set(bottom - inventoryBagViewH)
	}
}

// GamepadActivate uses the selected item — the same branch a double-click
// takes: cards start composition, equippables equip, usables are used.
func (w *InventoryBagWindow) GamepadActivate(ctx Context) {
	if !w.IsOpen() {
		return
	}
	items := w.tabItems(ctx.Session)
	if w.gamepadSelected < 0 || w.gamepadSelected >= len(items) {
		return
	}
	// A gamepad press must never resume a mouse drag: the stray companion
	// click can land anywhere.
	w.dragActive = false
	w.dragItem = session.InventoryItem{}
	w.activateItem(ctx, items[w.gamepadSelected])
	w.refresh(ctx, w.itemInfo)
}

// GamepadInfo opens the description window for the selected item, placed to
// the right of the inventory so the grid stays visible.
func (w *InventoryBagWindow) GamepadInfo(ctx Context) {
	if !w.IsOpen() || w.itemInfo == nil {
		return
	}
	items := w.tabItems(ctx.Session)
	if w.gamepadSelected < 0 || w.gamepadSelected >= len(items) {
		return
	}
	w.itemInfo.openItem(ctx, items[w.gamepadSelected], w.x+inventoryBagWidth-30, w.y+40)
}

// GamepadSelectedItem reports the item under the handheld cursor.
func (w *InventoryBagWindow) GamepadSelectedItem(ctx Context) (session.InventoryItem, bool) {
	if !w.IsOpen() {
		return session.InventoryItem{}, false
	}
	items := w.tabItems(ctx.Session)
	if w.gamepadSelected < 0 || w.gamepadSelected >= len(items) {
		return session.InventoryItem{}, false
	}
	return items[w.gamepadSelected], true
}

// GamepadTabKind reports the current tab constant (Item/Equip/Etc).
func (w *InventoryBagWindow) GamepadTabKind() int {
	return w.tab
}

// GamepadTabAllowsHotbar reports whether the current tab may add items to
// the handheld hotbar (Item and Equip only, per the hotbar design).
func (w *InventoryBagWindow) GamepadTabAllowsHotbar() bool {
	return w.tab == inventoryBagTabItem || w.tab == inventoryBagTabEquip
}

// UseByIndex activates the inventory item at the given list position — the
// hotbar's use path (same double-click semantics as clicking the item).
func (w *InventoryBagWindow) UseByIndex(ctx Context, index int) {
	if !w.IsOpen() && index < 0 {
		return
	}
	if index < 0 || index >= len(ctx.Session.Inventory.Items) {
		return
	}
	w.dragActive = false
	w.dragItem = session.InventoryItem{}
	w.activateItem(ctx, ctx.Session.Inventory.Items[index])
	if w.IsOpen() {
		w.refresh(ctx, w.itemInfo)
	}
}
