package state

import (
	"sync"

	"github.com/gogpu/ui/widget"
)

// Binding connects a reactive signal to a widget's invalidation lifecycle.
//
// When the bound signal's value changes the widget's context is invalidated,
// which marks the widget for re-render. A Binding must be cleaned up via
// [Binding.Unbind] when the widget is removed from the tree; otherwise the
// subscription leaks.
//
// Create a Binding with [Bind].
type Binding struct {
	mu      sync.Mutex
	cleanup Unsubscribe
	active  bool
}

// Unbind stops the binding so that future signal changes no longer
// invalidate the widget. Safe to call multiple times; subsequent calls
// are no-ops.
func (b *Binding) Unbind() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.active {
		return
	}
	b.active = false
	if b.cleanup != nil {
		b.cleanup()
		b.cleanup = nil
	}
}

// IsActive reports whether the binding is still active (not yet unbound).
func (b *Binding) IsActive() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.active
}

// Bind creates a [Binding] that invalidates ctx whenever sig changes.
//
// Deprecated: Bind triggers full-window layout+redraw via ctx.Invalidate().
// Use [BindToScheduler] for granular per-widget invalidation (enterprise pattern).
//
// The type parameter T must match the signal's value type. The binding
// subscribes to the signal using SubscribeForever; the caller must call
// [Binding.Unbind] to release the subscription.
//
// Example:
//
//	counter := state.NewSignal(0)
//	binding := state.Bind(counter, ctx)
//	defer binding.Unbind()
//
//	counter.Set(1) // ctx.Invalidate() is called automatically
func Bind[T any](sig ReadonlySignal[T], ctx widget.Context) *Binding {
	b := &Binding{active: true}
	unsub := sig.SubscribeForever(func(_ T) {
		ctx.Invalidate()
	})
	b.cleanup = unsub
	return b
}

// BindToScheduler creates a [Binding] that marks w as dirty in sched
// whenever sig changes.
//
// Use this instead of [Bind] when you want fine-grained control over
// render batching. The scheduler collects dirty widgets and processes
// them in a single flush.
//
// Example:
//
//	counter := state.NewSignal(0)
//	sched := state.NewScheduler(flushFn)
//	binding := state.BindToScheduler(counter, myWidget, sched)
//	defer binding.Unbind()
//
//	counter.Set(1) // sched.MarkDirty(myWidget) is called
func BindToScheduler[T any](sig ReadonlySignal[T], w widget.Widget, sched widget.SchedulerRef) *Binding {
	b := &Binding{active: true}
	unsub := sig.SubscribeForever(func(_ T) {
		sched.MarkDirty(w)
	})
	b.cleanup = unsub
	return b
}

// BindToSchedulerFunc creates a [Binding] that marks w as dirty in sched
// when sig changes AND the predicate returns true.
//
// The predicate receives the new signal value and returns true if the widget
// should be marked dirty. This allows callers to suppress redundant redraws
// when the signal fires with unchanged values.
//
// Example:
//
//	binding := state.BindToSchedulerFunc(sig, func(v float64) bool {
//	    return v != cachedValue  // only dirty when value actually changed
//	}, myWidget, sched)
func BindToSchedulerFunc[T any](sig ReadonlySignal[T], shouldDirty func(T) bool, w widget.Widget, sched widget.SchedulerRef) *Binding {
	b := &Binding{active: true}
	unsub := sig.SubscribeForever(func(v T) {
		if shouldDirty(v) {
			sched.MarkDirty(w)
		}
	})
	b.cleanup = unsub
	return b
}

// layoutInvalidator is satisfied by widgets that embed [widget.WidgetBase] and
// support layout cache invalidation (ADR-032). Used by [BindToSchedulerLayout]
// via type assertion so [widget.SchedulerRef] stays unchanged (no mock breakage).
type layoutInvalidator interface {
	MarkNeedsLayout()
}

// BindToSchedulerLayout creates a [Binding] that marks w as dirty in sched
// AND invalidates its layout cache whenever sig changes.
//
// Use this for signals whose value affects the widget's measured size (text
// content, item count, column count, ratio). Paint-only signals (color,
// checked state, scroll offset) should use [BindToScheduler] instead.
//
// The layout invalidation uses a type assertion (not an interface extension)
// so that [widget.SchedulerRef] and uitest mocks remain unchanged.
func BindToSchedulerLayout[T any](sig ReadonlySignal[T], w widget.Widget, sched widget.SchedulerRef) *Binding {
	b := &Binding{active: true}
	unsub := sig.SubscribeForever(func(_ T) {
		if lm, ok := w.(layoutInvalidator); ok {
			lm.MarkNeedsLayout()
		}
		widget.InvalidateLayoutTree(w)
		sched.MarkDirty(w)
	})
	b.cleanup = unsub
	return b
}

// BindToSchedulerLayoutFunc creates a [Binding] that marks w as dirty in sched
// AND invalidates its layout cache when sig changes AND the predicate returns true.
//
// This is the layout-aware variant of [BindToSchedulerFunc]. The predicate
// suppresses redundant invalidations when the signal fires with a value that
// doesn't actually change the widget's measured size.
func BindToSchedulerLayoutFunc[T any](sig ReadonlySignal[T], shouldDirty func(T) bool, w widget.Widget, sched widget.SchedulerRef) *Binding {
	b := &Binding{active: true}
	unsub := sig.SubscribeForever(func(v T) {
		if shouldDirty(v) {
			if lm, ok := w.(layoutInvalidator); ok {
				lm.MarkNeedsLayout()
			}
			widget.InvalidateLayoutTree(w)
			sched.MarkDirty(w)
		}
	})
	b.cleanup = unsub
	return b
}
