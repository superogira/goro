package widget

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
)

// lifecycleWidget is a mock widget that implements Lifecycle for testing.
type lifecycleWidget struct {
	WidgetBase
	mountCalled   int
	unmountCalled int
	lastCtx       Context
}

func (w *lifecycleWidget) Layout(_ Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(100, 50))
}

func (w *lifecycleWidget) Draw(_ Context, _ Canvas) {}

func (w *lifecycleWidget) Event(_ Context, _ event.Event) bool { return false }

func (w *lifecycleWidget) Mount(ctx Context) {
	w.mountCalled++
	w.lastCtx = ctx
}

func (w *lifecycleWidget) Unmount() {
	w.unmountCalled++
}

func newLifecycleWidget() *lifecycleWidget {
	w := &lifecycleWidget{}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

// Compile-time check.
var _ Lifecycle = (*lifecycleWidget)(nil)

func TestMountTree(t *testing.T) {
	ctx := NewContext()

	parent := newLifecycleWidget()
	child1 := newLifecycleWidget()
	child2 := newLifecycleWidget()
	parent.AddChild(child1)
	parent.AddChild(child2)

	MountTree(parent, ctx)

	if parent.mountCalled != 1 {
		t.Errorf("parent.mountCalled = %d, want 1", parent.mountCalled)
	}
	if child1.mountCalled != 1 {
		t.Errorf("child1.mountCalled = %d, want 1", child1.mountCalled)
	}
	if child2.mountCalled != 1 {
		t.Errorf("child2.mountCalled = %d, want 1", child2.mountCalled)
	}
	if !parent.IsMounted() {
		t.Error("parent should be mounted")
	}
	if !child1.IsMounted() {
		t.Error("child1 should be mounted")
	}
}

func TestUnmountTree(t *testing.T) {
	ctx := NewContext()

	parent := newLifecycleWidget()
	child := newLifecycleWidget()
	parent.AddChild(child)

	// Mount first.
	MountTree(parent, ctx)

	// Add a mock binding to verify cleanup.
	unbindCalled := false
	child.AddBinding(&mockUnbinder{fn: func() { unbindCalled = true }})

	UnmountTree(parent)

	if parent.unmountCalled != 1 {
		t.Errorf("parent.unmountCalled = %d, want 1", parent.unmountCalled)
	}
	if child.unmountCalled != 1 {
		t.Errorf("child.unmountCalled = %d, want 1", child.unmountCalled)
	}
	if parent.IsMounted() {
		t.Error("parent should not be mounted after unmount")
	}
	if child.IsMounted() {
		t.Error("child should not be mounted after unmount")
	}
	if !unbindCalled {
		t.Error("binding should have been cleaned up on unmount")
	}
}

func TestMountTreeSkipsMounted(t *testing.T) {
	ctx := NewContext()
	w := newLifecycleWidget()

	MountTree(w, ctx)
	MountTree(w, ctx) // should be skipped

	if w.mountCalled != 1 {
		t.Errorf("mountCalled = %d, want 1 (skip second mount)", w.mountCalled)
	}
}

func TestMountUnmountNilSafe(t *testing.T) {
	ctx := NewContext()

	// Should not panic.
	MountTree(nil, ctx)
	UnmountTree(nil)
}

func TestMountTreeNonLifecycleWidget(t *testing.T) {
	ctx := NewContext()
	w := newMockWidget() // does not implement Lifecycle

	MountTree(w, ctx)

	if !w.IsMounted() {
		t.Error("non-lifecycle widget should still get mounted state set")
	}
}

// mockUnbinder is a test helper implementing Unbinder.
type mockUnbinder struct {
	fn func()
}

func (m *mockUnbinder) Unbind() {
	if m.fn != nil {
		m.fn()
	}
}
