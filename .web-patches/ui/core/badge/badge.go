package badge

import (
	"strconv"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
)

// Widget implements a small notification badge displaying a count or a dot.
//
// A badge is created with [New] using functional options:
//
//	b := badge.New(
//	    badge.Count(5),
//	    badge.Max(99),
//	)
//
// Fluent styling methods may be chained after construction:
//
//	b.Padding(2)
type Widget struct {
	widget.WidgetBase
	cfg     config
	painter Painter

	// Styling overrides set via fluent methods.
	padding float32

	// lastDrawnCount caches the count from the most recent Draw call.
	// Signal bindings compare against this to suppress redundant redraws
	// when the signal fires with an unchanged value.
	lastDrawnCount    int
	lastDrawnCountSet bool
}

// New creates a new badge Widget with the given options.
//
// The returned widget is visible and enabled by default. It is not focusable
// because badges are display-only widgets.
func New(opts ...Option) *Widget {
	w := &Widget{
		painter: DefaultPainter{},
	}
	w.SetVisible(true)
	w.SetEnabled(true)

	for _, opt := range opts {
		opt(&w.cfg)
	}

	// Apply painter from config if set.
	if w.cfg.painter != nil {
		w.painter = w.cfg.painter
	}

	return w
}

// Default sizing values.
const (
	defaultBadgeHeight float32 = 16
	defaultDotDiameter float32 = 8
	defaultFontSize    float32 = 10
	// labelPaddingX is the inner horizontal padding on each side of the
	// count label within the pill.
	labelPaddingX float32 = 5
	// charWidthRatio approximates digit width as a fraction of the font size
	// for layout estimation (digits are slightly wider than this average).
	charWidthRatio float32 = 0.6
)

// SetCount updates the badge's static count. Negative values are treated as
// zero. The widget is marked for redraw when the value changes.
func (w *Widget) SetCount(n int) {
	if n < 0 {
		n = 0
	}
	if w.cfg.count != n {
		w.cfg.count = n
		w.SetNeedsRedraw(true)
	}
}

// Count returns the current resolved count (always non-negative).
func (w *Widget) Count() int {
	return w.cfg.ResolvedCount()
}

// IsHidden reports whether the badge currently renders nothing.
// A count badge is hidden when its count is non-positive and [ShowZero] is
// not set. Dot badges are never hidden.
func (w *Widget) IsHidden() bool {
	if w.cfg.dot {
		return false
	}
	return w.cfg.ResolvedCount() <= 0 && !w.cfg.showZero
}

// label returns the formatted count label for the current count, or the empty
// string in dot mode.
func (w *Widget) label() string {
	if w.cfg.dot {
		return ""
	}
	return formatCount(w.cfg.ResolvedCount(), w.cfg.ResolvedMax())
}

// formatCount formats count, rendering values greater than limit as "limit+".
func formatCount(count, limit int) string {
	if count > limit {
		return strconv.Itoa(limit) + "+"
	}
	return strconv.Itoa(count)
}

// Layout calculates the badge's preferred size within the given constraints.
func (w *Widget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	if w.IsHidden() {
		return constraints.Constrain(geometry.Sz(0, 0))
	}

	var contentW, contentH float32
	if w.cfg.dot {
		contentW = defaultDotDiameter
		contentH = defaultDotDiameter
	} else {
		contentH = defaultBadgeHeight
		text := w.label()
		textWidth := float32(len(text)) * defaultFontSize * charWidthRatio
		contentW = textWidth + labelPaddingX*2
		// Keep a single digit circular by enforcing a minimum width.
		if contentW < contentH {
			contentW = contentH
		}
	}

	preferred := geometry.Sz(
		contentW+w.padding*2,
		contentH+w.padding*2,
	)
	return constraints.Constrain(preferred)
}

// Draw renders the badge to the canvas.
func (w *Widget) Draw(_ widget.Context, canvas widget.Canvas) {
	count := w.cfg.ResolvedCount()
	w.lastDrawnCount = count
	w.lastDrawnCountSet = true

	if w.IsHidden() {
		return
	}

	bounds := w.contentBounds()

	w.painter.PaintBadge(canvas, PaintState{
		Bounds:      bounds,
		Dot:         w.cfg.dot,
		Label:       w.label(),
		Disabled:    w.cfg.ResolvedDisabled(),
		ColorScheme: w.cfg.colorScheme,
	})
}

// contentBounds returns the badge bounds inset by the outer padding.
func (w *Widget) contentBounds() geometry.Rect {
	b := w.Bounds()
	if w.padding == 0 {
		return b
	}
	return geometry.NewRect(
		b.Min.X+w.padding,
		b.Min.Y+w.padding,
		b.Width()-w.padding*2,
		b.Height()-w.padding*2,
	)
}

// Event handles an input event. Badges are display-only and always return
// false (events are never consumed).
func (w *Widget) Event(_ widget.Context, _ event.Event) bool {
	return false
}

// Children returns nil because a badge is a leaf widget.
func (w *Widget) Children() []widget.Widget {
	return nil
}

// Mount creates signal bindings for push-based invalidation.
// Implements [widget.Lifecycle].
//
// The count signal binding deduplicates notifications: if the signal fires
// with the same count that was last drawn, the widget is not marked dirty.
func (w *Widget) Mount(ctx widget.Context) {
	sched := ctx.Scheduler()
	if sched == nil {
		return
	}
	if w.cfg.readonlyCountSignal != nil {
		w.AddBinding(w.bindCountSignal(w.cfg.readonlyCountSignal, sched))
	} else if w.cfg.countSignal != nil {
		w.AddBinding(w.bindCountSignal(w.cfg.countSignal, sched))
	}
	if w.cfg.readonlyDisabledSignal != nil {
		w.AddBinding(state.BindToScheduler(w.cfg.readonlyDisabledSignal, w, sched))
	} else if w.cfg.disabledSignal != nil {
		w.AddBinding(state.BindToScheduler(w.cfg.disabledSignal, w, sched))
	}
}

// bindCountSignal creates a deduplicating binding for an int count signal.
// MarkDirty is only called when the clamped count differs from the last drawn
// count, preventing redundant redraws.
func (w *Widget) bindCountSignal(sig state.ReadonlySignal[int], sched widget.SchedulerRef) *state.Binding {
	return state.BindToSchedulerLayoutFunc(sig, func(newVal int) bool {
		if newVal < 0 {
			newVal = 0
		}
		if w.lastDrawnCountSet && newVal == w.lastDrawnCount {
			return false // suppress: count unchanged
		}
		return true
	}, w, sched)
}

// Unmount is called when the badge is removed from the widget tree.
// Implements [widget.Lifecycle].
func (w *Widget) Unmount() {
	// Bindings are cleaned up automatically by WidgetBase.CleanupBindings().
}

// Padding sets the outer padding around the badge content.
// Returns the widget for method chaining.
func (w *Widget) Padding(v float32) *Widget {
	w.padding = v
	return w
}

// Verify Widget implements required interfaces at compile time.
var (
	_ widget.Widget    = (*Widget)(nil)
	_ widget.Lifecycle = (*Widget)(nil)
)
