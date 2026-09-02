package progress

import (
	"testing"

	"github.com/gogpu/gg/scene"
	"github.com/gogpu/ui/geometry"
	internalRender "github.com/gogpu/ui/internal/render"
	"github.com/gogpu/ui/widget"
)

// TestSpinner_SceneRecordingProducesContent verifies that recording
// an indeterminate spinner into a SceneCanvas produces a non-empty scene.
// If this fails, the spinner is invisible in compositor pipeline because
// its PictureLayer has an empty scene.
func TestSpinner_SceneRecordingProducesContent(t *testing.T) {
	w := New(Indeterminate(true), Size(48))

	ctx := widget.NewContext()
	ctx.SetOnInvalidateRect(func(_ geometry.Rect) {})

	constraints := geometry.Constraints{
		MinWidth: 48, MaxWidth: 48,
		MinHeight: 48, MaxHeight: 48,
	}
	w.Layout(ctx, constraints)
	w.SetBounds(geometry.NewRect(0, 0, 48, 48))

	// Record into SceneCanvas (same path as compositor PaintBoundaryLayers).
	sc := scene.NewScene()
	recorder := internalRender.NewSceneCanvas(sc, 48, 48)

	w.Draw(ctx, recorder)
	recorder.Close()

	if sc.IsEmpty() {
		t.Error("spinner scene is EMPTY after recording into SceneCanvas; " +
			"spinner will be invisible in compositor pipeline. " +
			"Check if SceneCanvas supports StrokeArc (used by spinner painter)")
	}
}

// TestDeterminate_SceneRecordingProducesContent verifies determinate
// progress also records into SceneCanvas.
func TestDeterminate_SceneRecordingProducesContent(t *testing.T) {
	w := New(Value(0.42), Size(48), ShowLabel(true))

	ctx := widget.NewContext()

	constraints := geometry.Constraints{
		MinWidth: 48, MaxWidth: 48,
		MinHeight: 48, MaxHeight: 48,
	}
	w.Layout(ctx, constraints)
	w.SetBounds(geometry.NewRect(0, 0, 48, 48))

	sc := scene.NewScene()
	recorder := internalRender.NewSceneCanvas(sc, 48, 48)

	w.Draw(ctx, recorder)
	recorder.Close()

	if sc.IsEmpty() {
		t.Error("determinate progress scene is EMPTY after SceneCanvas recording")
	}
}
