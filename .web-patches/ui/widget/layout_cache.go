package widget

import "github.com/gogpu/ui/geometry"

// Layout caching (ADR-032 Phase 1).
//
// These methods implement the layout-side analog of the RepaintBoundary scene
// cache (see boundary.go). A widget's measured size is cached keyed by the
// input constraints; [LayoutChild] reuses it when a parent re-measures a child
// with the same constraints, and [WidgetBase.MarkNeedsLayout] invalidates it
// when a layout-affecting property changes.
//
// Nothing in this package calls [LayoutChild] or [MarkNeedsLayout] yet — the
// mechanism is wired into containers and widgets in a follow-up. On its own it
// is inert and cannot change behavior.

// MarkNeedsLayout marks this widget's cached layout result as stale and
// propagates the invalidation up the parent chain, because an ancestor's size
// generally depends on this subtree's size. When the invalidation reaches the
// root, the Window is notified via the callback registered with
// [WidgetBase.SetOnLayoutDirty] so it schedules a layout pass.
//
// Idempotency (Flutter object.dart `if (_needsLayout) return`, Android, Yoga):
// the widget's own valid cache acts as the "clean" marker. If this widget has
// no valid cache it is already pending relayout, so the call is a no-op; the
// upward walk likewise stops at the first ancestor that is already invalid (its
// own ancestors were invalidated by an earlier call). A batch of sibling
// invalidations therefore costs one full walk plus O(1) per subsequent sibling,
// and the root callback fires exactly once.
//
// Per ADR-032 GAP-1, this marks paint-dirty on THIS widget only. The upward
// walk clears ancestors' layout caches but does NOT mark them paint-dirty —
// paint propagation to the nearest RepaintBoundary happens in [LayoutChild] on
// the cache-miss path, after a widget's Layout actually runs (Flutter:
// markNeedsLayout does not call markNeedsPaint).
//
// Widgets call this when a layout-affecting property changes (text/content,
// font size, padding, child set, explicit size), as distinct from
// [WidgetBase.SetNeedsRedraw] which is paint-only.
//
// Note: because a valid cache is the "clean" marker, MarkNeedsLayout on a
// never-laid-out widget is a no-op. That is safe: the first layout is always
// forced by the Window (SetRoot/resize set needsLayout directly), after which
// every widget has a cache.
func (w *WidgetBase) MarkNeedsLayout() {
	w.mu.Lock()
	if !w.layoutCacheValid {
		w.mu.Unlock()
		return // already pending relayout — idempotent
	}
	w.layoutCacheValid = false
	w.lastConstraints = geometry.Constraints{}
	w.lastSize = geometry.Size{}
	w.needsRedraw = true // GAP-1: self-only paint dirty
	parent := w.parent
	selfCb := w.onLayoutDirty
	w.mu.Unlock()

	rootCb := invalidateLayoutUpward(parent)
	switch {
	case selfCb != nil: // this widget is itself the root
		selfCb()
	case rootCb != nil:
		rootCb()
	}
}

// invalidateLayoutUpward clears the layout cache on each ancestor (cache only —
// never paint, preserving GAP-1), stopping at the first ancestor whose cache is
// already invalid (a prior invalidation already walked its parents). It returns
// the onLayoutDirty callback of the root if the walk reaches it, so the caller
// can fire it exactly once after propagation completes.
func invalidateLayoutUpward(w Widget) func() {
	type layoutNode interface {
		IsLayoutCacheValid() bool
		InvalidateLayoutCache()
		layoutDirtyCallback() func()
	}
	var rootCb func()
	for w != nil {
		if n, ok := w.(layoutNode); ok {
			if !n.IsLayoutCacheValid() {
				return rootCb // already invalidated up to the root
			}
			n.InvalidateLayoutCache()
			if cb := n.layoutDirtyCallback(); cb != nil {
				rootCb = cb
			}
		}
		pg, ok := w.(interface{ Parent() Widget })
		if !ok {
			return rootCb
		}
		w = pg.Parent()
	}
	return rootCb
}

// InvalidateLayoutCache clears this node's cached layout result. It is a pure
// cache-clearing operation with no side effects beyond zeroing the fields — it
// does not propagate to ancestors, mark paint-dirty, or fire onLayoutDirty (use
// [WidgetBase.MarkNeedsLayout] for the full invalidation flow).
//
// The whole cache is zeroed, not just the validity flag (Android
// requestLayout/mMeasureCache.clear pattern), so a missed constraint
// comparison can never produce a stale hit.
func (w *WidgetBase) InvalidateLayoutCache() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.layoutCacheValid = false
	w.lastConstraints = geometry.Constraints{}
	w.lastSize = geometry.Size{}
}

// SetOnLayoutDirty sets the callback invoked once when a layout invalidation
// reaches this node. The Window wires this on the root widget in SetRoot so a
// descendant's MarkNeedsLayout schedules a layout pass — the layout-side analog
// of [WidgetBase.SetOnBoundaryDirty].
func (w *WidgetBase) SetOnLayoutDirty(fn func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onLayoutDirty = fn
}

// layoutDirtyCallback returns the registered onLayoutDirty callback (nil on
// non-root nodes). Used by the upward walk to fire the root notification once.
func (w *WidgetBase) layoutDirtyCallback() func() {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.onLayoutDirty
}

// IsLayoutCacheValid reports whether this widget currently has a valid cached
// layout result. Primarily for tests and the debug verifier.
func (w *WidgetBase) IsLayoutCacheValid() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.layoutCacheValid
}

// layoutCacheLookup returns the cached size if the cache is valid and was
// computed for exactly these constraints. Constraints is a value of four
// float32 fields, so equality is a direct, allocation-free comparison.
func (w *WidgetBase) layoutCacheLookup(c geometry.Constraints) (geometry.Size, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.layoutCacheValid && w.lastConstraints == c {
		return w.lastSize, true
	}
	return geometry.Size{}, false
}

// layoutCacheStore records the measured size for the given constraints.
func (w *WidgetBase) layoutCacheStore(c geometry.Constraints, size geometry.Size) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastConstraints = c
	w.lastSize = size
	w.layoutCacheValid = true
}
