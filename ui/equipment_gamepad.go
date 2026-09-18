package ui

import (
	"time"

	"github.com/kivutar/goro/session"
)

// Handheld controls for the equipment window, driven from game/gamepad.go the
// same way the inventory is: the d-pad walks the equip slots laid out as a
// three-column grid (left column rows 0-4, center ammo at row 1, right column
// rows 0-4), A takes the equipped item off, Y opens its description window.
// Card slots inside an open description reuse the shared item window group
// handling (d-pad moves the slot row, X opens the selected card).

// GamepadNavigate moves the slot selection across the window's column/row
// grid, skipping positions no visible slot occupies (the ammo slot only
// exists for ammo-using classes).
func (w *EquipmentWindow) GamepadNavigate(ctx Context, dx, dy int) {
	if !w.IsOpen() || (dx == 0 && dy == 0) {
		return
	}
	grid := equipmentSlotGrid(ctx)
	col, row, ok := equipmentSlotGridPos(w.gamepadSelected)
	if !ok {
		return
	}
	for {
		if dx != 0 {
			col += dx
		} else {
			row += dy
		}
		if col < 0 || col > 2 || row < 0 || row > 4 {
			return
		}
		if next, ok := grid[[2]int{col, row}]; ok {
			w.gamepadSelected = next
			w.refresh(ctx)
			return
		}
	}
}

// GamepadActivate takes the item in the selected slot off. One press is one
// takeoff: the gamepad must not arm the mouse double-click guard.
func (w *EquipmentWindow) GamepadActivate(ctx Context) {
	if !w.IsOpen() {
		return
	}
	item, hasItem := w.gamepadSelectedItem(ctx)
	if !hasItem || ctx.Network == nil {
		return
	}
	w.lastClickItem = 0
	w.lastClickAt = time.Time{}
	_ = ctx.Network.SendTakeoffEquip(item.Index)
}

// GamepadInfo opens the description window for the item in the selected slot,
// placed to the right of the equipment window so the slots stay visible.
func (w *EquipmentWindow) GamepadInfo(ctx Context) {
	if !w.IsOpen() || w.itemInfo == nil {
		return
	}
	item, hasItem := w.gamepadSelectedItem(ctx)
	if !hasItem {
		return
	}
	w.hideTooltip()
	w.itemInfo.openItem(ctx, item, w.x+equipmentWindowWidth-30, w.y+40)
}

// gamepadSelectedItem resolves the equipped item behind the selected slot.
func (w *EquipmentWindow) gamepadSelectedItem(ctx Context) (session.InventoryItem, bool) {
	if w.gamepadSelected < 0 || w.gamepadSelected >= len(equipmentSlots) {
		return session.InventoryItem{}, false
	}
	return equippedItemForSlot(ctx.Session, equipmentSlots[w.gamepadSelected].location)
}

// refresh rebuilds the window tree. The gamepad selection outline is part of
// the tree, so every selection move rebuilds it.
func (w *EquipmentWindow) refresh(ctx Context) {
	if !w.IsOpen() {
		return
	}
	w.SetContent(w.widgetTree(ctx, w.itemInfo, w.cart))
	w.Publish(ctx)
}

// equipmentSlotGrid maps (column,row) positions of the visible equip slots to
// their index in equipmentSlots: left column 0, center column 1, right
// column 2.
func equipmentSlotGrid(ctx Context) map[[2]int]int {
	grid := make(map[[2]int]int, len(equipmentSlots))
	for i, slot := range equipmentSlots {
		if !equipmentSlotVisible(ctx.Session, slot) {
			continue
		}
		switch slot.side {
		case equipmentSlotLeft:
			grid[[2]int{0, slot.row}] = i
		case equipmentSlotCenter:
			grid[[2]int{1, slot.row}] = i
		case equipmentSlotRight:
			grid[[2]int{2, slot.row}] = i
		}
	}
	return grid
}

func equipmentSlotGridPos(index int) (int, int, bool) {
	if index < 0 || index >= len(equipmentSlots) {
		return 0, 0, false
	}
	slot := equipmentSlots[index]
	switch slot.side {
	case equipmentSlotLeft:
		return 0, slot.row, true
	case equipmentSlotCenter:
		return 1, slot.row, true
	case equipmentSlotRight:
		return 2, slot.row, true
	}
	return 0, 0, false
}
