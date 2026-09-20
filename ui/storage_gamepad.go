package ui

import "github.com/kivutar/goro/session"

// Handheld controls for the storage window: d-pad walks the item rows, A
// withdraws the selected item into the inventory (stacks larger than one go
// through the same amount-picker flow as deposits). The close packet goes to
// the server via the handheld B stack (gamepad.go).

// GamepadNavigate moves the storage item selection (dy -1 up, +1 down).
func (w *StorageWindow) GamepadNavigate(ctx Context, dy int) {
	if !w.IsOpen() || dy == 0 {
		return
	}
	items := w.tabItems(ctx.Session)
	if len(items) == 0 {
		return
	}
	next := w.selectedRow + dy
	if next < 0 {
		next = 0
	}
	if next >= len(items) {
		next = len(items) - 1
	}
	if next == w.selectedRow {
		return
	}
	w.selectedRow = next
	w.ensureSelectedRowSignal().Set(next)
	w.gamepadScrollToRow(ctx, next)
	w.refresh(ctx, nil)
}

// GamepadSelectedItem reports the storage item under the handheld cursor.
func (w *StorageWindow) GamepadSelectedItem(ctx Context) (session.InventoryItem, int, bool) {
	if !w.IsOpen() {
		return session.InventoryItem{}, -1, false
	}
	items := w.tabItems(ctx.Session)
	if w.selectedRow < 0 || w.selectedRow >= len(items) {
		return session.InventoryItem{}, -1, false
	}
	return items[w.selectedRow], w.selectedRow, true
}

// GamepadWithdrawResult carries the withdraw request to the game layer
// (which sends the packet and shows the amount picker for stacks).
type GamepadWithdrawRequest struct {
	Item  session.InventoryItem
	Index int
}

// gamepadScrollToRow keeps the selected row inside the visible viewport.
func (w *StorageWindow) gamepadScrollToRow(ctx Context, row int) {
	scroll := w.ensureScrollSignal()
	top := float32(row * storageRowH)
	bottom := top + storageRowH
	value := scroll.Get()
	viewH := float32(storageWindowHeight - ROWindowTitleHeight - ROWindowFooterHeight - 20)
	switch {
	case top < value:
		scroll.Set(top)
	case bottom > value+viewH:
		scroll.Set(bottom - viewH)
	}
}
