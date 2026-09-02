package progress

import (
	"math"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// Painter draws the visual representation of a circular progress indicator.
// Each design system (Material 3, Fluent, Cupertino) provides its own
// Painter implementation to render the indicator in its visual style.
//
// If no Painter is set, the progress indicator uses [DefaultPainter].
type Painter interface {
	PaintProgress(canvas widget.Canvas, state PaintState)
}

// PaintState provides the current progress indicator state to the painter.
type PaintState struct {
	Value          float64             // current value clamped to [0, 1] (determinate mode)
	Bounds         geometry.Rect       // total widget bounds
	Diameter       float32             // indicator diameter in logical pixels
	StrokeWidth    float32             // arc stroke width in logical pixels
	ShowLabel      bool                // whether to show percentage label (determinate only)
	Label          string              // pre-formatted label text (empty if ShowLabel is false)
	Indeterminate  bool                // true for spinner mode
	Rotation       float64             // current rotation in radians (indeterminate mode)
	AnimationPhase float64             // 0-1 sawtooth phase within one grow/shrink cycle
	Disabled       bool                // widget is disabled
	ColorScheme    ProgressColorScheme // theme-derived colors (zero = use defaults)
}

// DefaultPainter provides a minimal fallback painter with no design system styling.
// It draws a circular progress indicator using cubic Bézier arc strokes.
type DefaultPainter struct{}

// PaintProgress renders the circular progress indicator.
// In determinate mode, it draws a track circle and a progress arc.
// In indeterminate mode, it draws a rotating partial arc.
func (p DefaultPainter) PaintProgress(canvas widget.Canvas, ps PaintState) {
	if ps.Bounds.IsEmpty() {
		return
	}

	bounds := ps.Bounds
	centerX := bounds.Min.X + bounds.Width()/2
	centerY := bounds.Min.Y + bounds.Height()/2
	center := geometry.Pt(centerX, centerY)

	// Use the smaller of width/height for the radius, minus stroke width.
	availDiameter := ps.Diameter
	if bounds.Width() < availDiameter {
		availDiameter = bounds.Width()
	}
	if bounds.Height() < availDiameter {
		availDiameter = bounds.Height()
	}
	radius := (availDiameter - ps.StrokeWidth) / 2
	if radius <= 0 {
		return
	}

	if ps.Indeterminate {
		p.paintIndeterminate(canvas, ps, center, radius)
	} else {
		p.paintDeterminate(canvas, ps, center, radius)
	}
}

// paintDeterminate draws a track circle and a progress arc.
func (p DefaultPainter) paintDeterminate(canvas widget.Canvas, ps PaintState, center geometry.Point, radius float32) {
	hasScheme := ps.ColorScheme != (ProgressColorScheme{})

	// Draw track circle (full 360 degrees).
	trackColor := resolveTrackColor(ps, hasScheme)
	canvas.StrokeCircle(center, radius, trackColor, ps.StrokeWidth)

	// Draw progress arc (0 to value*360 degrees, starting from top).
	if ps.Value > 0 {
		indicatorColor := resolveIndicatorColor(ps, hasScheme)
		// Start from top (-pi/2), sweep clockwise by value * 2*pi.
		startAngle := -math.Pi / 2
		sweepAngle := ps.Value * 2 * math.Pi
		drawArcStyled(canvas, center, radius, startAngle, sweepAngle, indicatorColor, ps.StrokeWidth, widget.LineCapButt)
	}

	// Draw label centered if enabled.
	if ps.ShowLabel && ps.Label != "" {
		labelColor := resolveLabelColor(ps, hasScheme)
		// Create a bounding rect centered on the indicator.
		labelSize := ps.Diameter
		labelBounds := geometry.NewRect(
			center.X-labelSize/2,
			center.Y-labelSize/2,
			labelSize,
			labelSize,
		)
		canvas.DrawText(ps.Label, labelBounds, defaultFontSize, labelColor, false, widget.TextAlignCenter)
	}
}

// paintIndeterminate draws a variable-length rotating arc.
func (p DefaultPainter) paintIndeterminate(canvas widget.Canvas, ps PaintState, center geometry.Point, radius float32) {
	hasScheme := ps.ColorScheme != (ProgressColorScheme{})

	// Draw track circle.
	trackColor := resolveTrackColor(ps, hasScheme)
	canvas.StrokeCircle(center, radius, trackColor, ps.StrokeWidth)

	// Compute variable sweep from AnimationPhase.
	phase := ps.AnimationPhase
	headValue := easeInOut(math.Min(phase*2, 1.0))
	tailValue := easeInOut(math.Max((phase-0.5)*2, 0.0))

	const maxSweep = math.Pi * 1.5 // 270°
	const minSweep = 0.05
	arcSweep := (headValue - tailValue) * maxSweep
	if arcSweep < minSweep {
		arcSweep = minSweep
	}
	arcStart := -math.Pi/2 + ps.Rotation + tailValue*maxSweep

	indicatorColor := resolveIndicatorColor(ps, hasScheme)
	drawArcStyled(canvas, center, radius, arcStart, arcSweep, indicatorColor, ps.StrokeWidth, widget.LineCapRound)
}

// drawArcStyled draws an arc with the specified line cap, falling back to StrokeArc.
func drawArcStyled(canvas widget.Canvas, center geometry.Point, radius float32,
	startAngle, sweepAngle float64, color widget.Color, strokeWidth float32, lineCap widget.LineCap) {
	if s, ok := canvas.(widget.ArcStroker); ok {
		s.StrokeArcStyled(center, radius, startAngle, sweepAngle, color, strokeWidth, lineCap)
		return
	}
	canvas.StrokeArc(center, radius, startAngle, sweepAngle, color, strokeWidth)
}

// easeInOut applies a cubic ease-in-out curve.
func easeInOut(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	v := -2*t + 2
	return 1 - v*v*v/2
}

// Color resolution helpers.

func resolveTrackColor(ps PaintState, hasScheme bool) widget.Color {
	if ps.Disabled {
		if hasScheme {
			return ps.ColorScheme.DisabledTrack
		}
		return defaultDisabledTrack
	}
	if hasScheme && ps.ColorScheme.trackSet {
		return ps.ColorScheme.Track
	}
	return defaultTrackColor
}

func resolveIndicatorColor(ps PaintState, hasScheme bool) widget.Color {
	if ps.Disabled {
		if hasScheme {
			return ps.ColorScheme.DisabledIndicator
		}
		return defaultDisabledIndicator
	}
	if hasScheme && ps.ColorScheme.indicatorSet {
		return ps.ColorScheme.Indicator
	}
	return defaultIndicatorColor
}

func resolveLabelColor(ps PaintState, hasScheme bool) widget.Color {
	if hasScheme {
		return ps.ColorScheme.Label
	}
	return defaultLabelColor
}

// Default colors for DefaultPainter.
var (
	defaultIndicatorColor    = widget.Hex(0x6750A4) // Material primary
	defaultTrackColor        = widget.RGBA(0.90, 0.90, 0.90, 1.0)
	defaultLabelColor        = widget.ColorBlack
	defaultDisabledIndicator = widget.RGBA(0.70, 0.70, 0.70, 1.0)
	defaultDisabledTrack     = widget.RGBA(0.93, 0.93, 0.93, 1.0)
)
