package material3

import (
	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/widget"
)

// ScrollbarPainter renders scrollbars using Material 3 design tokens.
// It maps scrollbar states (normal, hover, dragging) to the M3 color scheme.
//
// If Theme is nil, ScrollbarPainter falls back to the default M3 scrollbar palette.
type ScrollbarPainter struct {
	Theme *Theme // nil uses default M3 fallback
}

// resolveColors returns the ScrollbarColorScheme derived from the painter's Theme.
func (p ScrollbarPainter) resolveColors() scrollview.ScrollbarColorScheme {
	if p.Theme == nil {
		return m3DefaultScrollbarColors
	}
	cs := p.Theme.Colors
	return scrollview.ScrollbarColorScheme{
		Track:      cs.SurfaceVariant.WithAlpha(0.3),
		Thumb:      cs.OnSurfaceVariant.WithAlpha(0.4),
		ThumbHover: cs.OnSurfaceVariant.WithAlpha(0.6),
		ThumbDrag:  cs.OnSurfaceVariant.WithAlpha(0.8),
	}
}

// PaintScrollbar renders scrollbars according to Material 3 specifications.
func (p ScrollbarPainter) PaintScrollbar(canvas widget.Canvas, ps scrollview.PaintState) {
	if ps.Bounds.IsEmpty() {
		return
	}

	colors := ps.ColorScheme
	if colors == (scrollview.ScrollbarColorScheme{}) {
		colors = p.resolveColors()
	}

	// Override paint state with resolved colors.
	ps.ColorScheme = colors

	// Delegate to default painter with resolved colors.
	scrollview.DefaultPainter{}.PaintScrollbar(canvas, ps)
}

// m3DefaultScrollbarColors holds the default M3 scrollbar color scheme.
var m3DefaultScrollbarColors = scrollview.ScrollbarColorScheme{
	Track:      widget.Hex(0xE7E0EC).WithAlpha(0.3), // M3 surface variant
	Thumb:      widget.Hex(0x49454F).WithAlpha(0.4), // M3 on surface variant
	ThumbHover: widget.Hex(0x49454F).WithAlpha(0.6),
	ThumbDrag:  widget.Hex(0x49454F).WithAlpha(0.8),
}

// Compile-time check that ScrollbarPainter implements Painter.
var _ scrollview.Painter = ScrollbarPainter{}
