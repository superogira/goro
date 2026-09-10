package ui

import (
	"fmt"
)

// Equipment window DOM state. The page renders the three slot columns with
// the character preview between them, plus the Show Equip / Cart / Peco
// footer. Icons and the preview are cache keys resolved to data URLs by the
// wasm-side marshaler.

const (
	equipmentWebSideLeft   = 0
	equipmentWebSideRight  = 1
	equipmentWebSideCenter = 2
)

type equipmentWebSlot struct {
	Label    string
	Location uint16
	Side     int
	Row      int
	Visible  bool
	HasItem  bool
	Icon     string
	Name     string
	Refine   int
}

type equipmentWebState struct {
	Open        bool
	ShowEquip   bool
	HasCart     bool
	RemoveLabel string
	PreviewKey  string
	Slots       []equipmentWebSlot
}

// equipmentWebKey summarizes everything the DOM equipment window shows.
func (w *EquipmentWindow) equipmentWebKey(ctx Context) string {
	return fmt.Sprintf("%t|%s|%t|%t", w.webOpen, equipmentSnapshot(ctx.Session), inventoryBagHasCart(ctx), equipmentHasPeco(ctx))
}

// buildEquipmentWebState assembles the DOM snapshot. Pure data; icons stay
// cache keys and are resolved on the wasm side.
func (w *EquipmentWindow) buildEquipmentWebState(ctx Context) equipmentWebState {
	state := equipmentWebState{
		Open:        w.webOpen,
		ShowEquip:   ctx.Session != nil && ctx.Session.ShowEquip,
		HasCart:     inventoryBagHasCart(ctx),
		RemoveLabel: equipmentRemoveOptionLabel(ctx),
		PreviewKey:  "eqpreview:" + equipmentSnapshot(ctx.Session),
	}
	sideOf := map[equipmentSlotSide]int{
		equipmentSlotLeft:   equipmentWebSideLeft,
		equipmentSlotRight:  equipmentWebSideRight,
		equipmentSlotCenter: equipmentWebSideCenter,
	}
	for _, slot := range equipmentSlots {
		webSlot := equipmentWebSlot{
			Label:    slot.label,
			Location: slot.location,
			Side:     sideOf[slot.side],
			Row:      slot.row,
			Visible:  equipmentSlotVisible(ctx.Session, slot),
		}
		if item, ok := equippedItemForSlot(ctx.Session, slot.location); ok {
			webSlot.HasItem = true
			webSlot.Icon = fmt.Sprintf("item:%d:%t", item.ItemID, item.Identified)
			webSlot.Name = inventoryItemDisplayName(ctx.Resources, item)
			webSlot.Refine = int(item.Refine)
		}
		state.Slots = append(state.Slots, webSlot)
	}
	return state
}
