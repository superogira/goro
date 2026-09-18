package game

import (
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

// TestItemInfoOpenCostTimes opens the item description window for a card-slot
// weapon against the real GRF and reports where the open time goes. Run with:
//
//	go test ./game -run TestItemInfoOpenCostTimes -v
//
// It exists to diagnose on-device hitches when opening card-slot equipment
// details; there is no pass/fail, only logged timings.
func TestItemInfoOpenCostTimes(t *testing.T) {
	manager, err := res.NewManager("../dist")
	if err != nil {
		t.Skipf("dist resources unavailable: %v", err)
	}
	mode := NewWorldMode()
	item := session.InventoryItem{
		Index:       1,
		ItemID:      1201, // Knife, 4 slots
		Type:        db.ItemTypeWeapon,
		Location:    db.EquipWeapon,
		Identified:  true,
		Equip:       true,
		Equipped:    true,
		Cards:       [4]uint16{4001, 4002, 4003, 4004},
	}
	sess := session.New()
	sess.Inventory.Items = []session.InventoryItem{item}
	ctx := client.Context{Resources: manager, Session: sess, ScreenW: 1280, ScreenH: 720}

	start := time.Now()
	slots, ok := manager.ItemSlotCount(int(item.ItemID))
	if !ok {
		t.Fatalf("no slot count for knife")
	}
	t.Logf("ItemSlotCount (first, loads all item tables): %v slots=%d", time.Since(start), slots)

	// First open of the equipment window + description for the slotted knife.
	start = time.Now()
	mode.ui.equipmentWindow.Toggle(ctx)
	t.Logf("equipment window toggle: %v", time.Since(start))

	start = time.Now()
	mode.ui.equipmentWindow.Rebind(ctx, &mode.ui.itemWindows, nil, mode)
	mode.ui.equipmentWindow.GamepadInfo(ctx)
	t.Logf("first GamepadInfo (opens description): %v", time.Since(start))

	start = time.Now()
	mode.ui.itemWindows.Update(ctx, mode)
	t.Logf("first itemWindows Update (illustration raster): %v", time.Since(start))

	for i := 0; i < 3; i++ {
		start = time.Now()
		mode.ui.equipmentWindow.GamepadInfo(ctx)
		mode.ui.itemWindows.Update(ctx, mode)
		t.Logf("repeat open #%d (stacking window): %v", i+2, time.Since(start))
	}

	// Same open through a fresh mode, the way a map change rebuilds caches.
	fresh := NewWorldMode()
	fresh.ui.equipmentWindow.Toggle(ctx)
	fresh.ui.equipmentWindow.Rebind(ctx, &fresh.ui.itemWindows, nil, fresh)
	start = time.Now()
	fresh.ui.equipmentWindow.GamepadInfo(ctx)
	fresh.ui.itemWindows.Update(ctx, fresh)
	t.Logf("open after mode rebuild (caches cold, tables warm): %v", time.Since(start))
}
