package rotheme

import (
	"math"
	"strings"
	"testing"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
)

type svgRecordingCanvas struct {
	*tableViewHeaderCanvas
	fills []struct {
		path    string
		viewBox float32
		bounds  geometry.Rect
		color   widget.Color
	}
}

func (c *svgRecordingCanvas) FillSVGPath(path string, viewBox float32, bounds geometry.Rect, color widget.Color) {
	c.fills = append(c.fills, struct {
		path    string
		viewBox float32
		bounds  geometry.Rect
		color   widget.Color
	}{path: path, viewBox: viewBox, bounds: bounds, color: color})
}

func TestIconButtonGlyphKeepsIntegerXAndQuarterPixelY(t *testing.T) {
	canvas := &tableViewHeaderCanvas{}
	drawIconGlyph(canvas, geometry.NewRect(0, 0, IconButtonSize, IconButtonSize), IconButtonClose, widget.ColorBlack)

	if len(canvas.lines) != 2 {
		t.Fatalf("close icon lines = %d, want 2", len(canvas.lines))
	}
	if got := canvas.lines[0].from; got != geometry.Pt(5, 5.25) {
		t.Fatalf("close icon first point = %v, want 5,5.25", got)
	}
	if got := canvas.lines[0].to; got != geometry.Pt(11, 11.25) {
		t.Fatalf("close icon second point = %v, want 11,11.25", got)
	}
}

func TestIconButtonShadowMatchesSharedButton(t *testing.T) {
	bounds := geometry.NewRect(0, 0, IconButtonSize, IconButtonSize)
	for _, kind := range []IconButtonKind{IconButtonClose, IconButtonPlus, IconButtonMinus} {
		painted := &uitest.MockCanvas{}
		IconButtonPainter{Kind: kind}.PaintButton(painted, button.PaintState{Bounds: bounds})
		direct := &uitest.MockCanvas{}
		DrawIconButton(direct, bounds, kind, false, false)
		if len(painted.RoundRects) != 3 || len(direct.RoundRects) != 3 {
			t.Fatalf("icon %v: expected two shadow layers and one reflection in both draw paths", kind)
		}
		for i, offset := range []float32{2, 1} {
			shadow := painted.RoundRects[i]
			if shadow != direct.RoundRects[i] || shadow.Bounds != bounds.TranslateXY(0, offset) || shadow.Radius != ButtonRadius {
				t.Fatalf("icon %v: inconsistent shadow layer %d: painted %v, direct %v", kind, i, shadow, direct.RoundRects[i])
			}
		}
	}
}

func TestDirectionalIconButtonIsFilledAndBorderedWithoutButtonBox(t *testing.T) {
	bounds := geometry.NewRect(0, 0, IconButtonSize, IconButtonSize)
	canvas := &svgRecordingCanvas{tableViewHeaderCanvas: &tableViewHeaderCanvas{}}
	DrawIconButton(canvas, bounds, IconButtonLeft, false, false)

	if len(canvas.fills) != 2 {
		t.Fatalf("left icon fills = %d, want border and interior", len(canvas.fills))
	}
	border := canvas.fills[0]
	if strings.Count(border.path, "Q") != 3 {
		t.Fatalf("left icon border path = %q, want rounded triangle", border.path)
	}
	if border.viewBox != IconButtonSize || border.bounds != bounds {
		t.Fatalf("left icon border geometry = viewBox %v, bounds %v", border.viewBox, border.bounds)
	}
	if border.color != Default.Colors.ButtonBorder {
		t.Fatalf("left icon border = %v, want %v", border.color, Default.Colors.ButtonBorder)
	}
	fill := canvas.fills[1]
	_, wantFill := lighterTitleBarGradient(2)
	if fill.color != wantFill {
		t.Fatalf("left icon fill = %v, want %v", fill.color, wantFill)
	}
	if len(canvas.lines) != 0 {
		t.Fatalf("rounded directional icon drew %d sharp border lines, want none", len(canvas.lines))
	}
	if len(canvas.rects) != 0 {
		t.Fatalf("directional icon drew %d rectangular backgrounds, want none", len(canvas.rects))
	}
}

func TestDirectionalIconButtonBorderInsetIsUniform(t *testing.T) {
	outer := directionalArrows[IconButtonLeft].points
	inner := insetTriangle(outer, directionalArrowBorderInset)
	for i := range outer {
		start, end := outer[i], outer[(i+1)%len(outer)]
		direction := geometry.Pt(end.X-start.X, end.Y-start.Y)
		toInner := geometry.Pt(inner[i].X-start.X, inner[i].Y-start.Y)
		distance := float32(math.Abs(float64(crossProduct(direction, toInner)))) /
			float32(math.Hypot(float64(direction.X), float64(direction.Y)))
		if math.Abs(float64(distance-directionalArrowBorderInset)) > 0.001 {
			t.Fatalf("edge %d inset = %.3f, want %.3f", i, distance, directionalArrowBorderInset)
		}
	}
}

func TestIconButtonMouseButtonKeepsFullSurfaceSize(t *testing.T) {
	root := IconButton(IconButtonClose, nil)
	size := root.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(IconButtonSize, IconButtonSize)))
	if size.Width != IconButtonSize || size.Height != IconButtonSize {
		t.Fatalf("icon button size = %v, want %dx%d", size, int(IconButtonSize), int(IconButtonSize))
	}
	children := root.Children()
	if len(children) != 1 {
		t.Fatalf("icon button children = %d, want 1", len(children))
	}
	bounder, ok := children[0].(interface{ Bounds() geometry.Rect })
	if !ok {
		t.Fatalf("icon button child does not expose bounds")
	}
	bounds := bounder.Bounds()
	if bounds.Width() != IconButtonSize || bounds.Height() != IconButtonSize {
		t.Fatalf("icon button child bounds = %v, want %dx%d", bounds, int(IconButtonSize), int(IconButtonSize))
	}
	inner := children[0].Children()
	if len(inner) != 1 {
		t.Fatalf("icon button delegated children = %d, want 1", len(inner))
	}
	innerBounder, ok := inner[0].(interface{ Bounds() geometry.Rect })
	if !ok {
		t.Fatalf("icon button delegated child does not expose bounds")
	}
	innerBounds := innerBounder.Bounds()
	if innerBounds.Width() != IconButtonSize || innerBounds.Height() != IconButtonSize {
		t.Fatalf("icon button delegated child bounds = %v, want %dx%d", innerBounds, int(IconButtonSize), int(IconButtonSize))
	}
}
