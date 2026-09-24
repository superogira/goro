package ui

import (
	"testing"

	"github.com/gogpu/ui/primitives"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func TestLivePresentationDefersRebuildUntilWindowDragEnds(t *testing.T) {
	for _, kind := range []string{"inventory", "stats"} {
		t.Run(kind, func(t *testing.T) {
			app := &windowDragTestApp{}
			ctx := client.Context{
				Input: input.NewState(), UIApp: app, UIManager: NewManager(),
				ScreenW: 800, ScreenH: 600,
				Session: &session.Session{Inventory: session.Inventory{
					Items: []session.InventoryItem{{Index: 7, ItemID: 909, Type: 3, Amount: 5}},
				}},
			}
			var window *Window
			var refresh func()
			var update func() bool
			var snapshotCurrent func() bool
			if kind == "inventory" {
				bag := &InventoryBagWindow{}
				bag.Toggle(ctx)
				window = &bag.Window
				refresh = func() { bag.UpdatePresentation(ctx, nil) }
				update = func() bool { return bag.Update(ctx, nil, nil, nil, nil, nil, nil) }
				snapshotCurrent = func() bool { return bag.snapshot == bag.inventorySnapshot(ctx.Session) }
			} else {
				stats := &StatsWindow{}
				stats.OpenWindow(ctx)
				window = &stats.Window
				refresh = func() { stats.UpdatePresentation(ctx) }
				update = func() bool { return stats.Update(ctx) }
				snapshotCurrent = func() bool { return stats.snapshot == statsWindowSnapshot(ctx.Session) }
			}
			window.OpenAt(10, 20, window.content)
			window.Publish(ctx)
			before := window.content
			blocker := NewWindow(120, 90)
			blocker.OpenAt(50, 380, primitives.Box())
			blocker.Publish(ctx)

			ctx.Input.SetMousePosition(20, 25)
			ctx.Input.SetMouseButton(input.MouseButtonLeft, true)
			if !update() || !app.WindowDragActive() {
				t.Fatal("failed to start captured window drag")
			}
			ctx.Input.EndFrame()
			// Move over another window, outside the dragged window's old bounds.
			ctx.Input.SetMousePosition(80, 395)
			for _, amount := range []int{4, 3} {
				ctx.Session.Inventory.Items[0].Amount = amount
				ctx.Session.Stats.Attack = amount + 10
				refresh() // World mode runs this before dispatching input.
				if window.content != before || !app.WindowDragActive() || app.cancelToken != nil {
					t.Fatal("live update rebuilt the dragged window or released its capture")
				}
				if snapshotCurrent() {
					t.Fatal("deferred update was marked as rendered")
				}
				if blocker.Update(ctx) {
					t.Fatal("another window consumed input belonging to the drag")
				}
				if !update() || window.x != 70 || window.y != 390 {
					t.Fatalf("drag stopped: position=%d,%d", window.x, window.y)
				}
				ctx.Input.EndFrame()
			}

			ctx.Input.SetMouseButton(input.MouseButtonLeft, false)
			refresh()
			// The fork releases the drag layer through band restores (one
			// per frame) instead of one whole-window repaint, so the drag
			// ends either after the last band (end token) or when the
			// deferred content update supersedes the bands with a full
			// repaint (SetContent cancels the pending restore).
			for i := 0; i < 16 && app.endToken == nil && app.cancelToken == nil; i++ {
				_ = update() // yields while its own restore layer is up; bands still drain
				if window.dragging {
					t.Fatalf("drag release stuck dragging (frame %d)", i)
				}
				refresh()
				ctx.Input.EndFrame()
			}
			if app.endToken == nil && app.cancelToken == nil {
				t.Fatal("drag release did not complete normally")
			}
			refresh()
			if window.content == before || !snapshotCurrent() {
				t.Fatal("first presentation update after release did not render the latest state")
			}
			before = window.content
			refresh()
			if window.content != before {
				t.Fatal("idle presentation update rebuilt unchanged contents")
			}
		})
	}
}
