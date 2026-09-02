package widget

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
)

// redrawWidget is a mock widget that supports redraw tracking.
type redrawWidget struct {
	WidgetBase
}

func newRedrawWidget() *redrawWidget {
	w := &redrawWidget{}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *redrawWidget) Layout(_ Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(100, 50))
}

func (w *redrawWidget) Draw(_ Context, _ Canvas) {}

func (w *redrawWidget) Event(_ Context, _ event.Event) bool { return false }

var _ Widget = (*redrawWidget)(nil)

// customWidget does NOT embed WidgetBase (no NeedsRedraw method).
type customWidget struct {
	children []Widget
}

func (w *customWidget) Layout(_ Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(50, 50))
}

func (w *customWidget) Draw(_ Context, _ Canvas) {}

func (w *customWidget) Event(_ Context, _ event.Event) bool { return false }

func (w *customWidget) Children() []Widget { return w.children }

var _ Widget = (*customWidget)(nil)

func TestNeedsRedraw_DefaultState(t *testing.T) {
	w := NewWidgetBase()
	if !w.NeedsRedraw() {
		t.Error("new WidgetBase should need redraw by default")
	}
}

func TestNeedsRedraw_SetAndClear(t *testing.T) {
	w := NewWidgetBase()

	w.ClearRedraw()
	if w.NeedsRedraw() {
		t.Error("should not need redraw after ClearRedraw")
	}

	w.SetNeedsRedraw(true)
	if !w.NeedsRedraw() {
		t.Error("should need redraw after SetNeedsRedraw(true)")
	}

	w.SetNeedsRedraw(false)
	if w.NeedsRedraw() {
		t.Error("should not need redraw after SetNeedsRedraw(false)")
	}
}

func TestNeedsRedrawInTree_NilRoot(t *testing.T) {
	if NeedsRedrawInTree(nil) {
		t.Error("nil root should not need redraw")
	}
}

func TestNeedsRedrawInTree_SingleDirtyWidget(t *testing.T) {
	w := newRedrawWidget()
	w.SetNeedsRedraw(true)

	if !NeedsRedrawInTree(w) {
		t.Error("dirty widget should need redraw in tree")
	}
}

func TestNeedsRedrawInTree_SingleCleanWidget(t *testing.T) {
	w := newRedrawWidget()
	w.ClearRedraw()

	if NeedsRedrawInTree(w) {
		t.Error("clean widget should not need redraw in tree")
	}
}

func TestNeedsRedrawInTree_DirtyChild(t *testing.T) {
	parent := newRedrawWidget()
	parent.ClearRedraw()

	child := newRedrawWidget()
	child.SetNeedsRedraw(true)
	parent.AddChild(child)

	if !NeedsRedrawInTree(parent) {
		t.Error("tree with dirty child should need redraw")
	}
}

func TestNeedsRedrawInTree_AllClean(t *testing.T) {
	parent := newRedrawWidget()
	parent.ClearRedraw()

	child1 := newRedrawWidget()
	child1.ClearRedraw()
	child2 := newRedrawWidget()
	child2.ClearRedraw()
	parent.AddChild(child1)
	parent.AddChild(child2)

	if NeedsRedrawInTree(parent) {
		t.Error("tree with all clean widgets should not need redraw")
	}
}

func TestNeedsRedrawInTree_DeeplyNestedDirty(t *testing.T) {
	root := newRedrawWidget()
	root.ClearRedraw()

	mid := newRedrawWidget()
	mid.ClearRedraw()
	root.AddChild(mid)

	leaf := newRedrawWidget()
	leaf.SetNeedsRedraw(true)
	mid.AddChild(leaf)

	if !NeedsRedrawInTree(root) {
		t.Error("tree with deeply nested dirty widget should need redraw")
	}
}

func TestNeedsRedrawInTree_CustomWidgetWithoutBase(t *testing.T) {
	// Custom widget that does not embed WidgetBase — should be treated
	// as always needing redraw for safety.
	w := &customWidget{}

	if !NeedsRedrawInTree(w) {
		t.Error("widget without NeedsRedraw should be treated as dirty")
	}
}

func TestClearRedrawInTree_NilRoot(t *testing.T) {
	// Should not panic.
	ClearRedrawInTree(nil)
}

func TestClearRedrawInTree_ClearsAll(t *testing.T) {
	root := newRedrawWidget()
	root.SetNeedsRedraw(true)

	child := newRedrawWidget()
	child.SetNeedsRedraw(true)
	root.AddChild(child)

	grandchild := newRedrawWidget()
	grandchild.SetNeedsRedraw(true)
	child.AddChild(grandchild)

	ClearRedrawInTree(root)

	if root.NeedsRedraw() {
		t.Error("root should be clean after ClearRedrawInTree")
	}
	if child.NeedsRedraw() {
		t.Error("child should be clean after ClearRedrawInTree")
	}
	if grandchild.NeedsRedraw() {
		t.Error("grandchild should be clean after ClearRedrawInTree")
	}
}

func TestMarkRedrawInTree_NilRoot(t *testing.T) {
	// Should not panic.
	MarkRedrawInTree(nil)
}

func TestMarkRedrawInTree_MarksAll(t *testing.T) {
	root := newRedrawWidget()
	root.ClearRedraw()

	child := newRedrawWidget()
	child.ClearRedraw()
	root.AddChild(child)

	grandchild := newRedrawWidget()
	grandchild.ClearRedraw()
	child.AddChild(grandchild)

	MarkRedrawInTree(root)

	if !root.NeedsRedraw() {
		t.Error("root should be dirty after MarkRedrawInTree")
	}
	if !child.NeedsRedraw() {
		t.Error("child should be dirty after MarkRedrawInTree")
	}
	if !grandchild.NeedsRedraw() {
		t.Error("grandchild should be dirty after MarkRedrawInTree")
	}
}

func TestClearRedrawInTree_CustomWidget(t *testing.T) {
	// Custom widget without ClearRedraw should not panic.
	w := &customWidget{}
	ClearRedrawInTree(w)
}

func TestMarkRedrawInTree_CustomWidget(t *testing.T) {
	// Custom widget without SetNeedsRedraw should not panic.
	w := &customWidget{}
	MarkRedrawInTree(w)
}

func TestNeedsRedrawInTree_CustomWidgetWithCleanChildren(t *testing.T) {
	// Custom widget is considered dirty, so even with clean children
	// the tree should need redraw.
	child := newRedrawWidget()
	child.ClearRedraw()

	w := &customWidget{children: []Widget{child}}
	if !NeedsRedrawInTree(w) {
		t.Error("custom widget without NeedsRedraw should always be dirty")
	}
}
