package collapsible

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// headerTextWidget is an internal widget that represents the header title
// text as a proper widget in the Children() tree. This enables dirty.Collector
// to track header title changes independently — when TitleSignal updates,
// this widget gets dirty via signal binding, and the collector reports its
// bounds as a dirty region for the cyan overlay.
//
// The widget does NOT draw itself — the Painter.PaintHeader handles all
// header rendering (background, arrow, text). This widget exists solely
// for dirty tracking and screen origin stamping.
type headerTextWidget struct {
	widget.WidgetBase
}

func newHeaderTextWidget() *headerTextWidget {
	w := &headerTextWidget{}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *headerTextWidget) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(0, 0))
}

func (w *headerTextWidget) Draw(_ widget.Context, _ widget.Canvas) {}

func (w *headerTextWidget) Event(_ widget.Context, _ event.Event) bool { return false }
