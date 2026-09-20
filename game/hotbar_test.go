package game

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestHotbarSyncFromSessionMapsServerRowOne(t *testing.T) {
	// The server's 27-slot saved list is three rows of nine; the handheld
	// bar rides row one, items by ID and skills by ID.
	var h hotbar
	s := &session.Session{}
	s.Hotkeys.Slots = make([]session.HotkeySlot, network.HotkeyListSlots2008)
	s.Hotkeys.Slots[0] = session.HotkeySlot{Type: network.HotkeyTypeItem, ID: 501}
	s.Hotkeys.Slots[1] = session.HotkeySlot{Type: network.HotkeyTypeSkill, ID: 7, Level: 3}
	s.Hotkeys.Slots[2] = session.HotkeySlot{Type: network.HotkeyTypeItem, ID: 0} // saved empty
	s.Hotkeys.Slots[9] = session.HotkeySlot{Type: network.HotkeyTypeItem, ID: 999}

	h.entries[2] = hotbarEntry{kind: hotbarItem, itemID: 123} // local slot the server clears
	h.syncFromSession(s)

	if h.entries[0].kind != hotbarItem || h.entries[0].itemID != 501 {
		t.Fatalf("slot 1 = %+v, want item 501", h.entries[0])
	}
	if h.entries[1].kind != hotbarSkill || h.entries[1].skillID != 7 {
		t.Fatalf("slot 2 = %+v, want skill 7", h.entries[1])
	}
	if h.entries[2].kind != hotbarEmpty {
		t.Fatalf("slot 3 = %+v, want cleared", h.entries[2])
	}
	if h.entries[3].kind != hotbarEmpty {
		t.Fatalf("slot 4 = %+v, want empty", h.entries[3])
	}
	// Row two stays off the handheld bar.
	for i := 3; i < hotbarSlotCount; i++ {
		if h.entries[i].itemID == 999 {
			t.Fatalf("slot %d pulled row-two data", i+1)
		}
	}
}

func TestHotbarPushMirrorsSession(t *testing.T) {
	// Pushing a slot mirrors it into the session list (the desktop
	// shortcut bar reads the same store) and stamps the skill level.
	m := &WorldMode{}
	m.ui.hotbar.entries[0] = hotbarEntry{kind: hotbarSkill, skillID: 7}
	s := &session.Session{Skills: session.Skills{List: []session.Skill{{ID: 7, Level: 5}}}}
	ctx := client.Context{Session: s}

	m.pushHotbarSlot(ctx, 0)

	if len(s.Hotkeys.Slots) < 1 {
		t.Fatal("push did not extend the session slot list")
	}
	got := s.Hotkeys.Slots[0]
	if got.Type != network.HotkeyTypeSkill || got.ID != 7 || got.Level != 5 {
		t.Fatalf("session slot = %+v, want skill 7 level 5", got)
	}
	if !s.Hotkeys.Loaded || s.Hotkeys.Version == 0 {
		t.Fatal("push must mark the hotkey store loaded and bump its version")
	}

	// An empty slot saves as the canonical empty hotkey.
	m.ui.hotbar.entries[0] = hotbarEntry{}
	m.pushHotbarSlot(ctx, 0)
	if got := s.Hotkeys.Slots[0]; got.Type != 0 || got.ID != 0 {
		t.Fatalf("empty push = %+v, want zeroed", got)
	}
}

func TestHotbarRightInsetTracksVisibility(t *testing.T) {
	var h hotbar
	if h.rightInsetOf() != 0 {
		t.Fatal("hidden bar must not claim a right inset")
	}
	h.shown = true
	if h.rightInsetOf() != hotbarCellSize+hotbarWindowPad*2+hotbarRightMargin {
		t.Fatalf("inset = %d, want the drawn strip width", h.rightInsetOf())
	}
}

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
	if h.shown {
		t.Fatal("an ignored (empty) add must not force the bar visible")
	}
}

func TestHotbarAddShowsBar(t *testing.T) {
	// The add happens from inside the inventory window; popping the bar
	// into view is the only immediate confirmation the press landed.
	var h hotbar
	h.addItem(session.InventoryItem{Index: 1, ItemID: 501})
	if !h.shown {
		t.Fatal("successful item add must show the bar")
	}
	h.shown = false
	h.addSkill(7)
	if !h.shown {
		t.Fatal("successful skill add must show the bar")
	}
}

func TestHotbarAmountText(t *testing.T) {
	cases := []struct {
		amount int
		want   string
	}{
		{1, "1"},
		{300, "300"},
		{9999, "9999"},
		{10000, "10k"},
		{30000, "30k"},
	}
	for _, c := range cases {
		if got := hotbarAmountText(c.amount); got != c.want {
			t.Fatalf("hotbarAmountText(%d) = %q, want %q", c.amount, got, c.want)
		}
	}
}
