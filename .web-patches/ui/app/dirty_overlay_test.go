package app

import (
	"testing"

	"github.com/gogpu/ui/core/progress"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// simpleBox is a container for dirty overlay tests.
type simpleBox struct {
	widget.WidgetBase
	kids []widget.Widget
}

func (w *simpleBox) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(400, 300))
}
func (w *simpleBox) Draw(_ widget.Context, canvas widget.Canvas) {
	canvas.DrawRect(w.Bounds(), widget.RGBA8(255, 255, 255, 255))
	for _, child := range w.kids {
		widget.DrawChild(child, nil, canvas)
	}
}
func (w *simpleBox) Event(_ widget.Context, _ event.Event) bool { return false }
func (w *simpleBox) Children() []widget.Widget                  { return w.kids }

// TestDirtyOverlay_SpinnerRegionIs48x48 verifies that the dirty collector
// reports spinner bounds as ~48×48, NOT full parent width.
// This is the test that caught the VBox expansion bug.
func TestDirtyOverlay_SpinnerRegionIs48x48(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	spinner := progress.New(progress.Indeterminate(true), progress.Size(48))

	root := &simpleBox{}
	root.SetVisible(true)
	root.SetBounds(geometry.NewRect(0, 0, 400, 300))
	root.kids = []widget.Widget{spinner}

	win.SetRoot(root)

	ctx := win.Context()
	constraints := geometry.BoxConstraints(0, 400, 0, 300)
	root.Layout(ctx, constraints)

	spinnerConstraints := geometry.BoxConstraints(0, 400, 0, 300)
	spinnerSize := spinner.Layout(ctx, spinnerConstraints)
	spinner.SetBounds(geometry.NewRect(100, 100, spinnerSize.Width, spinnerSize.Height))

	// Spinner size must be 48×48, NOT 400 wide.
	if spinnerSize.Width != 48 {
		t.Errorf("spinner layout width = %v, want 48 (intrinsic, not parent width)", spinnerSize.Width)
	}
	if spinnerSize.Height != 48 {
		t.Errorf("spinner layout height = %v, want 48", spinnerSize.Height)
	}

	// Spinner bounds should be 48×48 at position (100,100).
	bounds := spinner.Bounds()
	bw := bounds.Max.X - bounds.Min.X
	bh := bounds.Max.Y - bounds.Min.Y
	if bw != 48 {
		t.Errorf("spinner bounds width = %v (min=%v max=%v), want 48", bw, bounds.Min.X, bounds.Max.X)
	}
	if bh != 48 {
		t.Errorf("spinner bounds height = %v (min=%v max=%v), want 48", bh, bounds.Min.Y, bounds.Max.Y)
	}

	// Collect dirty regions — spinner should be dirty (indeterminate animates).
	// First frame: all widgets dirty. Collect should report spinner at 48×48.
	win.CollectDirtyRegions()
	regions := win.DirtyRegions()

	// Find a region that matches spinner bounds (48×48 at 100,100).
	foundSpinner := false
	for _, r := range regions {
		w := r.Width()
		h := r.Height()
		if w >= 40 && w <= 56 && h >= 40 && h <= 56 {
			foundSpinner = true
			t.Logf("spinner dirty region: %v (%.0f×%.0f)", r, w, h)
		}
		if w > 100 {
			t.Errorf("dirty region too wide: %v (%.0f×%.0f) — spinner boundary leak", r, w, h)
		}
	}
	if !foundSpinner && len(regions) > 0 {
		t.Logf("dirty regions: %v", regions)
		t.Error("no ~48×48 dirty region found for spinner")
	}
}
