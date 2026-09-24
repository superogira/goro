package game

import (
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestDerivedStatParameterPackets(t *testing.T) {
	// Wire IDs from DHXJ's VAR_* and roBrowser's StatusProperty enums.
	tests := []struct {
		name  string
		id    uint16
		field func(*session.Stats) *int
	}{
		{"ATK", 41, func(s *session.Stats) *int { return &s.Attack }},
		{"ATK bonus", 42, func(s *session.Stats) *int { return &s.AttackBonus }},
		{"MATK max", 43, func(s *session.Stats) *int { return &s.MatkMax }},
		{"MATK min", 44, func(s *session.Stats) *int { return &s.MatkMin }},
		{"DEF", 45, func(s *session.Stats) *int { return &s.Defense }},
		{"DEF bonus", 46, func(s *session.Stats) *int { return &s.DefenseBonus }},
		{"MDEF", 47, func(s *session.Stats) *int { return &s.MDefense }},
		{"MDEF bonus", 48, func(s *session.Stats) *int { return &s.MDefenseBonus }},
		{"HIT", 49, func(s *session.Stats) *int { return &s.Hit }},
		{"FLEE", 50, func(s *session.Stats) *int { return &s.Flee }},
		{"FLEE bonus", 51, func(s *session.Stats) *int { return &s.FleeBonus }},
		{"CRIT", 52, func(s *session.Stats) *int { return &s.Critical }},
		{"ASPD", 53, func(s *session.Stats) *int { return &s.ASPD }},
		{"ASPD bonus", 54, func(s *session.Stats) *int { return &s.ASPDBonus }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := session.Stats{Str: 12, StrBonus: 2, Points: 5, Attack: 10, Defense: 3, ASPD: 1500}
			ctx := client.Context{Session: &session.Session{Stats: want}}
			mode := NewWorldMode()
			for _, value := range []int32{37, -2, 0} {
				mode.handleNetworkPacket(ctx, testParameterChangePacket(tc.id, uint32(value)), time.Now())
				*tc.field(&want) = int(value)
				if ctx.Session.Stats != want {
					t.Fatalf("value %d: stats=%+v, want %+v", value, ctx.Session.Stats, want)
				}
			}
		})
	}
}

func TestInventoryAndStatsRefreshWhileChatConsumesInput(t *testing.T) {
	ctx := client.Context{
		Session: &session.Session{Vitals: session.Vitals{HP: 100}, Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 7, ItemID: 909, Type: 3, Amount: 5, Identified: true}},
		}},
		World: worldstate.New(), Input: input.NewState(),
		Network:   &network.Client{},
		UIManager: &worldModeTestUIManager{}, ScreenW: 1280, ScreenH: 720,
	}
	mode := NewWorldMode()
	mode.ui.inventoryBag.Toggle(ctx)
	mode.ui.inventoryBag.Update(ctx, nil, nil, nil, nil, nil, &mode.ui.itemWindows)
	mode.ui.statsWindow.OpenWindow(ctx)
	mode.ui.console.UpdatePresentation(ctx)
	mode.ui.console.PrepareTextInput(ctx, gpucontext.KeyA)
	if !mode.ui.console.UpdateInput(ctx) {
		t.Fatal("chat should consume input during this test")
	}
	content := func(root widget.Widget) widget.Widget {
		// Damaged overlays expose their new children after a render step.
		wctx := widget.NewContext()
		root.Layout(wctx, geometry.Loose(geometry.Sz(1280, 720)))
		root.Draw(wctx, &uitest.MockCanvas{})
		return root.Children()[0]
	}

	for _, amount := range []uint16{2, 3} {
		bagBefore := content(mode.ui.inventoryBag.Widget())
		statsBefore := content(mode.ui.statsWindow.Widget())
		data := []byte{0xaf, 0, 7, 0, byte(amount), 0}
		mode.deferredPackets = []network.Packet{
			{ID: 0x00AF, Data: data},
			testParameterChangePacket(41, uint32(amount)+10),
		}
		if next, err := mode.Update(ctx); err != nil || next != nil {
			t.Fatalf("world update: next=%T err=%v", next, err)
		}
		if content(mode.ui.inventoryBag.Widget()) == bagBefore {
			t.Fatal("drop acknowledgement left stale inventory widgets while chat was focused")
		}
		if content(mode.ui.statsWindow.Widget()) == statsBefore {
			t.Fatal("parameter update left stale status widgets while chat was focused")
		}
		if !mode.ui.console.Active() {
			t.Fatal("refresh stole chat focus")
		}
		if amount == 2 && (len(ctx.Session.Inventory.Items) != 1 || ctx.Session.Inventory.Items[0].Amount != 3) {
			t.Fatalf("partial drop inventory = %+v", ctx.Session.Inventory.Items)
		}
	}
	if len(ctx.Session.Inventory.Items) != 0 {
		t.Fatalf("full drop inventory = %+v, want empty", ctx.Session.Inventory.Items)
	}
	bagBefore := content(mode.ui.inventoryBag.Widget())
	statsBefore := content(mode.ui.statsWindow.Widget())
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if content(mode.ui.inventoryBag.Widget()) != bagBefore || content(mode.ui.statsWindow.Widget()) != statsBefore {
		t.Fatal("idle update rebuilt unchanged inventory or status widgets")
	}
	mode.ui.inventoryBag.Close()
	mode.ui.statsWindow.Close()
	mode.deferredPackets = []network.Packet{testParameterChangePacket(45, 5)}
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.ui.inventoryBag.IsOpen() || mode.ui.statsWindow.IsOpen() {
		t.Fatal("server update reopened a closed window")
	}
}

func TestRapidPickupPacketsPreserveEveryChatAmount(t *testing.T) {
	ctx := client.Context{Session: &session.Session{Inventory: session.Inventory{
		Items: []session.InventoryItem{{Index: 7, ItemID: 909, Type: 3, Amount: 20, Identified: true}},
	}}}
	mode := NewWorldMode()
	amounts := []uint16{1, 1, 3, 3, 4}
	for _, amount := range amounts {
		data := make([]byte, 23)
		binary.LittleEndian.PutUint16(data, 0x00A0)
		binary.LittleEndian.PutUint16(data[2:], 7)
		binary.LittleEndian.PutUint16(data[4:], amount)
		binary.LittleEndian.PutUint16(data[6:], 909)
		data[8], data[21] = 1, 3
		mode.handleNetworkPacket(ctx, network.Packet{ID: 0x00A0, Data: data}, time.Now())
	}
	messages := mode.ui.console.Messages()
	if len(messages) != len(amounts) {
		t.Fatalf("messages = %+v, want one per pickup (%d)", messages, len(amounts))
	}
	for i, amount := range amounts {
		if want := fmt.Sprintf("You got item 909 %d.", amount); messages[i].Text != want {
			t.Fatalf("message %d = %q, want %q", i, messages[i].Text, want)
		}
	}
	if got := ctx.Session.Inventory.Items[0].Amount; got != 32 {
		t.Fatalf("inventory amount = %d, want 32", got)
	}
}
