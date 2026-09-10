package rotheme

import (
	"fmt"
	"math"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

const IconButtonSize float32 = 17

type IconButtonKind int

const (
	IconButtonClose IconButtonKind = iota
	IconButtonPlus
	IconButtonMinus
	IconButtonLeft
	IconButtonRight
	IconButtonUp
	IconButtonDown
)

var iconButtonSegments = map[IconButtonKind][][4]float32{
	IconButtonClose: {{-1, -1, 1, 1}, {1, -1, -1, 1}},
	IconButtonPlus:  {{-1, 0, 1, 0}, {0, -1, 0, 1}},
	IconButtonMinus: {{-1, 0, 1, 0}},
}

const iconButtonGlyphYOffset float32 = 0.25

func IconButton(kind IconButtonKind, onClick func()) *primitives.BoxWidget {
	return IconButtonDisabled(kind, false, onClick)
}

func IconButtonDisabled(kind IconButtonKind, disabled bool, onClick func()) *primitives.BoxWidget {
	opts := []button.Option{
		button.TextOpt(""),
		button.SizeOpt(button.Small),
		button.PainterOpt(IconButtonPainter{Kind: kind}),
		button.RoundedOpt(ButtonRadius),
		button.Disabled(disabled),
	}
	if !disabled && onClick != nil {
		opts = append(opts, button.OnClick(onClick))
	}
	return primitives.Box(newMouseOnlyButton(button.New(opts...).PaddingXY(0, 0).MinWidth(IconButtonSize))).
		Width(IconButtonSize).
		Height(IconButtonSize)
}

type IconButtonPainter struct {
	Kind IconButtonKind
}

func (p IconButtonPainter) PaintButton(canvas widget.Canvas, state button.PaintState) {
	if isDirectionalIconButton(p.Kind) {
		fill, border := directionalIconButtonColors(state.Hovered, state.Pressed, state.Disabled, state.Background)
		drawDirectionalIconButton(canvas, state.Bounds, p.Kind, fill, border)
		return
	}
	ButtonPainter{}.PaintButton(canvas, state)
	color := Default.Colors.Text
	if state.Disabled {
		color = Default.Colors.MutedText
	}
	drawIconGlyph(canvas, state.Bounds, p.Kind, color)
}

func DrawIconButton(canvas widget.Canvas, bounds geometry.Rect, kind IconButtonKind, hovered, disabled bool) {
	if isDirectionalIconButton(kind) {
		fill, border := directionalIconButtonColors(hovered, false, disabled, nil)
		drawDirectionalIconButton(canvas, bounds, kind, fill, border)
		return
	}
	top, bottom := buttonTitleBarGradient(2)
	color := Default.Colors.Text
	border := Default.Colors.ButtonBorder
	if hovered {
		top, bottom = buttonTitleBarGradient(4)
	}
	if disabled {
		top = Default.Colors.Disabled
		bottom = buttonGradientLight(top)
		color = Default.Colors.MutedText
		border = Default.Colors.FooterLine
	}
	drawButtonShadow(canvas, bounds, ButtonRadius)
	drawButtonGradientColors(canvas, bounds, top, bottom, ButtonRadius)
	drawButtonReflect(canvas, bounds, ButtonRadius)
	canvas.StrokeRoundRect(bounds, border, ButtonRadius, 1)
	drawIconGlyph(canvas, bounds, kind, color)
}

func drawIconGlyph(canvas widget.Canvas, bounds geometry.Rect, kind IconButtonKind, color widget.Color) {
	size := bounds.Width()
	if bounds.Height() < size {
		size = bounds.Height()
	}
	icon := int(size) / 2
	if icon < 6 {
		icon = int(size) - 6
	}
	if icon%2 == 0 {
		icon--
	}
	if icon < 2 {
		return
	}
	midX := float32(int(bounds.Min.X + bounds.Width()/2))
	midY := float32(int(bounds.Min.Y+bounds.Height()/2)) + iconButtonGlyphYOffset
	half := float32(icon / 2)
	for _, s := range iconButtonSegments[kind] {
		canvas.DrawLine(
			geometry.Pt(midX+s[0]*half, midY+s[1]*half),
			geometry.Pt(midX+s[2]*half, midY+s[3]*half),
			color,
			1,
		)
	}
}

type directionalArrowSpec struct {
	borderPath string
	fillPath   string
	points     [3]geometry.Point
}

const (
	directionalArrowCornerFraction float32 = 0.24
	directionalArrowBorderInset    float32 = 1.25
)

var directionalArrows = [...]directionalArrowSpec{
	IconButtonLeft: newDirectionalArrow([3]geometry.Point{
		geometry.Pt(12.95, 1.65), geometry.Pt(3.05, 8.25), geometry.Pt(12.95, 14.85),
	}),
	IconButtonRight: newDirectionalArrow([3]geometry.Point{
		geometry.Pt(3.05, 1.65), geometry.Pt(12.95, 8.25), geometry.Pt(3.05, 14.85),
	}),
	IconButtonUp: newDirectionalArrow([3]geometry.Point{
		geometry.Pt(1.4, 13.2), geometry.Pt(8, 3.3), geometry.Pt(14.6, 13.2),
	}),
	IconButtonDown: newDirectionalArrow([3]geometry.Point{
		geometry.Pt(1.4, 3.3), geometry.Pt(8, 13.2), geometry.Pt(14.6, 3.3),
	}),
}

func isDirectionalIconButton(kind IconButtonKind) bool {
	return kind >= IconButtonLeft && kind <= IconButtonDown
}

func directionalIconButtonColors(hovered, pressed, disabled bool, background *widget.Color) (fill, border widget.Color) {
	_, fill = lighterTitleBarGradient(2)
	if background != nil {
		fill = *background
	}
	if hovered {
		_, fill = lighterTitleBarGradient(4)
	}
	if pressed {
		fill = Default.Colors.ButtonDown
	}
	border = Default.Colors.ButtonBorder
	if disabled {
		fill = Default.Colors.Disabled
		border = Default.Colors.FooterLine
	}
	return fill, border
}

func drawDirectionalIconButton(canvas widget.Canvas, bounds geometry.Rect, kind IconButtonKind, fill, border widget.Color) {
	spec, ok := directionalArrowFor(kind)
	if !ok || bounds.IsEmpty() {
		return
	}
	if filler, ok := canvas.(widget.SVGFiller); ok {
		filler.FillSVGPath(spec.borderPath, IconButtonSize, bounds, border)
		filler.FillSVGPath(spec.fillPath, IconButtonSize, bounds, fill)
		return
	}
	points := scaleDirectionalArrow(bounds, spec.points)
	canvas.DrawLine(points[0], points[1], border, 1)
	canvas.DrawLine(points[1], points[2], border, 1)
	canvas.DrawLine(points[2], points[0], border, 1)
}

func directionalArrowFor(kind IconButtonKind) (directionalArrowSpec, bool) {
	if kind < IconButtonLeft || kind > IconButtonDown {
		return directionalArrowSpec{}, false
	}
	return directionalArrows[kind], true
}

func newDirectionalArrow(points [3]geometry.Point) directionalArrowSpec {
	fillPoints := insetTriangle(points, directionalArrowBorderInset)
	return directionalArrowSpec{
		borderPath: roundedTrianglePath(points, directionalArrowCornerFraction),
		fillPath:   roundedTrianglePath(fillPoints, directionalArrowCornerFraction),
		points:     points,
	}
}

func insetTriangle(points [3]geometry.Point, inset float32) [3]geometry.Point {
	center := geometry.Pt(
		(points[0].X+points[1].X+points[2].X)/3,
		(points[0].Y+points[1].Y+points[2].Y)/3,
	)
	type line struct {
		point     geometry.Point
		direction geometry.Point
	}
	var edges [3]line
	for i := range points {
		start, end := points[i], points[(i+1)%len(points)]
		direction := geometry.Pt(end.X-start.X, end.Y-start.Y)
		length := float32(math.Hypot(float64(direction.X), float64(direction.Y)))
		if length == 0 {
			return points
		}
		normal := geometry.Pt(-direction.Y/length, direction.X/length)
		midpoint := geometry.Pt((start.X+end.X)/2, (start.Y+end.Y)/2)
		if (center.X-midpoint.X)*normal.X+(center.Y-midpoint.Y)*normal.Y < 0 {
			normal = geometry.Pt(-normal.X, -normal.Y)
		}
		edges[i] = line{
			point:     geometry.Pt(start.X+normal.X*inset, start.Y+normal.Y*inset),
			direction: direction,
		}
	}
	var insetPoints [3]geometry.Point
	for i := range points {
		previous := edges[(i+len(edges)-1)%len(edges)]
		current := edges[i]
		intersection, ok := lineIntersection(previous.point, previous.direction, current.point, current.direction)
		if !ok {
			return points
		}
		insetPoints[i] = intersection
	}
	return insetPoints
}

func lineIntersection(a, aDirection, b, bDirection geometry.Point) (geometry.Point, bool) {
	denominator := crossProduct(aDirection, bDirection)
	if denominator == 0 {
		return geometry.Point{}, false
	}
	delta := geometry.Pt(b.X-a.X, b.Y-a.Y)
	distance := crossProduct(delta, bDirection) / denominator
	return geometry.Pt(a.X+distance*aDirection.X, a.Y+distance*aDirection.Y), true
}

func crossProduct(a, b geometry.Point) float32 {
	return a.X*b.Y - a.Y*b.X
}

func roundedTrianglePath(points [3]geometry.Point, cornerFraction float32) string {
	a, b, c := points[0], points[1], points[2]
	aFromC, aFromB := pointToward(a, c, cornerFraction), pointToward(a, b, cornerFraction)
	bFromA, bFromC := pointToward(b, a, cornerFraction), pointToward(b, c, cornerFraction)
	cFromB, cFromA := pointToward(c, b, cornerFraction), pointToward(c, a, cornerFraction)
	return fmt.Sprintf(
		"M%.3f %.3fQ%.3f %.3f %.3f %.3fL%.3f %.3fQ%.3f %.3f %.3f %.3fL%.3f %.3fQ%.3f %.3f %.3f %.3fZ",
		aFromC.X, aFromC.Y, a.X, a.Y, aFromB.X, aFromB.Y,
		bFromA.X, bFromA.Y, b.X, b.Y, bFromC.X, bFromC.Y,
		cFromB.X, cFromB.Y, c.X, c.Y, cFromA.X, cFromA.Y,
	)
}

func pointToward(from, to geometry.Point, fraction float32) geometry.Point {
	return geometry.Pt(
		from.X+(to.X-from.X)*fraction,
		from.Y+(to.Y-from.Y)*fraction,
	)
}

func scaleDirectionalArrow(bounds geometry.Rect, points [3]geometry.Point) [3]geometry.Point {
	size := bounds.Width()
	if bounds.Height() < size {
		size = bounds.Height()
	}
	scale := size / IconButtonSize
	offsetX := bounds.Min.X + (bounds.Width()-IconButtonSize*scale)/2
	offsetY := bounds.Min.Y + (bounds.Height()-IconButtonSize*scale)/2
	for i, point := range points {
		points[i] = geometry.Pt(offsetX+point.X*scale, offsetY+point.Y*scale)
	}
	return points
}
