package state_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
)

// mockContext is a minimal Context implementation for testing bindings.
type mockContext struct {
	widget.Context
	invalidated atomic.Int32
}

func (m *mockContext) Invalidate() {
	m.invalidated.Add(1)
}

// mockWidget is a minimal Widget implementation for testing scheduler bindings.
// It implements all methods required by the widget.Widget interface.
type mockWidget struct {
	widget.WidgetBase
	name string
}

func (m *mockWidget) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(0, 0))
}

func (m *mockWidget) Draw(_ widget.Context, _ widget.Canvas) {}

func (m *mockWidget) Event(_ widget.Context, _ event.Event) bool { return false }

func TestBindInvalidatesOnChange(t *testing.T) {
	sig := state.NewSignal(0)
	ctx := &mockContext{}

	binding := state.Bind(sig.AsReadonly(), ctx)
	defer binding.Unbind()

	sig.Set(1)
	sig.Set(2)
	sig.Set(3)

	if got := ctx.invalidated.Load(); got != 3 {
		t.Errorf("invalidated count = %d, want 3", got)
	}
}

func TestBindUnbindStopsInvalidation(t *testing.T) {
	sig := state.NewSignal(0)
	ctx := &mockContext{}

	binding := state.Bind(sig.AsReadonly(), ctx)
	sig.Set(1)
	binding.Unbind()
	sig.Set(2)
	sig.Set(3)

	if got := ctx.invalidated.Load(); got != 1 {
		t.Errorf("invalidated count = %d, want 1 (after unbind)", got)
	}
}

func TestBindUnbindIdempotent(t *testing.T) {
	sig := state.NewSignal(0)
	ctx := &mockContext{}

	binding := state.Bind(sig.AsReadonly(), ctx)
	binding.Unbind()
	binding.Unbind() // second call should be safe
	binding.Unbind() // third call should be safe

	// No panic is the success criterion.
}

func TestBindIsActive(t *testing.T) {
	sig := state.NewSignal(0)
	ctx := &mockContext{}

	binding := state.Bind(sig.AsReadonly(), ctx)
	if !binding.IsActive() {
		t.Error("new binding should be active")
	}

	binding.Unbind()
	if binding.IsActive() {
		t.Error("unbound binding should not be active")
	}
}

func TestBindToScheduler(t *testing.T) {
	sig := state.NewSignal(0)
	w := &mockWidget{name: "test"}
	var flushed []widget.Widget
	sched := state.NewScheduler(func(dirty []widget.Widget) {
		flushed = append(flushed, dirty...)
	})

	binding := state.BindToScheduler(sig.AsReadonly(), w, sched)
	defer binding.Unbind()

	sig.Set(1)
	sig.Set(2)

	// Two changes, but widget should be deduplicated.
	if got := sched.PendingCount(); got != 1 {
		t.Errorf("pending count = %d, want 1 (dedup)", got)
	}

	sched.Flush()
	if len(flushed) != 1 {
		t.Fatalf("flushed count = %d, want 1", len(flushed))
	}
}

func TestBindToSchedulerUnbind(t *testing.T) {
	sig := state.NewSignal(0)
	w := &mockWidget{name: "test"}
	sched := state.NewScheduler(func(_ []widget.Widget) {})

	binding := state.BindToScheduler(sig.AsReadonly(), w, sched)
	sig.Set(1)
	binding.Unbind()
	// Clear pending from previous set.
	sched.Flush()

	sig.Set(2)
	if got := sched.PendingCount(); got != 0 {
		t.Errorf("after unbind, pending count = %d, want 0", got)
	}
}

func TestBindConcurrent(t *testing.T) {
	sig := state.NewSignal(0)
	ctx := &mockContext{}

	binding := state.Bind(sig.AsReadonly(), ctx)
	defer binding.Unbind()

	var wg sync.WaitGroup
	const goroutines = 20
	const iterations = 50

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for j := range iterations {
				sig.Set(j)
			}
		}()
	}
	wg.Wait()

	if got := ctx.invalidated.Load(); got == 0 {
		t.Error("expected at least one invalidation from concurrent sets")
	}
}

func TestBindStringSignal(t *testing.T) {
	sig := state.NewSignal("initial")
	ctx := &mockContext{}

	binding := state.Bind(sig.AsReadonly(), ctx)
	defer binding.Unbind()

	sig.Set("updated")
	if got := ctx.invalidated.Load(); got != 1 {
		t.Errorf("invalidated count = %d, want 1", got)
	}
}

func TestBindToSchedulerFunc(t *testing.T) {
	sig := state.NewSignal(0.5)
	w := &mockWidget{name: "bar"}
	var dirtyCount int
	sched := state.NewScheduler(func(_ []widget.Widget) { dirtyCount++ })

	// Only dirty when value > 0.5.
	binding := state.BindToSchedulerFunc(sig.AsReadonly(), func(v float64) bool {
		return v > 0.5
	}, w, sched)
	defer binding.Unbind()

	sig.Set(0.3) // below threshold, should NOT dirty
	if sched.PendingCount() != 0 {
		t.Errorf("predicate returned false, pending = %d, want 0", sched.PendingCount())
	}

	sig.Set(0.8) // above threshold, SHOULD dirty
	if sched.PendingCount() != 1 {
		t.Errorf("predicate returned true, pending = %d, want 1", sched.PendingCount())
	}
}

func TestBindToSchedulerLayout(t *testing.T) {
	sig := state.NewSignal(0)
	w := &mockWidget{name: "layout-test"}
	var flushed []widget.Widget
	sched := state.NewScheduler(func(dirty []widget.Widget) {
		flushed = append(flushed, dirty...)
	})

	ctx := widget.NewContext()
	widget.LayoutChild(w, ctx, geometry.Loose(geometry.Sz(100, 100)))
	if !w.IsLayoutCacheValid() {
		t.Fatal("cache should be valid after LayoutChild")
	}

	binding := state.BindToSchedulerLayout(sig.AsReadonly(), w, sched)
	defer binding.Unbind()

	sig.Set(1)

	if w.IsLayoutCacheValid() {
		t.Error("BindToSchedulerLayout must invalidate layout cache on signal change")
	}
	if got := sched.PendingCount(); got != 1 {
		t.Errorf("pending count = %d, want 1", got)
	}
}

func TestBindToSchedulerLayout_Unbind(t *testing.T) {
	sig := state.NewSignal(0)
	w := &mockWidget{name: "layout-unbind"}
	sched := state.NewScheduler(func(_ []widget.Widget) {})

	ctx := widget.NewContext()
	widget.LayoutChild(w, ctx, geometry.Loose(geometry.Sz(100, 100)))

	binding := state.BindToSchedulerLayout(sig.AsReadonly(), w, sched)
	sig.Set(1)
	binding.Unbind()
	sched.Flush()

	widget.LayoutChild(w, ctx, geometry.Loose(geometry.Sz(100, 100)))

	sig.Set(2)
	if !w.IsLayoutCacheValid() {
		t.Error("after unbind, layout cache should remain valid")
	}
	if sched.PendingCount() != 0 {
		t.Errorf("after unbind, pending = %d, want 0", sched.PendingCount())
	}
}

func TestBindToSchedulerLayoutFunc(t *testing.T) {
	sig := state.NewSignal(0.5)
	w := &mockWidget{name: "layout-func"}
	sched := state.NewScheduler(func(_ []widget.Widget) {})

	ctx := widget.NewContext()
	widget.LayoutChild(w, ctx, geometry.Loose(geometry.Sz(100, 100)))

	binding := state.BindToSchedulerLayoutFunc(sig.AsReadonly(), func(v float64) bool {
		return v > 0.5
	}, w, sched)
	defer binding.Unbind()

	sig.Set(0.3)
	if !w.IsLayoutCacheValid() {
		t.Error("predicate false: layout cache should remain valid")
	}
	if sched.PendingCount() != 0 {
		t.Errorf("predicate false: pending = %d, want 0", sched.PendingCount())
	}

	sig.Set(0.8)
	if w.IsLayoutCacheValid() {
		t.Error("predicate true: layout cache must be invalidated")
	}
	if sched.PendingCount() != 1 {
		t.Errorf("predicate true: pending = %d, want 1", sched.PendingCount())
	}
}

func TestBindToSchedulerFunc_Unbind(t *testing.T) {
	sig := state.NewSignal(1.0)
	w := &mockWidget{name: "bar"}
	sched := state.NewScheduler(func(_ []widget.Widget) {})

	binding := state.BindToSchedulerFunc(sig.AsReadonly(), func(_ float64) bool {
		return true
	}, w, sched)

	sig.Set(2.0)
	if sched.PendingCount() == 0 {
		t.Error("before unbind, signal change should mark dirty")
	}

	binding.Unbind()
	sched.Flush()

	sig.Set(3.0)
	if sched.PendingCount() != 0 {
		t.Errorf("after unbind, pending = %d, want 0", sched.PendingCount())
	}
}
