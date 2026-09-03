package ui

import (
	"fmt"
	"image/color"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	perfHUDWidth   = 260
	perfHUDHeight  = 60
	perfHUDSample  = 500 * time.Millisecond
	perfHUDFont    = 13
	perfHUDPadding = 6
)

var (
	perfHUDBackdrop = Color(color.RGBA{R: 12, G: 16, B: 22, A: 140})
	perfHUDFg       = Color(color.RGBA{R: 180, G: 255, B: 200, A: 255})
)

// PerfHUD is a small always-on overlay in the bottom-left corner showing
// frame rate and memory use. Tablets run out of memory mid-session and the
// tab dies silently; watching these numbers climb is the fastest diagnosis.
type PerfHUD struct {
	widget   *perfHUDWidget
	root     widget.Widget
	visible  bool
	x, y     int
	lines    [3]string
	frames   int
	lastTick time.Time
	fps      float64
	frameMS  float64
}

func (h *PerfHUD) Update(ctx Context) {
	if ctx.UIManager == nil {
		return
	}
	now := time.Now()
	if h.lastTick.IsZero() {
		h.lastTick = now
	}
	h.frames++
	if elapsed := now.Sub(h.lastTick); elapsed >= perfHUDSample {
		h.fps = float64(h.frames) / elapsed.Seconds()
		h.frameMS = float64(elapsed.Milliseconds()) / float64(h.frames)
		h.frames = 0
		h.lastTick = now
		h.refreshLines()
	}
	if h.widget == nil {
		h.widget = newPerfHUDWidget()
	}
	width, height := ctx.ScreenSize()
	x, y := perfHUDBounds(width, height)
	if h.root == nil || h.x != x || h.y != y {
		h.Unpublish(ctx)
		h.x, h.y = x, y
		h.root = positionedWidget(h.widget, x, y, perfHUDWidth, perfHUDHeight)
		if enabled, ok := h.root.(interface{ SetEnabled(bool) }); ok {
			enabled.SetEnabled(false)
		}
		ctx.UIManager.AddOverlay(h.root)
		h.visible = true
	}
}

func (h *PerfHUD) refreshLines() {
	mem := perfMemStats()
	h.lines[0] = fmt.Sprintf("FPS %.1f (%.1fms)", h.fps, h.frameMS)
	h.lines[1] = fmt.Sprintf("Go %.1fMB sys %.1fMB gc %d", mem.goHeapMB, mem.goSysMB, mem.goGC)
	h.lines[2] = mem.extra
	if h.widget != nil {
		h.widget.lines = h.lines
		h.widget.SetNeedsRedraw(true)
		if redraw, ok := h.root.(interface{ SetNeedsRedraw(bool) }); ok {
			redraw.SetNeedsRedraw(true)
		}
	}
}

func (h *PerfHUD) Unpublish(ctx Context) {
	if !h.visible || h.root == nil || ctx.UIManager == nil {
		return
	}
	ctx.UIManager.RemoveOverlay(h.root)
	h.root = nil
	h.visible = false
}

func perfHUDBounds(width, height int) (int, int) {
	// Bottom-right: the chat bar occupies the bottom-left corner.
	x := width - perfHUDWidth
	if x < 0 {
		x = 0
	}
	y := height - perfHUDHeight
	if y < 0 {
		y = 0
	}
	return x, y
}

type perfHUDWidget struct {
	widget.WidgetBase
	lines [3]string
}

func newPerfHUDWidget() *perfHUDWidget {
	w := &perfHUDWidget{}
	w.SetVisible(true)
	w.SetEnabled(false)
	return w
}

func (w *perfHUDWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.BiggestFinite(perfHUDWidth, perfHUDHeight)
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *perfHUDWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	canvas.DrawRect(bounds, perfHUDBackdrop)
	lineH := float32(perfHUDFont + 3)
	for i, line := range w.lines {
		if line == "" {
			continue
		}
		row := geometry.Rect{
			Min: geometry.Pt(bounds.Min.X+perfHUDPadding, bounds.Min.Y+float32(i)*lineH),
			Max: geometry.Pt(bounds.Max.X, bounds.Min.Y+float32(i+1)*lineH),
		}
		rotheme.DrawText(canvas, line, row, perfHUDFont, perfHUDFg, false, widget.TextAlignLeft)
	}
}

func (w *perfHUDWidget) Event(_ widget.Context, _ event.Event) bool {
	return false
}
