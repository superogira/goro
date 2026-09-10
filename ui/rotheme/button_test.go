package rotheme

import (
	"image"
	"image/color"
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
)

func TestMouseOnlyButtonDrawStampsInnerScreenOrigin(t *testing.T) {
	inner := button.New(button.PainterOpt(ButtonPainter{}))
	wrapper := newMouseOnlyButton(inner)
	wrapper.SetBounds(geometry.NewRect(10, 20, 80, 22))
	canvas := &uitest.MockCanvas{}
	canvas.PushTransform(geometry.Pt(100, 200))

	wrapper.Draw(widget.NewContext(), canvas)

	want := geometry.NewRect(110, 220, 80, 22)
	if got := inner.ScreenBounds(); got != want {
		t.Fatalf("inner button screen bounds = %v, want %v for correct hover invalidation", got, want)
	}
}

func TestMouseOnlyButtonHoverEventChangesGradient(t *testing.T) {
	inner := button.New(button.PainterOpt(ButtonPainter{}))
	wrapper := newMouseOnlyButton(inner)
	bounds := geometry.NewRect(0, 0, 80, 22)
	wrapper.SetBounds(bounds)

	normalCanvas := &uitest.MockCanvas{}
	wrapper.Draw(widget.NewContext(), normalCanvas)
	normal := color.RGBAModel.Convert(normalCanvas.Images[0].Image.At(0, 21)).(color.RGBA)

	ctx := widget.NewContext()
	wrapper.Event(ctx, event.NewMouseEvent(
		event.MouseEnter,
		event.ButtonNone,
		0,
		bounds.Center(),
		bounds.Center(),
		event.ModNone,
	))
	hoverCanvas := &uitest.MockCanvas{}
	wrapper.Draw(ctx, hoverCanvas)
	hover := color.RGBAModel.Convert(hoverCanvas.Images[0].Image.At(0, 21)).(color.RGBA)

	if hover == normal {
		t.Fatalf("hover gradient color = %v, unchanged from normal", hover)
	}
}

func TestButtonHoverInvalidatesItsScreenPosition(t *testing.T) {
	app := uiapp.New(uiapp.WithRenderMode(uiapp.RenderModeFrameworkManaged))
	themedButton := Button("Hover", nil)
	root := primitives.Box(themedButton).PaddingXY(100, 100)
	app.SetRoot(root)
	app.Frame()

	initialCanvas := &uitest.MockCanvas{}
	if !app.Window().DrawTo(initialCanvas) {
		t.Fatal("initial button frame was not drawn")
	}
	wrapper, ok := themedButton.Children()[0].(*mouseOnlyButtonWidget)
	if !ok {
		t.Fatalf("button child = %T, want mouse-only wrapper", themedButton.Children()[0])
	}
	buttonCenter := wrapper.button.ScreenBounds().Center()
	app.HandleEvent(event.NewMouseEvent(
		event.MouseMove,
		event.ButtonNone,
		0,
		buttonCenter,
		buttonCenter,
		event.ModNone,
	))

	hoverCanvas := &uitest.MockCanvas{}
	if !app.Window().DrawTo(hoverCanvas) {
		t.Fatal("hovered button frame was not drawn")
	}
	// Hover changes the face and border; the unchanged shadow outside the
	// repaint clip remains in the backing image.
	borderBounds := wrapper.button.ScreenBounds().Expand(0.5)
	if clip := app.Window().LastDirtyUnion(); !clip.ContainsRect(borderBounds) {
		t.Fatalf("hover repaint clip %v does not cover button border %v", clip, borderBounds)
	}
	for _, dirty := range app.Window().DirtyRegions() {
		if dirty.Contains(buttonCenter) {
			return
		}
	}
	t.Fatalf("hover dirty regions %v do not cover button center %v", app.Window().DirtyRegions(), buttonCenter)
}

func TestButtonUsesContinuousLightenedTitleBarGradient(t *testing.T) {
	canvas := &uitest.MockCanvas{}
	bounds := geometry.NewRect(3, 5, 80, 22)

	ButtonPainter{}.PaintButton(canvas, button.PaintState{Bounds: bounds})

	if len(canvas.Rects) != 0 || len(canvas.RoundRects) != 3 || len(canvas.Images) != 1 {
		t.Fatalf("button background draws = %d rectangles, %d rounded rectangles, and %d images; want two shadow layers, one gradient and one reflection", len(canvas.Rects), len(canvas.RoundRects), len(canvas.Images))
	}
	call := canvas.Images[0]
	if call.At != bounds.Min || call.Image.Bounds().Size() != image.Pt(80, 22) {
		t.Fatalf("button gradient image = at %v size %v, want at %v size 80x22", call.At, call.Image.Bounds().Size(), bounds.Min)
	}
	assertGradientColor(t, call.Image, 0, 0, expectedLighterColor(Default.Colors.WindowTitle, 2))
	assertGradientColor(t, call.Image, 0, 21, expectedLighterColor(Default.Colors.WindowTitleTop, 2))
	reflect := canvas.RoundRects[2]
	wantReflectBounds := geometry.NewRect(6, 8, 74, 8)
	if reflect.Bounds != wantReflectBounds || reflect.Radius != 3 {
		t.Fatalf("button reflection = bounds %v radius %.1f, want bounds %v radius 3", reflect.Bounds, reflect.Radius, wantReflectBounds)
	}
	uitest.AssertColorEqual(t, reflect.Color, widget.RGBA(1, 1, 1, buttonReflectAlpha))
	if len(canvas.ClipRoundRects) != 1 || canvas.ClipRoundRects[0].Bounds != bounds || canvas.ClipRoundRects[0].Radius != ButtonRadius {
		t.Fatalf("button rounded clips = %v, want bounds %v radius %.1f", canvas.ClipRoundRects, bounds, ButtonRadius)
	}
	if len(canvas.StrokeRoundRects) != 1 {
		t.Fatalf("button borders = %d, want 1", len(canvas.StrokeRoundRects))
	}
}

func TestButtonShadowMatchesShapeInEveryState(t *testing.T) {
	bounds := geometry.NewRect(3, 5, 80, 22)
	radius := float32(4)
	for name, state := range map[string]button.PaintState{
		"normal":   {},
		"hovered":  {Hovered: true},
		"pressed":  {Pressed: true},
		"disabled": {Disabled: true},
	} {
		t.Run(name, func(t *testing.T) {
			state.Bounds = bounds
			state.Radius = &radius
			canvas := &uitest.MockCanvas{}
			ButtonPainter{}.PaintButton(canvas, state)
			if len(canvas.RoundRects) != 3 {
				t.Fatalf("rounded rectangles = %d, want two shadow layers and one reflection", len(canvas.RoundRects))
			}
			for i, want := range []uitest.DrawRoundRectCall{
				{Bounds: geometry.NewRect(3, 7, 80, 22), Color: widget.RGBA(0.4, 0.4, 0.4, 0.15), Radius: radius},
				{Bounds: geometry.NewRect(3, 6, 80, 22), Color: widget.RGBA(0.4, 0.4, 0.4, 0.35), Radius: radius},
			} {
				if got := canvas.RoundRects[i]; got != want {
					t.Fatalf("shadow layer %d = %v, want %v", i, got, want)
				}
			}
		})
	}
}

func TestButtonHoverUsesFourTimesLighterTitleBarGradient(t *testing.T) {
	canvas := &uitest.MockCanvas{}
	ButtonPainter{}.PaintButton(canvas, button.PaintState{
		Bounds:  geometry.NewRect(0, 0, 80, 22),
		Hovered: true,
	})

	if len(canvas.Images) != 1 {
		t.Fatalf("hovered button gradient images = %d, want 1", len(canvas.Images))
	}
	assertGradientColor(t, canvas.Images[0].Image, 0, 0, expectedLighterColor(Default.Colors.WindowTitle, 4))
	assertGradientColor(t, canvas.Images[0].Image, 0, 21, expectedLighterColor(Default.Colors.WindowTitleTop, 4))
}

func TestButtonGradientRasterizesOpaqueAtFractionalScale(t *testing.T) {
	canvas := &uitest.MockCanvas{}
	top, bottom := buttonTitleBarGradient(1)
	drawButtonGradientColors(canvas, geometry.NewRect(0, 0, 80, 22), top, bottom, ButtonRadius)
	rasterizer, ok := canvas.Images[0].Image.(interface {
		RasterizeForScale(scale float32, width, height int) image.Image
	})
	if !ok {
		t.Fatal("button gradient image does not support native scale rasterization")
	}
	scaled := rasterizer.RasterizeForScale(1.25, 100, 28)
	if scaled.Bounds().Size() != image.Pt(100, 28) {
		t.Fatalf("fractional-scale button gradient size = %v, want 100x28", scaled.Bounds().Size())
	}
	for y := 0; y < scaled.Bounds().Dy(); y++ {
		for x := 0; x < scaled.Bounds().Dx(); x++ {
			if alpha := color.RGBAModel.Convert(scaled.At(x, y)).(color.RGBA).A; alpha != 255 {
				t.Fatalf("fractional-scale button pixel %d,%d alpha = %d, want opaque", x, y, alpha)
			}
		}
	}
	assertGradientColor(t, scaled, 0, 0, Default.Colors.WindowTitle)
	assertGradientColor(t, scaled, 0, 27, Default.Colors.WindowTitleTop)
}

func assertGradientColor(t *testing.T, img image.Image, x, y int, want widget.Color) {
	t.Helper()
	got := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
	r, g, b, a := want.RGBA8()
	if got != (color.RGBA{R: r, G: g, B: b, A: a}) {
		t.Fatalf("gradient pixel %d,%d = %v, want rgba(%d,%d,%d,%d)", x, y, got, r, g, b, a)
	}
}

func expectedLighterColor(base widget.Color, factor float32) widget.Color {
	return base.Lerp(widget.RGBA(1, 1, 1, base.A), 1-1/factor)
}
