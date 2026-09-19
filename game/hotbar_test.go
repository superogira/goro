package game

import (
	"testing"

	"github.com/kivutar/goro/session"
)

func TestHotbarFillsInOrderAndWraps(t *testing.T) {
	var h hotbar
	for i := 0; i < 12; i++ {
		h.addItem(session.InventoryItem{Index: uint16(i), ItemID: 500 + uint16(i)})
	}
	// First nine fills are slots 0..8; the next three wrap and overwrite
	// slots 0..2 with the newer items.
	for i := 3; i < hotbarSlotCount; i++ {
		if h.entries[i].kind != hotbarItem || h.entries[i].itemID != 500+uint16(i) {
			t.Fatalf("slot %d = %+v, want item %d", i, h.entries[i], 500+i)
		}
	}
	for i := 0; i < 3; i++ {
		if h.entries[i].itemID != 500+uint16(hotbarSlotCount+i) {
			t.Fatalf("wrapped slot %d = %+v, want item %d", i, h.entries[i], 500+hotbarSlotCount+i)
		}
	}
	if h.fill != 3 {
		t.Fatalf("fill cursor = %d, want 3", h.fill)
	}
}

func TestHotbarCycleWraps(t *testing.T) {
	var h hotbar
	h.cycle(1)
	if h.active != 1 {
		t.Fatalf("after right: active = %d, want 1", h.active)
	}
	h.active = 0
	h.cycle(-1)
	if h.active != hotbarSlotCount-1 {
		t.Fatalf("wrap left: active = %d, want %d", h.active, hotbarSlotCount-1)
	}
	h.active = hotbarSlotCount - 1
	h.cycle(1)
	if h.active != 0 {
		t.Fatalf("wrap right: active = %d, want 0", h.active)
	}
}

func TestHotbarResolveItemFollowsIndexThenID(t *testing.T) {
	var h hotbar
	h.addItem(session.InventoryItem{Index: 7, ItemID: 501})
	entry := h.activeEntry()
	if entry.itemIndex != 7 || entry.itemID != 501 {
		t.Fatalf("entry = %+v", entry)
	}
	s := &session.Session{Inventory: session.Inventory{Items: []session.InventoryItem{
		{Index: 7, ItemID: 501},
	}}}
	if _, i, ok := h.resolveItem(s, entry); !ok || i != 0 {
		t.Fatalf("resolve by index = %d %v", i, ok)
	}
	// Index moved (stack consumed and restocked elsewhere): fall back to ID.
	s.Inventory.Items = []session.InventoryItem{{Index: 9, ItemID: 501}}
	if item, _, ok := h.resolveItem(s, entry); !ok || item.ItemID != 501 {
		t.Fatalf("resolve by id = %+v %v", item, ok)
	}
	// Gone entirely: no resolution.
	s.Inventory.Items = nil
	if _, _, ok := h.resolveItem(s, entry); ok {
		t.Fatal("resolve should fail for a removed item")
	}
}

func TestHotbarAddIgnoresEmpty(t *testing.T) {
	var h hotbar
	h.addItem(session.InventoryItem{})
	h.addSkill(0)
	if h.entries[0].kind != hotbarEmpty || h.fill != 0 {
		t.Fatalf("empty adds changed state: %+v fill=%d", h.entries[0], h.fill)
	}
}
