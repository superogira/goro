package ui

import (
	"github.com/gogpu/ui/core/listview"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/ui/rotheme"
)

// HandheldMenu is the MENU-tap overlay for small-screen devices: a simple
// selectable list of game actions (open inventory, skills, ...), driven by
// the d-pad and confirmed with A/Enter. While open, world input (walking,
// attacks) is suspended.
type HandheldMenu struct {
	Window
	labels    []string
	selected  int
	scrollY   state.Signal[float32]
	onSelect  func(ctx client.Context, index int)
	open      bool
	focusNext bool
}

const (
	heldMenuRowH    = 24
	heldMenuRows    = 9
	heldMenuWidth   = 220
	heldMenuPadding = 8
	heldMenuHeight  = ROWindowTitleHeight + heldMenuPadding*2 + heldMenuRows*heldMenuRowH
)

// NewHandheldMenu builds the overlay; it stays closed until Toggle opens it.
func NewHandheldMenu(labels []string, onSelect func(ctx client.Context, index int)) *HandheldMenu {
	m := &HandheldMenu{
		labels:   append([]string(nil), labels...),
		selected: 0,
		scrollY:  state.NewSignal[float32](0),
		onSelect: onSelect,
	}
	return m
}

// IsOpen reports whether the overlay is showing.
func (m *HandheldMenu) IsOpen() bool {
	return m != nil && m.open
}

// Toggle shows or hides the overlay.
func (m *HandheldMenu) Toggle(ctx client.Context) {
	if m == nil {
		return
	}
	if m.open {
		m.Close(ctx)
		return
	}
	if m.Window.width == 0 {
		m.Window = NewWindow(heldMenuWidth, heldMenuHeight)
		m.CloseOnEsc = true
	}
	w, h := ctx.ScreenSize()
	x := (w - heldMenuWidth) / 2
	y := (h - heldMenuHeight) / 2
	m.Window.SetSize(heldMenuWidth, heldMenuHeight)
	m.Window.SetContent(m.tree(ctx))
	m.Window.OpenAt(x, y, m.tree(ctx))
	m.open = true
}

// Close hides the overlay.
func (m *HandheldMenu) Close(ctx client.Context) {
	if m == nil || !m.open {
		return
	}
	m.Window.Close()
	m.open = false
}

// Update handles d-pad selection and activation. Returns true when the
// overlay consumed input this frame (the world should pause its own).
func (m *HandheldMenu) Update(ctx client.Context) bool {
	if !m.IsOpen() || ctx.Input == nil {
		return false
	}
	moved := false
	switch {
	case ctx.Input.JustPressed(input.KeyArrowUp):
		if m.selected > 0 {
			m.selected--
		}
		moved = true
	case ctx.Input.JustPressed(input.KeyArrowDown):
		if m.selected < len(m.labels)-1 {
			m.selected++
		}
		moved = true
	case ctx.Input.JustPressed(input.KeyArrowLeft):
		if m.selected > 0 {
			m.selected--
		}
		moved = true
	case ctx.Input.JustPressed(input.KeyArrowRight):
		if m.selected < len(m.labels)-1 {
			m.selected++
		}
		moved = true
	}
	if moved {
		m.Window.SetContent(m.tree(ctx))
	}
	if ctx.Input.JustPressed(input.KeyEnter) {
		m.Activate(ctx)
		return true
	}
	if ctx.Input.JustPressed(input.KeyEscape) {
		m.Close(ctx)
		return true
	}
	if m.Window.Update(ctx) {
		return true
	}
	return moved
}

// Activate runs the currently selected item and closes the overlay.
func (m *HandheldMenu) Activate(ctx client.Context) {
	if m == nil || !m.open || m.selected < 0 || m.selected >= len(m.labels) {
		return
	}
	index := m.selected
	m.Close(ctx)
	if m.onSelect != nil {
		m.onSelect(ctx, index)
	}
}

func (m *HandheldMenu) tree(ctx client.Context) widget.Widget {
	return Win(
		Title("Menu"),
		CloseButton(false),
		Size(float32(heldMenuWidth), float32(heldMenuHeight)),
		Content(
			primitives.Box(m.list()).Padding(heldMenuPadding),
		),
	)
}

func (m *HandheldMenu) list() widget.Widget {
	lv := listview.New(
		listview.ItemCount(len(m.labels)),
		listview.FixedItemHeight(heldMenuRowH),
		listview.ScrollYSignal(m.scrollY),
		listview.SelectionModeOpt(listview.SelectionSingle),
		listview.SelectedIndex(m.selected),
		listview.OnSelectionChange(func(index int) {
			m.selected = index
		}),
		listview.PainterOpt(rotheme.SelectListPainter{EmptyText: ""}),
		listview.BuildItem(func(item listview.ItemContext) widget.Widget {
			label := ""
			if item.Index >= 0 && item.Index < len(m.labels) {
				label = m.labels[item.Index]
			}
			return rotheme.SelectListRow(label, true, heldMenuRowH)
		}),
	)
	return lv
}
