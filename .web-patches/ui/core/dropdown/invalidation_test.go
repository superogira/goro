package dropdown

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// --- Granular Invalidation Tests (TASK-UI-INVAL-001e) ---

func TestGranularInvalidation_Trigger_HoverEnter(t *testing.T) {
	w := New(Items("A", "B"))
	w.SetBounds(geometry.NewRect(0, 0, 200, 40))
	ctx := widget.NewContext()

	e := event.NewMouseEvent(event.MouseEnter, event.ButtonNone, 0,
		geometry.Pt(100, 20), geometry.Pt(100, 20), event.ModNone)
	w.handleMouseEvent(ctx, e)

	if ctx.IsInvalidated() {
		t.Error("trigger hover enter should use granular invalidation, not ctx.Invalidate()")
	}
	if !w.NeedsRedraw() {
		t.Error("trigger hover enter should set needsRedraw")
	}
}

func TestGranularInvalidation_Trigger_HoverLeave(t *testing.T) {
	w := New(Items("A", "B"))
	w.SetBounds(geometry.NewRect(0, 0, 200, 40))
	ctx := widget.NewContext()

	e := event.NewMouseEvent(event.MouseLeave, event.ButtonNone, 0,
		geometry.Pt(300, 20), geometry.Pt(300, 20), event.ModNone)
	w.handleMouseEvent(ctx, e)

	if ctx.IsInvalidated() {
		t.Error("trigger hover leave should use granular invalidation")
	}
	if !w.NeedsRedraw() {
		t.Error("trigger hover leave should set needsRedraw")
	}
}

func TestGranularInvalidation_Trigger_Press(t *testing.T) {
	w := New(Items("A", "B"))
	w.SetBounds(geometry.NewRect(0, 0, 200, 40))
	ctx := widget.NewContext()

	e := event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft,
		geometry.Pt(100, 20), geometry.Pt(100, 20), event.ModNone)
	w.handleMouseEvent(ctx, e)

	if ctx.IsInvalidated() {
		t.Error("trigger press should use granular invalidation")
	}
	if !w.NeedsRedraw() {
		t.Error("trigger press should set needsRedraw")
	}
}

func TestGranularInvalidation_Menu_MouseHover(t *testing.T) {
	m := &menuWidget{
		items:      []ItemDef{{Label: "A"}, {Label: "B"}, {Label: "C"}},
		itemHeight: 30,
	}
	m.SetBounds(geometry.NewRect(0, 0, 200, 90))
	m.highlightedIndex = -1
	ctx := widget.NewContext()

	e := event.NewMouseEvent(event.MouseMove, event.ButtonNone, 0,
		geometry.Pt(100, 15), geometry.Pt(100, 15), event.ModNone)
	m.handleMouseEvent(ctx, e)

	if ctx.IsInvalidated() {
		t.Error("menu hover should use granular invalidation, not ctx.Invalidate()")
	}
	if !m.NeedsRedraw() {
		t.Error("menu hover should set needsRedraw")
	}
}

func TestGranularInvalidation_Menu_KeyNav(t *testing.T) {
	m := &menuWidget{
		items: []ItemDef{{Label: "A"}, {Label: "B"}, {Label: "C"}},
	}
	m.SetBounds(geometry.NewRect(0, 0, 200, 90))
	m.highlightedIndex = 0

	tests := []struct {
		name string
		key  event.Key
	}{
		{"KeyDown", event.KeyDown},
		{"KeyUp", event.KeyUp},
		{"KeyHome", event.KeyHome},
		{"KeyEnd", event.KeyEnd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := widget.NewContext()
			m.ClearRedraw()

			e := &event.KeyEvent{KeyType: event.KeyPress, Key: tt.key}
			m.handleKeyEvent(ctx, e)

			if ctx.IsInvalidated() {
				t.Errorf("%s should use granular invalidation", tt.name)
			}
			if !m.NeedsRedraw() {
				t.Errorf("%s should set needsRedraw", tt.name)
			}
		})
	}
}

func TestGranularInvalidation_Menu_WheelScroll(t *testing.T) {
	m := &menuWidget{
		items:        []ItemDef{{Label: "A"}, {Label: "B"}, {Label: "C"}, {Label: "D"}, {Label: "E"}},
		scrollOffset: 1,
	}
	m.SetBounds(geometry.NewRect(0, 0, 200, 60))

	ctx := widget.NewContext()
	up := &event.WheelEvent{
		Position: geometry.Pt(100, 30),
		Delta:    geometry.Pt(0, 1),
	}
	m.handleWheelEvent(ctx, up)

	if ctx.IsInvalidated() {
		t.Error("wheel scroll should use granular invalidation")
	}
	if !m.NeedsRedraw() {
		t.Error("wheel scroll should set needsRedraw")
	}
}

func TestGranularInvalidation_OpenClose_UsesGranular(t *testing.T) {
	w := New(Items("A", "B"))
	w.SetBounds(geometry.NewRect(0, 0, 200, 40))

	ctx := widget.NewContext()
	w.open = true
	w.close(ctx)

	// ADR-028: close uses granular invalidation. Overlay removal is handled
	// separately by DrawOverlays; the trigger widget just redraws itself.
	if ctx.IsInvalidated() {
		t.Error("close should use granular invalidation, not ctx.Invalidate()")
	}
	if !w.NeedsRedraw() {
		t.Error("close should set needsRedraw on the trigger widget")
	}
}
