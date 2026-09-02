package state

import (
	"sync"

	"github.com/gogpu/ui/widget"
)

// Scheduler collects widgets that need re-rendering and flushes them in
// a single batch. This avoids redundant render passes when multiple
// signals change during the same event cycle.
//
// Scheduler is instance-based (no global state) and safe for concurrent use.
//
// Create a Scheduler with [NewScheduler].
type Scheduler struct {
	mu       sync.Mutex
	pending  map[widget.Widget]struct{}
	flushFn  func([]widget.Widget)
	onDirty  func()
	batching bool
	flushing bool
}

// NewScheduler creates a Scheduler that calls flushFn with the
// deduplicated list of dirty widgets when [Scheduler.Flush] is invoked.
//
// flushFn must not be nil. It receives a slice of unique widgets that
// were marked dirty since the last flush. The order of widgets in the
// slice is not guaranteed.
//
// Example:
//
//	sched := state.NewScheduler(func(dirty []widget.Widget) {
//	    for _, w := range dirty {
//	        w.Layout(ctx, constraints)
//	        w.Draw(ctx, canvas)
//	    }
//	})
func NewScheduler(flushFn func([]widget.Widget)) *Scheduler {
	if flushFn == nil {
		panic("state: NewScheduler flushFn must not be nil")
	}
	return &Scheduler{
		pending: make(map[widget.Widget]struct{}),
		flushFn: flushFn,
	}
}

// SetOnDirty registers a callback that is invoked when the pending set
// transitions from empty to non-empty. This is typically used to wake the
// render loop (e.g., call RequestRedraw) so that a new frame is scheduled.
//
// The callback is called outside the scheduler's lock and must be safe
// for concurrent use. Only one callback can be registered; subsequent
// calls replace the previous one. Pass nil to remove the callback.
func (s *Scheduler) SetOnDirty(fn func()) {
	s.mu.Lock()
	s.onDirty = fn
	s.mu.Unlock()
}

// MarkDirty queues a widget for re-render.
//
// If the same widget is marked dirty multiple times before the next
// [Scheduler.Flush] it is only processed once (deduplication).
//
// When the pending set transitions from empty to non-empty, the onDirty
// callback (if set via [Scheduler.SetOnDirty]) is invoked to wake the
// render loop.
//
// If the scheduler is not currently inside a [Scheduler.Batch] call,
// MarkDirty is a lightweight enqueue operation. The actual flush happens
// when the render loop calls Flush.
func (s *Scheduler) MarkDirty(w widget.Widget) {
	if w == nil {
		return
	}

	s.mu.Lock()
	wasEmpty := len(s.pending) == 0
	s.pending[w] = struct{}{}
	onDirty := s.onDirty
	s.mu.Unlock()

	// Notify on first dirty widget (wake render loop).
	if wasEmpty && onDirty != nil {
		onDirty()
	}
}

// Flush processes all pending dirty widgets by calling the flush function
// provided to [NewScheduler].
//
// After the call the pending set is empty. If there are no pending widgets
// Flush is a no-op. Flush is safe to call from any goroutine.
//
// Widgets added during the flush callback are not included in the current
// flush; they will be picked up by the next call to Flush.
func (s *Scheduler) Flush() {
	s.mu.Lock()
	if len(s.pending) == 0 {
		s.mu.Unlock()
		return
	}

	// Snapshot and clear pending set.
	dirty := make([]widget.Widget, 0, len(s.pending))
	for w := range s.pending {
		dirty = append(dirty, w)
	}
	s.pending = make(map[widget.Widget]struct{})
	s.flushing = true
	s.mu.Unlock()

	s.flushFn(dirty)

	s.mu.Lock()
	s.flushing = false
	s.mu.Unlock()
}

// Batch groups multiple state changes so that no automatic flush happens
// until fn returns. After fn completes the pending widgets are NOT
// automatically flushed; call [Scheduler.Flush] explicitly when the
// render loop is ready.
//
// Batch calls may be nested. The batching flag is reference-counted
// internally via a simple boolean; the outermost Batch call clears it.
//
// Example:
//
//	sched.Batch(func() {
//	    counter.Set(1)
//	    name.Set("Alice")
//	    // Both changes enqueue dirty widgets, but nothing is flushed yet.
//	})
//	sched.Flush() // one flush for both changes
func (s *Scheduler) Batch(fn func()) {
	s.mu.Lock()
	wasBatching := s.batching
	s.batching = true
	s.mu.Unlock()

	fn()

	if !wasBatching {
		s.mu.Lock()
		s.batching = false
		s.mu.Unlock()
	}
}

// PendingCount returns the number of widgets currently awaiting flush.
//
// This is primarily useful for testing and diagnostics.
func (s *Scheduler) PendingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.pending)
}

// IsFlushing reports whether the scheduler is currently executing its
// flush function. This is useful for re-entrancy guards.
func (s *Scheduler) IsFlushing() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flushing
}
