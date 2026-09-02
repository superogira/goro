package badge

import (
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// Painter draws the visual representation of a badge.
// Each design system (Material 3, Fluent, Cupertino) provides its own
// Painter implementation to render the badge in its visual style.
//
// If no Painter is set, the badge uses [DefaultPainter].
type Painter interface {
	PaintBadge(canvas widget.Canvas, state PaintState)
}

// PaintState provides the current badge state to the painter.
type PaintState struct {
	Bounds      geometry.Rect    // total widget bounds (excludes outer padding)
	Dot         bool             // true for dot mode (no label)
	Label       string           // pre-formatted count label (empty in dot mode)
	Disabled    bool             // widget is disabled
	ColorScheme BadgeColorScheme // theme-derived colors (zero = use defaults)
}

// DefaultPainter provides a minimal fallback painter with no design system
// styling. It draws a red pill (or dot) with a centered count label -- useful
// for testing and as a base reference.
type DefaultPainter struct{}

// PaintBadge renders the badge. In dot mode it draws a small filled circle.
// In count mode it draws a pill (a rounded rectangle whose radius is half its
// height) with the count label centered inside.
//
// If state.ColorScheme is non-zero, its colors are used instead of the
// built-in defaults.
func (p DefaultPainter) PaintBadge(canvas widget.Canvas, ps PaintState) {
	if ps.Bounds.IsEmpty() {
		return
	}

	hasScheme := ps.ColorScheme != (BadgeColorScheme{})
	bg := resolveBackgroundColor(ps, hasScheme)
	bounds := ps.Bounds

	if ps.Dot {
		// Bounds is non-empty here, so both dimensions are positive.
		radius := min(bounds.Width(), bounds.Height()) / 2
		center := geometry.Pt(
			bounds.Min.X+bounds.Width()/2,
			bounds.Min.Y+bounds.Height()/2,
		)
		canvas.DrawCircle(center, radius, bg)
		return
	}

	// Count mode: pill background.
	radius := bounds.Height() / 2
	canvas.DrawRoundRect(bounds, bg, radius)

	if ps.Label != "" {
		labelColor := resolveLabelColor(ps, hasScheme)
		canvas.DrawText(ps.Label, bounds, defaultFontSize, labelColor, true, widget.TextAlignCenter)
	}
}

// Color resolution helpers.

func resolveBackgroundColor(ps PaintState, hasScheme bool) widget.Color {
	if ps.Disabled {
		if hasScheme {
			return ps.ColorScheme.DisabledBackground
		}
		return defaultDisabledBackground
	}
	if hasScheme {
		return ps.ColorScheme.Background
	}
	return defaultBackground
}

func resolveLabelColor(ps PaintState, hasScheme bool) widget.Color {
	if ps.Disabled {
		if hasScheme {
			return ps.ColorScheme.DisabledLabel
		}
		return defaultDisabledLabel
	}
	if hasScheme {
		return ps.ColorScheme.Label
	}
	return defaultLabel
}

// defaultMax is the count threshold above which the badge renders "N+".
const defaultMax = 99

// Default colors for DefaultPainter.
var (
	defaultBackground         = widget.Hex(0xB3261E) // Material 3 error
	defaultLabel              = widget.ColorWhite
	defaultDisabledBackground = widget.RGBA(0.70, 0.70, 0.70, 1.0)
	defaultDisabledLabel      = widget.RGBA(0.96, 0.96, 0.96, 1.0)
)
