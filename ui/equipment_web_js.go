//go:build js && wasm

package ui

import (
	"strconv"
	"strings"
	"syscall/js"
)

// equipmentWebEnabled reports whether the page provides the DOM equipment
// window.
func equipmentWebEnabled() bool {
	return js.Global().Get("goroEquipSync").Type() == js.TypeFunction
}

// webSync pushes the equipment window snapshot to the page.
func (w *EquipmentWindow) webSync(ctx Context) {
	fn := js.Global().Get("goroEquipSync")
	if fn.Type() != js.TypeFunction {
		return
	}
	state := w.buildEquipmentWebState(ctx)
	obj := js.Global().Get("Object").New()
	obj.Set("open", state.Open)
	obj.Set("showEquip", state.ShowEquip)
	obj.Set("hasCart", state.HasCart)
	obj.Set("removeLabel", state.RemoveLabel)
	obj.Set("preview", w.webPreview(state.PreviewKey, ctx))
	slots := js.Global().Get("Array").New(len(state.Slots))
	for i, slot := range state.Slots {
		entry := js.Global().Get("Object").New()
		entry.Set("label", slot.Label)
		entry.Set("loc", slot.Location)
		entry.Set("side", slot.Side)
		entry.Set("row", slot.Row)
		entry.Set("visible", slot.Visible)
		entry.Set("hasItem", slot.HasItem)
		entry.Set("icon", w.webSlotIcon(ctx, slot))
		entry.Set("name", slot.Name)
		entry.Set("refine", slot.Refine)
		slots.SetIndex(i, entry)
	}
	obj.Set("slots", slots)
	fn.Invoke(obj)
}

// webSlotIcon resolves an equipped item's icon cache key to a data URL.
func (w *EquipmentWindow) webSlotIcon(ctx Context, slot equipmentWebSlot) string {
	if !slot.HasItem {
		return ""
	}
	item, ok := equippedItemForSlot(ctx.Session, slot.Location)
	if !ok {
		return ""
	}
	return hotbarWebIcon(slot.Icon, w.itemIconImage(ctx.Resources, item))
}

// webPreview renders the paperdoll preview as a cached data URL keyed by
// the equipment snapshot, so gear changes regenerate it.
func (w *EquipmentWindow) webPreview(key string, ctx Context) string {
	if w.assets == nil {
		return ""
	}
	return hotbarWebIcon(key, w.assets.EquipmentPreviewImage(ctx, 112, 106))
}

// handleEquipWebAction services one drained "eq:" action from the page.
func (w *EquipmentWindow) handleEquipWebAction(ctx Context, action string) {
	resync := true
	switch {
	case action == "eq:close":
		w.webOpen = false
	case action == "eq:cart":
		if w.cart != nil {
			w.cart.Toggle(ctx)
		}
	case action == "eq:remove":
		w.removeOption(ctx)
	case strings.HasPrefix(action, "eq:showequip:"):
		enabled := strings.TrimPrefix(action, "eq:showequip:") == "1"
		if ctx.Session != nil {
			ctx.Session.ShowEquip = enabled
		}
		if ctx.Network != nil {
			_ = ctx.Network.SendShowEquipConfig(enabled)
		}
	case strings.HasPrefix(action, "eq:unequip:"):
		// Double-click on an equipped item takes it off.
		loc, err := strconv.ParseUint(strings.TrimPrefix(action, "eq:unequip:"), 10, 16)
		if err != nil {
			return
		}
		if item, ok := equippedItemForSlot(ctx.Session, uint16(loc)); ok {
			if ctx.Network != nil {
				_ = ctx.Network.SendTakeoffEquip(item.Index)
			}
		}
	case strings.HasPrefix(action, "eq:info:"):
		// Right-click shows the item info panel, like the inventory.
		loc, err := strconv.ParseUint(strings.TrimPrefix(action, "eq:info:"), 10, 16)
		if err != nil {
			return
		}
		if item, ok := equippedItemForSlot(ctx.Session, uint16(loc)); ok {
			itemInfoWebShow(ctx, item)
		}
	case strings.HasPrefix(action, "eq:equip:"):
		// An inventory item dragged onto the equipment window equips it.
		index, err := strconv.ParseUint(strings.TrimPrefix(action, "eq:equip:"), 10, 16)
		if err != nil {
			return
		}
		if item, ok := inventoryItemForShortcut(ctx.Session, uint16(index), 0); ok && inventoryItemCanEquip(item) {
			equipInventoryItem(ctx, item)
		}
	default:
		resync = false
	}
	if resync {
		w.webSyncKey = ""
	}
}
