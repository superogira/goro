package ui

import (
	"testing"

	"github.com/gogpu/ui/uitest"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
)

func TestCookingWindowButtons(t *testing.T) {
	for _, button := range []int{0, 1, 2} {
		ctx, manager, app := newWindowInstanceTest()
		client, server := newIdentifyTestConnection(t)
		ctx.Network = client
		var window MakingItemWindow
		window.OpenList(ctx, network.MakingItemList{Type: 1, Items: []network.MakingItemOption{{ItemID: 12001}}})
		app.Frame()
		app.Window().DrawTo(&uitest.MockCanvas{})
		buttons := collectEscapeMenuButtons(window.content)
		if len(buttons) != 3 {
			t.Fatalf("buttons = %d, want Close, Cancel and OK", len(buttons))
		}
		point := buttons[button].ScreenBounds().Center()
		app.Window().HandleEvent(uitest.Click(point.X, point.Y))
		app.Window().HandleEvent(uitest.Release(point.X, point.Y))
		want := []byte{0x5b, 0x02, 1, 0, 0xff, 0xff}
		if button == 2 {
			want[4], want[5] = 0xe1, 0x2e
		}
		readIdentifyTestPacket(t, server, want)
		if window.IsOpen() || len(manager.overlays) != 0 {
			t.Fatal("submission left the cooking window visible")
		}
		window.confirm(ctx)
		window.cancel(ctx)
		assertNoIdentifyTestPackets(t, client, server)
	}
}

func TestMakingItemWindowKeyboardSelection(t *testing.T) {
	inputState := input.NewState()
	ctx := Context{Input: inputState, ScreenW: 800, ScreenH: 600}
	window := MakingItemWindow{}
	window.OpenList(ctx, network.MakingItemList{
		Items: []network.MakingItemOption{
			{ItemID: 501},
			{ItemID: 502},
			{ItemID: 503},
		},
	})

	inputState.SetKey(input.KeyArrowDown, true)
	if !window.Update(ctx) {
		t.Fatal("arrow down was not consumed")
	}
	if window.selectedRow != 1 {
		t.Fatalf("selected row after down = %d, want 1", window.selectedRow)
	}

	inputState.EndFrame()
	inputState.SetKey(input.KeyArrowDown, false)
	inputState.EndFrame()
	inputState.SetKey(input.KeyArrowUp, true)
	if !window.Update(ctx) {
		t.Fatal("arrow up was not consumed")
	}
	if window.selectedRow != 0 {
		t.Fatalf("selected row after up = %d, want 0", window.selectedRow)
	}

	inputState.EndFrame()
	inputState.SetKey(input.KeyArrowUp, false)
	inputState.EndFrame()
	inputState.SetKey(input.KeyEnter, true)
	if !window.Update(ctx) {
		t.Fatal("enter was not consumed")
	}
}

func TestRepairItemWindowOpensServerList(t *testing.T) {
	window := RepairItemWindow{}
	window.OpenList(Context{ScreenW: 800, ScreenH: 600}, network.RepairItemList{
		Items: []network.RepairItem{
			{Index: 7, ItemID: 1201, Refine: 5, Cards: [4]uint16{4001, 4002, 0, 0}},
		},
	})

	if !window.IsOpen() {
		t.Fatal("repair item window did not open")
	}
	if len(window.items) != 1 || window.items[0].Index != 7 || window.items[0].ItemID != 1201 || window.items[0].Refine != 5 {
		t.Fatalf("repair item window items = %+v", window.items)
	}

	window.OpenList(Context{ScreenW: 800, ScreenH: 600}, network.RepairItemList{})
	if window.IsOpen() {
		t.Fatal("repair item window stayed open for empty list")
	}
}

func TestWeaponRefineWindowOpensServerList(t *testing.T) {
	window := WeaponRefineWindow{}
	window.OpenList(Context{ScreenW: 800, ScreenH: 600}, network.WeaponRefineList{
		Items: []network.WeaponRefineItem{
			{Index: 11, ItemID: 1101, Refine: 4, Cards: [4]uint16{4001, 0, 0, 0}},
		},
	})

	if !window.IsOpen() {
		t.Fatal("weapon refine window did not open")
	}
	if len(window.items) != 1 || window.items[0].Index != 11 || window.items[0].ItemID != 1101 || window.items[0].Refine != 4 {
		t.Fatalf("weapon refine window items = %+v", window.items)
	}

	window.OpenList(Context{ScreenW: 800, ScreenH: 600}, network.WeaponRefineList{})
	if window.IsOpen() {
		t.Fatal("weapon refine window stayed open for empty list")
	}
}
