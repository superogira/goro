package badge_test

import (
	"testing"

	"github.com/gogpu/ui/core/badge"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
)

// --- Construction & Defaults ---

func TestNew_Defaults(t *testing.T) {
	b := badge.New()

	if !b.IsVisible() {
		t.Error("new badge should be visible")
	}
	if !b.IsEnabled() {
		t.Error("new badge should be enabled")
	}
	if b.Count() != 0 {
		t.Errorf("default count = %d, want 0", b.Count())
	}
}

func TestNew_AppliesOptions(t *testing.T) {
	b := badge.New(badge.Count(7))
	if b.Count() != 7 {
		t.Errorf("Count() = %d, want 7", b.Count())
	}
}

// --- Count resolution & priority ---

func TestResolvedCount_Priority(t *testing.T) {
	roSig := state.NewSignal[int](100)
	rwSig := state.NewSignal[int](50)

	tests := []struct {
		name string
		opts []badge.Option
		want int
	}{
		{"static", []badge.Option{badge.Count(3)}, 3},
		{"fn over static", []badge.Option{badge.Count(3), badge.CountFn(func() int { return 9 })}, 9},
		{"signal over fn", []badge.Option{badge.CountFn(func() int { return 9 }), badge.CountSignal(rwSig)}, 50},
		{"readonly over signal", []badge.Option{badge.CountSignal(rwSig), badge.CountReadonlySignal(roSig)}, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := badge.New(tt.opts...)
			if got := b.Count(); got != tt.want {
				t.Errorf("Count() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResolvedCount_NegativeClamped(t *testing.T) {
	if got := badge.New(badge.Count(-5)).Count(); got != 0 {
		t.Errorf("negative count = %d, want clamped to 0", got)
	}
	if got := badge.New(badge.CountFn(func() int { return -1 })).Count(); got != 0 {
		t.Errorf("negative fn count = %d, want clamped to 0", got)
	}
}

func TestSetCount(t *testing.T) {
	b := badge.New()
	b.SetCount(4)
	if b.Count() != 4 {
		t.Errorf("Count() = %d, want 4", b.Count())
	}
	b.SetCount(-2)
	if b.Count() != 0 {
		t.Errorf("Count() after negative = %d, want 0", b.Count())
	}
}

// --- Hidden logic ---

func TestIsHidden(t *testing.T) {
	tests := []struct {
		name string
		opts []badge.Option
		want bool
	}{
		{"zero count hidden", []badge.Option{badge.Count(0)}, true},
		{"positive count visible", []badge.Option{badge.Count(1)}, false},
		{"zero with ShowZero visible", []badge.Option{badge.Count(0), badge.ShowZero(true)}, false},
		{"dot always visible", []badge.Option{badge.Dot(true), badge.Count(0)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := badge.New(tt.opts...).IsHidden(); got != tt.want {
				t.Errorf("IsHidden() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Layout ---

func TestLayout_HiddenIsZeroSize(t *testing.T) {
	b := badge.New(badge.Count(0))
	size := uitest.LayoutWidget(b, 100, 100)
	if size.Width != 0 || size.Height != 0 {
		t.Errorf("hidden badge size = %v, want 0x0", size)
	}
}

func TestLayout_DotIsSquare(t *testing.T) {
	b := badge.New(badge.Dot(true))
	size := uitest.LayoutWidget(b, 100, 100)
	if size.Width != size.Height {
		t.Errorf("dot size = %v, want square", size)
	}
	if size.Width <= 0 {
		t.Errorf("dot width = %v, want > 0", size.Width)
	}
}

func TestLayout_SingleDigitIsCircular(t *testing.T) {
	b := badge.New(badge.Count(5))
	size := uitest.LayoutWidget(b, 100, 100)
	if size.Width != size.Height {
		t.Errorf("single-digit size = %v, want square (circular)", size)
	}
}

func TestLayout_MultiDigitIsWiderThanTall(t *testing.T) {
	b := badge.New(badge.Count(100), badge.Max(99)) // renders "99+"
	size := uitest.LayoutWidget(b, 100, 100)
	if size.Width <= size.Height {
		t.Errorf("multi-digit size = %v, want width > height", size)
	}
}

func TestLayout_PaddingAddsSize(t *testing.T) {
	base := uitest.LayoutWidget(badge.New(badge.Dot(true)), 100, 100)
	padded := uitest.LayoutWidget(badge.New(badge.Dot(true)).Padding(4), 100, 100)
	if padded.Width != base.Width+8 || padded.Height != base.Height+8 {
		t.Errorf("padded size = %v, want base %v + 8 each axis", padded, base)
	}
}

func TestLayout_RespectsConstraints(t *testing.T) {
	b := badge.New(badge.Count(100000)) // very wide label
	size := uitest.LayoutWidget(b, 10, 10)
	if size.Width > 10 || size.Height > 10 {
		t.Errorf("size = %v, want constrained to 10x10", size)
	}
}

// --- Draw ---

func drawAt(b *badge.Widget, w, h float32) *uitest.MockCanvas {
	b.SetBounds(geometry.NewRect(0, 0, w, h))
	return uitest.DrawWidget(b)
}

func TestDraw_DotDrawsCircle(t *testing.T) {
	b := badge.New(badge.Dot(true))
	canvas := drawAt(b, 8, 8)

	if len(canvas.Circles) != 1 {
		t.Fatalf("dot draw circles = %d, want 1", len(canvas.Circles))
	}
	if len(canvas.Texts) != 0 {
		t.Errorf("dot draw should not draw text, got %d", len(canvas.Texts))
	}
	if canvas.Circles[0].Radius != 4 {
		t.Errorf("dot radius = %v, want 4 (min(8,8)/2)", canvas.Circles[0].Radius)
	}
}

func TestDraw_DotRadiusUsesSmallerDimension(t *testing.T) {
	b := badge.New(badge.Dot(true))
	canvas := drawAt(b, 20, 8) // non-square bounds set directly
	if len(canvas.Circles) != 1 {
		t.Fatalf("circles = %d, want 1", len(canvas.Circles))
	}
	if canvas.Circles[0].Radius != 4 {
		t.Errorf("radius = %v, want 4 (min(20,8)/2)", canvas.Circles[0].Radius)
	}
}

func TestDraw_CountDrawsPillAndText(t *testing.T) {
	b := badge.New(badge.Count(5))
	canvas := drawAt(b, 16, 16)

	if len(canvas.RoundRects) != 1 {
		t.Fatalf("count draw round rects = %d, want 1", len(canvas.RoundRects))
	}
	if canvas.RoundRects[0].Radius != 8 {
		t.Errorf("pill radius = %v, want 8 (height/2)", canvas.RoundRects[0].Radius)
	}
	uitest.AssertDrawnText(t, canvas, "5")
}

func TestDraw_CountOverMaxRendersPlus(t *testing.T) {
	b := badge.New(badge.Count(250), badge.Max(99))
	canvas := drawAt(b, 30, 16)
	uitest.AssertDrawnText(t, canvas, "99+")
}

func TestDraw_DefaultMaxIs99(t *testing.T) {
	b := badge.New(badge.Count(100)) // no Max option
	canvas := drawAt(b, 30, 16)
	uitest.AssertDrawnText(t, canvas, "99+")
}

func TestDraw_MaxNonPositiveIgnored(t *testing.T) {
	b := badge.New(badge.Count(100), badge.Max(0)) // Max(0) ignored -> default 99
	canvas := drawAt(b, 30, 16)
	uitest.AssertDrawnText(t, canvas, "99+")
}

func TestDraw_ShowZeroDrawsZero(t *testing.T) {
	b := badge.New(badge.Count(0), badge.ShowZero(true))
	canvas := drawAt(b, 16, 16)
	uitest.AssertDrawnText(t, canvas, "0")
}

func TestDraw_HiddenDrawsNothing(t *testing.T) {
	b := badge.New(badge.Count(0))
	canvas := drawAt(b, 16, 16)
	if canvas.TotalDrawCalls() != 0 {
		t.Errorf("hidden badge draw calls = %d, want 0", canvas.TotalDrawCalls())
	}
}

func TestDraw_DisabledUsesDisabledColors(t *testing.T) {
	enabled := drawAt(badge.New(badge.Count(3)), 16, 16)
	disabled := drawAt(badge.New(badge.Count(3), badge.Disabled(true)), 16, 16)

	if enabled.RoundRects[0].Color == disabled.RoundRects[0].Color {
		t.Error("disabled badge should use a different background color")
	}
}

func TestDraw_ColorSchemeOverridesDefaults(t *testing.T) {
	scheme := badge.BadgeColorScheme{
		Background: widget.Hex(0x00FF00),
		Label:      widget.Hex(0x000000),
	}
	canvas := drawAt(badge.New(badge.Count(3), badge.ColorSchemeOpt(scheme)), 16, 16)
	uitest.AssertColorEqual(t, canvas.RoundRects[0].Color, widget.Hex(0x00FF00))
}

func TestDraw_PaddingInsetsContent(t *testing.T) {
	b := badge.New(badge.Dot(true)).Padding(3)
	b.SetBounds(geometry.NewRect(0, 0, 14, 14)) // 8 content + 3 padding each side
	canvas := uitest.DrawWidget(b)
	if len(canvas.Circles) != 1 {
		t.Fatalf("circles = %d, want 1", len(canvas.Circles))
	}
	// Content bounds are inset by padding: 14 - 6 = 8 -> radius 4, centered at 7,7.
	center := canvas.Circles[0].Center
	if center.X != 7 || center.Y != 7 {
		t.Errorf("circle center = %v, want (7,7)", center)
	}
}

// --- Custom painter ---

type recordPainter struct{ called bool }

func (p *recordPainter) PaintBadge(_ widget.Canvas, _ badge.PaintState) { p.called = true }

func TestPainterOpt_UsesCustomPainter(t *testing.T) {
	p := &recordPainter{}
	b := badge.New(badge.Count(1), badge.PainterOpt(p))
	drawAt(b, 16, 16)
	if !p.called {
		t.Error("custom painter PaintBadge was not called")
	}
}

// --- DefaultPainter edge cases ---

func TestDefaultPainter_EmptyBoundsNoOp(t *testing.T) {
	canvas := &uitest.MockCanvas{}
	badge.DefaultPainter{}.PaintBadge(canvas, badge.PaintState{
		Bounds: geometry.Rect{},
		Label:  "5",
	})
	if canvas.TotalDrawCalls() != 0 {
		t.Errorf("empty bounds draw calls = %d, want 0", canvas.TotalDrawCalls())
	}
}

func TestDraw_DotRadiusUsesSmallerHeight(t *testing.T) {
	b := badge.New(badge.Dot(true))
	canvas := drawAt(b, 8, 20) // height > width: radius driven by width
	if len(canvas.Circles) != 1 {
		t.Fatalf("circles = %d, want 1", len(canvas.Circles))
	}
	if canvas.Circles[0].Radius != 4 {
		t.Errorf("radius = %v, want 4 (min(8,20)/2)", canvas.Circles[0].Radius)
	}
}

// --- Leaf widget behavior ---

func TestEvent_NeverConsumed(t *testing.T) {
	b := badge.New(badge.Count(1))
	if uitest.SimulateClick(b, 5, 5) {
		t.Error("badge should never consume events")
	}
}

func TestChildren_Nil(t *testing.T) {
	if badge.New().Children() != nil {
		t.Error("badge should have no children")
	}
}

// --- Mount / signal bindings ---

type mockScheduler struct{ dirtyCount int }

func (s *mockScheduler) MarkDirty(_ widget.Widget) { s.dirtyCount++ }

func TestMount_CountSignalNotifies(t *testing.T) {
	sig := state.NewSignal[int](1)
	b := badge.New(badge.CountSignal(sig))

	sched := &mockScheduler{}
	ctx := widget.NewContext()
	ctx.SetScheduler(sched)
	b.Mount(ctx)

	sig.Set(2)
	if sched.dirtyCount == 0 {
		t.Error("scheduler should be notified after count change")
	}
}

func TestMount_ReadonlyCountSignalNotifies(t *testing.T) {
	sig := state.NewSignal[int](1)
	b := badge.New(badge.CountReadonlySignal(sig))

	sched := &mockScheduler{}
	ctx := widget.NewContext()
	ctx.SetScheduler(sched)
	b.Mount(ctx)

	sig.Set(2)
	if sched.dirtyCount == 0 {
		t.Error("scheduler should be notified after readonly count change")
	}
}

func TestMount_DisabledSignalNotifies(t *testing.T) {
	sig := state.NewSignal(false)
	b := badge.New(badge.Count(1), badge.DisabledSignal(sig))

	sched := &mockScheduler{}
	ctx := widget.NewContext()
	ctx.SetScheduler(sched)
	b.Mount(ctx)

	sig.Set(true)
	if sched.dirtyCount == 0 {
		t.Error("scheduler should be notified after disabled change")
	}
}

func TestMount_ReadonlyDisabledSignalNotifies(t *testing.T) {
	sig := state.NewSignal(false)
	b := badge.New(badge.Count(1), badge.DisabledReadonlySignal(sig))

	sched := &mockScheduler{}
	ctx := widget.NewContext()
	ctx.SetScheduler(sched)
	b.Mount(ctx)

	sig.Set(true)
	if sched.dirtyCount == 0 {
		t.Error("scheduler should be notified after readonly disabled change")
	}
}

func TestMount_CountSignalDedupesUnchanged(t *testing.T) {
	sig := state.NewSignal[int](3)
	b := badge.New(badge.CountSignal(sig))

	// Draw once so lastDrawnCount is populated.
	drawAt(b, 16, 16)

	sched := &mockScheduler{}
	ctx := widget.NewContext()
	ctx.SetScheduler(sched)
	b.Mount(ctx)

	sig.Set(3) // same value -> should be suppressed
	if sched.dirtyCount != 0 {
		t.Errorf("dirtyCount = %d, want 0 for unchanged count", sched.dirtyCount)
	}

	sig.Set(4) // changed -> should notify
	if sched.dirtyCount == 0 {
		t.Error("scheduler should be notified after count actually changes")
	}
}

func TestMount_NilSchedulerNoPanic(t *testing.T) {
	b := badge.New(badge.CountSignal(state.NewSignal[int](1)))
	ctx := widget.NewContext()
	b.Mount(ctx) // scheduler is nil; must not panic
}

func TestUnmount_CleanupStopsNotifications(t *testing.T) {
	sig := state.NewSignal[int](1)
	b := badge.New(badge.CountSignal(sig))

	sched := &mockScheduler{}
	ctx := widget.NewContext()
	ctx.SetScheduler(sched)
	b.Mount(ctx)
	b.CleanupBindings()
	b.Unmount()

	sched.dirtyCount = 0
	sig.Set(99)
	if sched.dirtyCount > 0 {
		t.Error("scheduler should not be notified after cleanup")
	}
}

// --- Fluent & interface compliance ---

func TestPadding_Chaining(t *testing.T) {
	b := badge.New()
	if b.Padding(2) != b {
		t.Error("Padding should return the same widget for chaining")
	}
}

func TestWidget_ImplementsInterfaces(t *testing.T) {
	var _ widget.Widget = badge.New()
	var _ widget.Lifecycle = badge.New()
}

// --- DisabledFn / CountFn resolution coverage ---

func TestResolvedDisabled_Priority(t *testing.T) {
	tests := []struct {
		name string
		opts []badge.Option
		want bool
	}{
		{"static", []badge.Option{badge.Disabled(true)}, true},
		{"fn over static", []badge.Option{badge.Disabled(false), badge.DisabledFn(func() bool { return true })}, true},
		{"signal over fn", []badge.Option{badge.DisabledFn(func() bool { return false }), badge.DisabledSignal(state.NewSignal(true))}, true},
		{"readonly over signal", []badge.Option{badge.DisabledSignal(state.NewSignal(false)), badge.DisabledReadonlySignal(state.NewSignal(true))}, true},
	}
	enabled := drawAt(badge.New(badge.Count(1)), 16, 16)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := badge.New(append(tt.opts, badge.Count(1))...)
			canvas := drawAt(b, 16, 16)
			// When disabled, background differs from the enabled default.
			gotDisabled := canvas.RoundRects[0].Color != enabled.RoundRects[0].Color
			if gotDisabled != tt.want {
				t.Errorf("disabled rendering = %v, want %v", gotDisabled, tt.want)
			}
		})
	}
}

func TestDraw_DisabledColorSchemeOverrides(t *testing.T) {
	scheme := badge.BadgeColorScheme{
		Background:         widget.Hex(0x00FF00),
		Label:              widget.Hex(0x111111),
		DisabledBackground: widget.Hex(0x0000FF),
		DisabledLabel:      widget.Hex(0x222222),
	}
	canvas := drawAt(badge.New(badge.Count(3), badge.Disabled(true), badge.ColorSchemeOpt(scheme)), 16, 16)
	uitest.AssertColorEqual(t, canvas.RoundRects[0].Color, widget.Hex(0x0000FF))
	if len(canvas.Texts) != 1 {
		t.Fatalf("texts = %d, want 1", len(canvas.Texts))
	}
	uitest.AssertColorEqual(t, canvas.Texts[0].Color, widget.Hex(0x222222))
}

func TestMount_CountSignalNegativeClamped(t *testing.T) {
	sig := state.NewSignal[int](2)
	b := badge.New(badge.CountSignal(sig))
	drawAt(b, 16, 16) // populate lastDrawnCount = 2

	sched := &mockScheduler{}
	ctx := widget.NewContext()
	ctx.SetScheduler(sched)
	b.Mount(ctx)

	sig.Set(-5) // clamps to 0, differs from 2 -> should notify
	if sched.dirtyCount == 0 {
		t.Error("scheduler should be notified when negative count clamps to a new value")
	}
}
