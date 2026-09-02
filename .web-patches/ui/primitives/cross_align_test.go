package primitives_test

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

func TestVBox_CrossAxisCenter(t *testing.T) {
	ctx := widget.NewContext()

	// 48px wide spinner in 400px wide VBox → should be centered at X=176.
	child := primitives.Text("X").FontSize(14)

	box := primitives.VBox(child).
		CrossAlign(primitives.CrossAxisCenter).
		Padding(0)

	constraints := geometry.BoxConstraints(400, 400, 0, 400)
	box.Layout(ctx, constraints)

	bounds := child.Bounds()
	boxBounds := box.Bounds()
	childW := bounds.Width()
	boxW := boxBounds.Width()
	expectedX := (boxW - childW) / 2

	t.Logf("box bounds: %v (width=%f)", boxBounds, boxW)
	t.Logf("child bounds: %v (width=%f)", bounds, childW)
	t.Logf("expectedX: %f", expectedX)

	if bounds.Min.X < expectedX-2 || bounds.Min.X > expectedX+2 {
		t.Errorf("child X = %f, want ~%f (centered in %fpx VBox, child width=%f)",
			bounds.Min.X, expectedX, boxW, childW)
	}
}

func TestVBox_CrossAxisStart_Default(t *testing.T) {
	ctx := widget.NewContext()

	child := primitives.Text("Left").FontSize(14)

	box := primitives.VBox(child).Padding(0)

	constraints := geometry.BoxConstraints(400, 400, 0, 400)
	box.Layout(ctx, constraints)

	bounds := child.Bounds()

	// Default = start = X should be 0 (left-aligned).
	if bounds.Min.X > 1 {
		t.Errorf("default cross-align: child X = %f, want 0 (left-aligned)", bounds.Min.X)
	}
}

func TestVBox_CrossAxisEnd(t *testing.T) {
	ctx := widget.NewContext()

	child := primitives.Text("Right").FontSize(14)

	box := primitives.VBox(child).
		CrossAlign(primitives.CrossAxisEnd).
		Padding(0)

	constraints := geometry.BoxConstraints(400, 400, 0, 400)
	box.Layout(ctx, constraints)

	bounds := child.Bounds()
	childW := bounds.Width()
	expectedX := 400 - childW

	if bounds.Min.X < expectedX-2 || bounds.Min.X > expectedX+2 {
		t.Errorf("child X = %f, want ~%f (end-aligned, width=%f)",
			bounds.Min.X, expectedX, childW)
	}
}
