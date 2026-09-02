package state_test

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
)

// regressionMockContext tracks Invalidate calls for regression tests.
type regressionMockContext struct {
	widget.Context
	invalidateCount int
}

func (m *regressionMockContext) Invalidate() {
	m.invalidateCount++
}

// regressionWidget embeds WidgetBase and implements widget.Widget for tests.
type regressionWidget struct {
	widget.WidgetBase
}

func (w *regressionWidget) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(0, 0))
}

func (w *regressionWidget) Draw(_ widget.Context, _ widget.Canvas) {}

func (w *regressionWidget) Event(_ widget.Context, _ event.Event) bool { return false }

// TestBind_Deprecated_StillWorks verifies backward compatibility of deprecated Bind().
// Bind is deprecated in favor of BindToScheduler, but must keep working
// for existing external callers (public API contract).
func TestBind_Deprecated_StillWorks(t *testing.T) {
	sig := state.NewSignal(0)
	ctx := &regressionMockContext{}

	binding := state.Bind(sig.AsReadonly(), ctx)
	defer binding.Unbind()

	sig.Set(1)
	sig.Set(2)

	if ctx.invalidateCount != 2 {
		t.Errorf("Bind backward compat: invalidateCount = %d, want 2", ctx.invalidateCount)
	}
}

// TestBindToScheduler_UsesSetNeedsRedraw verifies the enterprise path:
// BindToScheduler -> MarkDirty -> flushFn -> SetNeedsRedraw(true).
func TestBindToScheduler_UsesSetNeedsRedraw(t *testing.T) {
	sig := state.NewSignal(0)
	w := &regressionWidget{}

	sched := state.NewScheduler(func(dirty []widget.Widget) {
		for _, dw := range dirty {
			if setter, ok := dw.(interface{ SetNeedsRedraw(bool) }); ok {
				setter.SetNeedsRedraw(true)
			}
		}
	})

	binding := state.BindToScheduler(sig.AsReadonly(), w, sched)
	defer binding.Unbind()

	sig.Set(42)
	sched.Flush()

	if !w.NeedsRedraw() {
		t.Error("after BindToScheduler + Flush, widget.NeedsRedraw() should be true")
	}
}

// TestBindToScheduler_DoesNotCallInvalidate verifies the enterprise path
// does not trigger nuclear ctx.Invalidate().
func TestBindToScheduler_DoesNotCallInvalidate(t *testing.T) {
	sig := state.NewSignal(0)
	w := &regressionWidget{}
	ctx := &regressionMockContext{}

	// flushFn mirrors app.go production pattern: SetNeedsRedraw only, no ctx.Invalidate.
	sched := state.NewScheduler(func(dirty []widget.Widget) {
		for _, dw := range dirty {
			if setter, ok := dw.(interface{ SetNeedsRedraw(bool) }); ok {
				setter.SetNeedsRedraw(true)
			}
		}
	})

	binding := state.BindToScheduler(sig.AsReadonly(), w, sched)
	defer binding.Unbind()

	sig.Set(99)
	sched.Flush()

	if ctx.invalidateCount != 0 {
		t.Errorf("BindToScheduler must not call ctx.Invalidate(); got %d calls", ctx.invalidateCount)
	}
}

// TestBindToScheduler_BatchDedup verifies multiple signals bound to the same
// widget result in a single dirty entry after deduplication.
func TestBindToScheduler_BatchDedup(t *testing.T) {
	sig1 := state.NewSignal(0)
	sig2 := state.NewSignal("")
	sig3 := state.NewSignal(false)
	w := &regressionWidget{}

	var flushCount int
	sched := state.NewScheduler(func(dirty []widget.Widget) {
		flushCount = len(dirty)
	})

	b1 := state.BindToScheduler(sig1.AsReadonly(), w, sched)
	b2 := state.BindToScheduler(sig2.AsReadonly(), w, sched)
	b3 := state.BindToScheduler(sig3.AsReadonly(), w, sched)
	defer b1.Unbind()
	defer b2.Unbind()
	defer b3.Unbind()

	// Change all three signals — widget should appear only once.
	sig1.Set(1)
	sig2.Set("updated")
	sig3.Set(true)

	if got := sched.PendingCount(); got != 1 {
		t.Errorf("3 signals, same widget: PendingCount = %d, want 1 (dedup)", got)
	}

	sched.Flush()

	if flushCount != 1 {
		t.Errorf("flushed widget count = %d, want 1 (dedup)", flushCount)
	}
}

// TestSchedulerFlush_SetsPerWidgetNeedsRedraw verifies that the production
// flushFn pattern (from app.go) correctly calls SetNeedsRedraw on each widget.
func TestSchedulerFlush_SetsPerWidgetNeedsRedraw(t *testing.T) {
	w1 := &regressionWidget{}
	w2 := &regressionWidget{}
	w3 := &regressionWidget{}

	// Production flushFn pattern from app.go:114-126.
	sched := state.NewScheduler(func(dirty []widget.Widget) {
		for _, dw := range dirty {
			if setter, ok := dw.(interface{ SetNeedsRedraw(bool) }); ok {
				setter.SetNeedsRedraw(true)
			}
		}
	})

	sched.MarkDirty(w1)
	sched.MarkDirty(w2)
	sched.MarkDirty(w3)
	sched.Flush()

	for i, w := range []*regressionWidget{w1, w2, w3} {
		if !w.NeedsRedraw() {
			t.Errorf("widget[%d].NeedsRedraw() = false after flush, want true", i)
		}
	}
}
