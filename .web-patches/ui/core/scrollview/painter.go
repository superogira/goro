package scrollview

import (
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// Painter draws the visual representation of scrollbars.
// Each design system (Material 3, Fluent, Cupertino) provides its own
// Painter implementation to render scrollbars in its visual style.
//
// If no Painter is set, the scroll view uses [DefaultPainter].
type Painter interface {
	PaintScrollbar(canvas widget.Canvas, state PaintState)
}

// PaintState provides the current scrollbar state to the painter.
type PaintState struct {
	Bounds      geometry.Rect // total widget bounds (viewport)
	Direction   ScrollDirection
	Focused     bool
	Hovered     bool
	Dragging    bool
	ColorScheme ScrollbarColorScheme

	// Vertical scrollbar state.
	VScrollVisible bool          // whether vertical scrollbar should be drawn
	VThumbRect     geometry.Rect // vertical thumb rectangle
	VTrackRect     geometry.Rect // vertical track rectangle

	// Horizontal scrollbar state.
	HScrollVisible bool          // whether horizontal scrollbar should be drawn
	HThumbRect     geometry.Rect // horizontal thumb rectangle
	HTrackRect     geometry.Rect // horizontal track rectangle
}

// ScrollbarColorScheme provides theme-derived colors for scrollbar painting.
// Zero value means the painter should use its built-in defaults.
type ScrollbarColorScheme struct {
	Track      widget.Color // scrollbar track background
	Thumb      widget.Color // scrollbar thumb fill
	ThumbHover widget.Color // scrollbar thumb fill when hovered
	ThumbDrag  widget.Color // scrollbar thumb fill when dragging
}

// DefaultPainter provides a minimal fallback painter with no design system styling.
// It draws simple scrollbars -- useful for testing and as a base reference.
type DefaultPainter struct{}

// PaintScrollbar renders minimal scrollbars with gray track and darker thumb.
func (p DefaultPainter) PaintScrollbar(canvas widget.Canvas, ps PaintState) {
	if ps.Bounds.IsEmpty() {
		return
	}

	hasScheme := ps.ColorScheme != (ScrollbarColorScheme{})

	if ps.VScrollVisible {
		paintScrollbarTrack(canvas, ps.VTrackRect, hasScheme, ps.ColorScheme)
		paintScrollbarThumb(canvas, ps.VThumbRect, ps, hasScheme)
	}

	if ps.HScrollVisible {
		paintScrollbarTrack(canvas, ps.HTrackRect, hasScheme, ps.ColorScheme)
		paintScrollbarThumb(canvas, ps.HThumbRect, ps, hasScheme)
	}
}

// paintScrollbarTrack draws the scrollbar track background.
func paintScrollbarTrack(canvas widget.Canvas, trackRect geometry.Rect, hasScheme bool, cs ScrollbarColorScheme) {
	color := defaultTrackColor
	if hasScheme {
		color = cs.Track
	}
	canvas.DrawRoundRect(trackRect, color, scrollbarRadius)
}

// paintScrollbarThumb draws the scrollbar thumb.
func paintScrollbarThumb(canvas widget.Canvas, thumbRect geometry.Rect, ps PaintState, hasScheme bool) {
	color := resolveThumbColor(ps, hasScheme)
	canvas.DrawRoundRect(thumbRect, color, scrollbarRadius)
}

// resolveThumbColor returns the thumb color based on interaction state.
func resolveThumbColor(ps PaintState, hasScheme bool) widget.Color {
	if ps.Dragging {
		if hasScheme {
			return ps.ColorScheme.ThumbDrag
		}
		return defaultThumbDragColor
	}
	if ps.Hovered {
		if hasScheme {
			return ps.ColorScheme.ThumbHover
		}
		return defaultThumbHoverColor
	}
	if hasScheme {
		return ps.ColorScheme.Thumb
	}
	return defaultThumbColor
}

// Scrollbar dimensions.
const (
	scrollbarWidth   float32 = 8  // width of the scrollbar track
	scrollbarPadding float32 = 2  // padding around the scrollbar
	scrollbarRadius  float32 = 4  // corner radius for track and thumb
	minThumbSize     float32 = 20 // minimum thumb size in pixels
)

// Default colors for DefaultPainter.
var (
	defaultTrackColor      = widget.RGBA(0.9, 0.9, 0.9, 0.5)
	defaultThumbColor      = widget.RGBA(0.6, 0.6, 0.6, 0.7)
	defaultThumbHoverColor = widget.RGBA(0.5, 0.5, 0.5, 0.8)
	defaultThumbDragColor  = widget.RGBA(0.4, 0.4, 0.4, 0.9)
)
