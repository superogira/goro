package ui

import (
	"net"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestEquipmentGamepadNavigateWalksSlotGrid(t *testing.T) {
	window := EquipmentWindow{}
	window.Toggle(Context{ScreenW: 1280, ScreenH: 720})
	if window.gamepadSelected != 0 {
		t.Fatalf("initial selection = %d, want 0 (head top)", window.gamepadSelected)
	}
	// Job 3 (hunter) can use ammo, so all three columns exist.
	ctx := Context{Session: &session.Session{Selected: session.Character{ID: 1, Job: 3}}}

	// Right from the head-top slot steps to head-mid (left column -> right
	// column, no center slot on row 0).
	window.GamepadNavigate(ctx, 1, 0)
	if window.gamepadSelected != 1 {
		t.Fatalf("after right, selection = %d, want 1 (head mid)", window.gamepadSelected)
	}
	// Down walks the right column.
	window.GamepadNavigate(ctx, 0, 1)
	if window.gamepadSelected != 3 {
		t.Fatalf("after down, selection = %d, want 3 (armor)", window.gamepadSelected)
	}
	// Left from armor lands on the ammo slot (center column, row 1).
	window.GamepadNavigate(ctx, -1, 0)
	if window.gamepadSelected != 10 {
		t.Fatalf("after left, selection = %d, want 10 (ammo)", window.gamepadSelected)
	}
	// Left again continues to the left column.
	window.GamepadNavigate(ctx, -1, 0)
	if window.gamepadSelected != 2 {
		t.Fatalf("after left, selection = %d, want 2 (head low)", window.gamepadSelected)
	}
	// Edges clamp: up from row 0 stays put.
	window.gamepadSelected = 0
	window.GamepadNavigate(ctx, 0, -1)
	if window.gamepadSelected != 0 {
		t.Fatalf("after up at top, selection = %d, want 0", window.gamepadSelected)
	}
}

func TestEquipmentGamepadNavigateSkipsHiddenAmmo(t *testing.T) {
	// Job 1 (swordman) cannot use ammo, so the center slot does not exist.
	window := EquipmentWindow{}
	window.Toggle(Context{ScreenW: 1280, ScreenH: 720})
	s := &session.Session{Selected: session.Character{ID: 1, Job: 1}}
	ctx := Context{Session: s}
	window.gamepadSelected = 2 // head low, left column row 1
	window.GamepadNavigate(ctx, 1, 0)
	if window.gamepadSelected != 3 {
		t.Fatalf("after right, selection = %d, want 3 (armor, skipping hidden ammo)", window.gamepadSelected)
	}
}

func TestEquipmentGamepadActivateSendsTakeoff(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	written := make(chan []byte, 4)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 256)
		for {
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, err := conn.Read(buf)
			if n > 0 {
				out := make([]byte, n)
				copy(out, buf[:n])
				written <- out
			}
			if err != nil {
				return
			}
		}
	}()

	client := network.NewClient(20080910, false)
	addr := listener.Addr().(*net.TCPAddr)
	if err := client.Connect(t.Context(), addr.IP.String(), addr.Port); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer client.Close()

	s := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 7, ItemID: 2101, Location: db.EquipShield, WearLocation: db.EquipShield, Equip: true, Equipped: true},
			},
		},
	}
	window := EquipmentWindow{}
	window.Toggle(Context{ScreenW: 1280, ScreenH: 720})
	window.gamepadSelected = 5 // shield
	window.lastClickItem = 9   // armed by an earlier mouse click
	window.GamepadActivate(Context{Session: s, Network: client})

	select {
	case data := <-written:
		if len(data) == 0 {
			t.Fatal("takeoff packet was empty")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no takeoff packet was sent")
	}
	// A single gamepad press must not leave the mouse double-click armed.
	if window.lastClickItem != 0 {
		t.Fatalf("lastClickItem = %d, want 0", window.lastClickItem)
	}
}

func TestEquipmentGamepadActivateWithoutItemIsNoop(t *testing.T) {
	window := EquipmentWindow{}
	window.Toggle(Context{ScreenW: 1280, ScreenH: 720})
	window.gamepadSelected = 0 // empty head-top slot
	window.GamepadActivate(Context{Session: &session.Session{}})
	if window.lastClickItem != 0 {
		t.Fatalf("lastClickItem = %d, want untouched-empty", window.lastClickItem)
	}
}

func TestEquipmentGamepadInfoOpensDescription(t *testing.T) {
	s := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 4, ItemID: 1201, Location: db.EquipWeapon, WearLocation: db.EquipWeapon, Equip: true, Equipped: true},
			},
		},
	}
	itemInfo := &ItemWindows{}
	window := EquipmentWindow{}
	window.Toggle(Context{ScreenW: 1280, ScreenH: 720})
	window.Rebind(Context{Session: s}, itemInfo, nil, nil)
	window.gamepadSelected = 4 // weapon
	window.GamepadInfo(Context{Session: s})
	if !itemInfo.HasOpenDescriptions() {
		t.Fatal("expected an open item description window")
	}
}
