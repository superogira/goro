package ui

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/ui/rotheme"
)

func TestFooterStripesStayBehindContentAndWithinBounds(t *testing.T) {
	child := primitives.Box().Background(widget.ColorRed)
	child.SetBounds(geometry.NewRect(2, 4, 10, 4))
	footer := &roFooterWidget{BoxWidget: primitives.Box(child)}
	bounds := geometry.NewRect(3, 5, 80, 9)
	footer.SetBounds(bounds)
	canvas := &uitest.MockCanvas{}
	canvas.PushTransform(geometry.Pt(30, 40))
	footer.Draw(widget.NewContext(), canvas)

	if len(canvas.Rects) != 5 {
		t.Fatalf("rectangle draws = %d, want background, three stripes, then content", len(canvas.Rects))
	}
	if canvas.Rects[0].Bounds != bounds || canvas.Rects[0].Color != rotheme.Default.Colors.WindowFooter {
		t.Fatalf("footer background = %v, want existing grey over %v", canvas.Rects[0], bounds)
	}
	for i, want := range []geometry.Rect{
		geometry.NewRect(3, 5, 80, 2),
		geometry.NewRect(3, 9, 80, 2),
		geometry.NewRect(3, 13, 80, 1),
	} {
		stripe := canvas.Rects[i+1]
		if stripe.Bounds != want || stripe.Color != widget.RGBA(1, 1, 1, 0.5) {
			t.Errorf("stripe %d = %v, want 50%% white over %v", i, stripe, want)
		}
	}
	if last := canvas.Rects[4]; last.Color != widget.ColorRed {
		t.Fatal("footer stripes painted over the content")
	}
	if got, want := child.ScreenBounds(), geometry.NewRect(35, 49, 10, 4); got != want {
		t.Fatalf("footer child screen bounds = %v, want %v", got, want)
	}
}

func TestFooterStripesSkipInvisibleAndEmptyFooter(t *testing.T) {
	footer := &roFooterWidget{BoxWidget: primitives.Box()}
	canvas := &uitest.MockCanvas{}
	footer.Draw(widget.NewContext(), canvas)
	footer.SetBounds(geometry.NewRect(0, 0, 80, 41))
	footer.SetVisible(false)
	footer.Draw(widget.NewContext(), canvas)
	if len(canvas.Rects) != 0 {
		t.Fatal("invisible or empty footer painted a background")
	}
}
