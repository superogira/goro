package ui

import (
	"fmt"
	"slices"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
)

// ItemWindows owns independently closable descriptions and card artwork.
// Each description is a snapshot, so inspecting another item cannot change it.
// The book reader remains a single window.
type ItemWindows struct {
	descriptions  []*ItemInfoWindow
	illustrations []*CardIllustrationWindow
	book          BookWindow
}

func (w *ItemWindows) openItem(ctx Context, item session.InventoryItem, x, y int) {
	if item.ItemID == 0 {
		return
	}
	info := &ItemInfoWindow{}
	info.openItem(ctx, item, x, y)
	info.placeNew(ctx, info.x, info.y)
	w.descriptions = append(w.descriptions, info)
}

func (w *ItemWindows) Update(ctx Context, assets AssetProvider) (bool, error) {
	w.pruneClosed()
	if consumed, err := w.handleRequests(ctx); consumed {
		return true, err
	}
	if w.book.Update(ctx) {
		return true, nil
	}
	for _, art := range w.illustrations {
		if art.Update(ctx) {
			return true, nil
		}
	}
	for _, info := range w.descriptions {
		if info.Update(ctx, assets) {
			return true, nil
		}
	}
	return false, nil
}

func (w *ItemWindows) handleRequests(ctx Context) (bool, error) {
	for _, info := range w.descriptions {
		if request := info.cardInfoRequest; request.ItemID != 0 {
			info.cardInfoRequest = itemInfoCardRequest{}
			w.openItem(ctx, session.InventoryItem{ItemID: request.ItemID, Type: db.ItemTypeCard, Identified: true}, request.X, request.Y)
			return true, nil
		}
		if request := info.PopCardIllustrationRequest(); request.ItemID != 0 {
			art := &CardIllustrationWindow{}
			if err := art.Open(ctx, request.ItemID, request.Title); err != nil {
				return true, fmt.Errorf("unable to display card: %w", err)
			}
			art.placeNew(ctx, info.x+ROWindowTitleHeight, info.y+ROWindowTitleHeight)
			w.illustrations = append(w.illustrations, art)
			return true, nil
		}
		if request := info.PopReadBookRequest(); request.ItemID != 0 {
			if err := w.book.Open(ctx, request.ItemID, request.Title); err != nil {
				return true, fmt.Errorf("unable to read book: %w", err)
			}
			info.Close()
			w.book.Raise(ctx)
			return true, nil
		}
	}
	return false, nil
}

func (w *ItemWindows) Rebind(ctx Context, assets AssetProvider) {
	w.pruneClosed()
	for _, info := range w.descriptions {
		info.Rebind(ctx, assets)
	}
	for _, art := range w.illustrations {
		art.Rebind(ctx)
	}
	w.book.Rebind(ctx)
}

func (w *ItemWindows) KeyboardShortcutsBlocked() bool { return w.book.IsOpen() }

func (w *ItemWindows) DrawTooltip(ctx Context, screen *render.Frame) {
	for _, info := range w.descriptions {
		if info.IsOpen() {
			info.DrawTooltip(ctx, screen)
		}
	}
}

func (w *ItemWindows) pruneClosed() {
	w.descriptions = slices.DeleteFunc(w.descriptions, func(info *ItemInfoWindow) bool { return !info.IsOpen() })
	w.illustrations = slices.DeleteFunc(w.illustrations, func(art *CardIllustrationWindow) bool { return !art.IsOpen() })
}

// HasOpenDescriptions reports whether any item description window is open —
// the handheld layer treats them as sitting on top of the inventory.
func (w *ItemWindows) HasOpenDescriptions() bool {
	for _, info := range w.descriptions {
		if info.IsOpen() {
			return true
		}
	}
	return false
}

// CloseTopDescription closes the most recently opened description window.
// The handheld B button walks the stack top-down so one press never closes
// the inventory underneath.
func (w *ItemWindows) CloseTopDescription(ctx Context) bool {
	for i := len(w.descriptions) - 1; i >= 0; i-- {
		info := w.descriptions[i]
		if !info.IsOpen() {
			continue
		}
		info.tooltip.Hide()
		info.Close()
		info.Publish(ctx)
		return true
	}
	return false
}

// CloseTopIllustration closes the most recently opened card artwork window.
func (w *ItemWindows) CloseTopIllustration(ctx Context) bool {
	for i := len(w.illustrations) - 1; i >= 0; i-- {
		art := w.illustrations[i]
		if !art.IsOpen() {
			continue
		}
		art.Close()
		art.Publish(ctx)
		return true
	}
	return false
}

// GamepadCardNavigate steps the card-slot selection of the top-most open
// description window (dir < 0 left, dir > 0 right). It reports false when
// that window shows no card slots, so the caller can route the d-pad
// elsewhere.
func (w *ItemWindows) GamepadCardNavigate(ctx Context, dir int) bool {
	for i := len(w.descriptions) - 1; i >= 0; i-- {
		info := w.descriptions[i]
		if !info.IsOpen() {
			continue
		}
		return info.GamepadCardNavigate(ctx, dir)
	}
	return false
}

// GamepadOpenSelectedCard opens the description of the card in the selected
// slot of the top-most open description window, stacking another window.
func (w *ItemWindows) GamepadOpenSelectedCard(ctx Context) bool {
	for i := len(w.descriptions) - 1; i >= 0; i-- {
		info := w.descriptions[i]
		if !info.IsOpen() {
			continue
		}
		info.GamepadOpenSelectedCard(ctx)
		return true
	}
	return false
}
