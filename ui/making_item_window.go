package ui

import (
	"image"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	makingItemWindowWidth  = 312
	makingItemTableHeaderH = 24
	makingItemRowH         = 32
	makingItemRows         = 6
	makingItemWindowHeight = ROWindowTitleHeight + makingItemTableHeaderH + makingItemRows*makingItemRowH + ROWindowFooterHeight
	makingItemCancelID     = uint16(0xFFFF)
)

type MakingItemWindow struct {
	Window
	scrollY      state.Signal[float32]
	selectedRow  int
	items        []network.MakingItemOption
	makeType     uint16 // Zero for forging; nonzero types use the cooking reply.
	lastClickAt  time.Time
	lastClickRow int
	icons        map[identifyItemIconKey]image.Image
	iconMiss     map[identifyItemIconKey]struct{}
}

func (w *MakingItemWindow) OpenList(ctx Context, list network.MakingItemList) {
	w.EnsureWindow(makingItemWindowWidth, makingItemWindowHeight)
	w.items = append(w.items[:0], list.Items...)
	w.makeType = list.Type
	w.selectedRow = 0
	w.lastClickRow = -1
	w.lastClickAt = time.Time{}
	w.ensureScrollSignal().Set(0)
	w.clampScroll()
	if len(w.items) == 0 {
		if w.makeType != 0 && ctx.Network != nil {
			if err := w.sendRequest(ctx, makingItemCancelID, [3]uint16{}); err != nil {
				glog.Warnf("empty cooking list cancel failed: %v", err)
			}
		}
		w.Close()
		w.Publish(ctx)
		return
	}
	w.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *MakingItemWindow) ApplyAck(ctx Context, ack network.MakingItemAck) {
	w.EnsureWindow(makingItemWindowWidth, makingItemWindowHeight)
	if !ack.Success() {
		glog.Warnf("item creation failed item=%d result=%d alchemist=%v", ack.ItemID, ack.Result, ack.Alchemist())
	}
	w.Close()
	w.Publish(ctx)
}

func (w *MakingItemWindow) Update(ctx Context) bool {
	w.EnsureWindow(makingItemWindowWidth, makingItemWindowHeight)
	if !w.IsOpen() {
		return false
	}
	w.clampScroll()
	if ctx.Input != nil && ctx.Input.JustPressed(input.KeyEscape) {
		w.cancel(ctx)
		w.Publish(ctx)
		return true
	}
	consumed := w.Window.Update(ctx)
	keyboardConsumed := false
	if w.IsOpen() {
		keyboardConsumed = updateSelectionTableKeyboard(ctx, &w.selectedRow, len(w.items), makingItemRows, makingItemRowH, w.ensureScrollSignal(), func() {
			w.confirm(ctx)
		})
		if keyboardConsumed && w.IsOpen() {
			w.RebindContent(ctx, w.widgetTree(ctx))
		}
	}
	if w.IsOpen() {
		w.updateDoubleClick(ctx)
	}
	w.Publish(ctx)
	return consumed || keyboardConsumed
}

func (w *MakingItemWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Item Creation"),
		CloseButton(true),
		OnClose(func() {
			w.cancel(ctx)
			w.Publish(ctx)
		}),
		Size(makingItemWindowWidth, makingItemWindowHeight),
		Content(
			primitives.Box(w.tableWidget(ctx)).
				Height(makingItemTableHeight()).
				Background(rotheme.Default.Colors.PanelBody),
		),
		Footer(
			primitives.Expanded(primitives.Box()),
			rotheme.Button("Cancel", func() {
				w.cancel(ctx)
				w.Publish(ctx)
			}),
			rotheme.Button("OK", func() {
				w.confirm(ctx)
				w.Publish(ctx)
			}),
		),
	)
}

func (w *MakingItemWindow) tableWidget(ctx Context) *rotheme.TableViewWidget {
	items := append([]network.MakingItemOption(nil), w.items...)
	return itemTableView(
		w.itemTableRows(ctx, items),
		"Item",
		makingItemRowH,
		makingItemTableHeaderH,
		"No items",
		w.ensureScrollSignal(),
		w.selectedRow,
		func(row int) {
			w.selectedRow = row
		},
	)
}

func (w *MakingItemWindow) confirm(ctx Context) {
	if !w.IsOpen() || w.selectedRow < 0 || w.selectedRow >= len(w.items) {
		return
	}
	item := w.items[w.selectedRow]
	w.makeItem(ctx, item.ItemID, item.Material)
}

func (w *MakingItemWindow) updateDoubleClick(ctx Context) {
	if ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonLeft) {
		return
	}
	row, ok := w.rowAtMouse(ctx.Input.MouseX, ctx.Input.MouseY)
	if !ok {
		return
	}
	now := time.Now()
	if w.lastClickRow == row && now.Sub(w.lastClickAt) <= 360*time.Millisecond {
		w.selectedRow = row
		w.lastClickRow = -1
		w.lastClickAt = time.Time{}
		w.confirm(ctx)
		return
	}
	w.lastClickRow = row
	w.lastClickAt = now
}

func (w *MakingItemWindow) rowAtMouse(mouseX, mouseY int) (int, bool) {
	tableX := w.x
	tableY := w.y + ROWindowTitleHeight
	rowY := tableY + makingItemTableHeaderH
	if !pointInRect(mouseX, mouseY, tableX, rowY, scrollbarSafeIntWidth(makingItemWindowWidth), makingItemRows*makingItemRowH) {
		return 0, false
	}
	row := int((float32(mouseY-rowY) + w.ensureScrollSignal().Get()) / makingItemRowH)
	if row < 0 || row >= len(w.items) {
		return 0, false
	}
	return row, true
}

func (w *MakingItemWindow) makeItem(ctx Context, itemID uint16, material [3]uint16) {
	if ctx.Network == nil {
		glog.Warnf("item creation failed: not connected")
		return
	}
	if err := w.sendRequest(ctx, itemID, material); err != nil {
		glog.Warnf("item creation failed: %v", err)
		return
	}
	glog.Debugf("item creation requested item=%d material=%v", itemID, material)
	w.Close()
}

func (w *MakingItemWindow) cancel(ctx Context) {
	if !w.IsOpen() {
		return
	}
	if ctx.Network != nil {
		if err := w.sendRequest(ctx, makingItemCancelID, [3]uint16{}); err != nil {
			glog.Warnf("item creation cancel failed: %v", err)
		}
	}
	w.Close()
}

func (w *MakingItemWindow) sendRequest(ctx Context, itemID uint16, material [3]uint16) error {
	if w.makeType != 0 {
		return ctx.Network.SendCooking(w.makeType, itemID)
	}
	return ctx.Network.SendMakingItem(itemID, material)
}

func (w *MakingItemWindow) itemTableRows(ctx Context, items []network.MakingItemOption) []itemTableRow {
	rows := make([]itemTableRow, len(items))
	for i, item := range items {
		rows[i] = itemTableRow{
			name: inventoryItemDisplayName(ctx.Resources, session.InventoryItem{ItemID: item.ItemID, Identified: true}),
			icon: w.itemIconImage(ctx.Resources, item.ItemID),
		}
	}
	return rows
}

func (w *MakingItemWindow) itemIconImage(manager *res.Manager, itemID uint16) image.Image {
	return cachedIdentifiedItemIcon(manager, itemID, &w.icons, &w.iconMiss)
}

func (w *MakingItemWindow) clampScroll() {
	if w.selectedRow >= len(w.items) {
		w.selectedRow = -1
	}
	scroll := w.ensureScrollSignal()
	maxScroll := float32(maxInt(0, len(w.items)-makingItemRows) * makingItemRowH)
	switch value := scroll.Get(); {
	case value < 0:
		scroll.Set(0)
	case value > maxScroll:
		scroll.Set(maxScroll)
	}
}

func (w *MakingItemWindow) ensureScrollSignal() state.Signal[float32] {
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
	}
	return w.scrollY
}

func makingItemTableHeight() float32 {
	return makingItemTableHeaderH + makingItemRows*makingItemRowH
}
