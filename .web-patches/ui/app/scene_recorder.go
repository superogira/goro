package app

import (
	"github.com/gogpu/gg/scene"
	internalRender "github.com/gogpu/ui/internal/render"
	"github.com/gogpu/ui/widget"
)

func init() {
	// Register the SceneRecorder factory so that widget.DrawTree can create
	// recording canvases for WidgetBase-based repaint boundaries (ADR-024).
	//
	// The widget package cannot import internal/render (circular dep), so
	// we inject the factory here during package initialization.
	widget.RegisterSceneRecorder(func(s *scene.Scene, width, height int) (widget.Canvas, func()) {
		recorder := internalRender.NewSceneCanvas(s, width, height)
		return recorder, recorder.Close
	})
}
