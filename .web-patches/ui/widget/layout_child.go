package widget

import (
	"fmt"
	"os"

	"github.com/gogpu/ui/geometry"
)

// layoutDebugEnabled turns on the layout-cache verifier (ADR-032 Phase 1c).
// Enable with GOGPU_DEBUG_LAYOUT=1. Tests flip it via [SetLayoutDebug].
var layoutDebugEnabled = os.Getenv("GOGPU_DEBUG_LAYOUT") == "1"

// SetLayoutDebug toggles the layout-cache verifier at runtime and returns the
// previous value. Intended for tests (so CI can exercise the verifier without
// an env var); production code uses GOGPU_DEBUG_LAYOUT.
func SetLayoutDebug(on bool) bool {
	prev := layoutDebugEnabled
	layoutDebugEnabled = on
	return prev
}

// layoutCacheable is implemented by any widget embedding [WidgetBase] (the
// cache lives on WidgetBase). Widgets that don't embed it simply never cache.
type layoutCacheable interface {
	layoutCacheLookup(geometry.Constraints) (geometry.Size, bool)
	layoutCacheStore(geometry.Constraints, geometry.Size)
}

// LayoutChild measures a child widget through the layout cache (ADR-032).
//
// Container widgets should call this instead of child.Layout(ctx, constraints)
// directly, mirroring how [DrawChild] wraps child.Draw. On a cache hit the
// child's Layout is skipped entirely; on a miss the result is cached and the
// child's nearest RepaintBoundary is marked dirty (GAP-1: paint propagation
// happens here, on the miss path, after Layout has run — not in
// MarkNeedsLayout).
//
// It returns the child's size only; positioning (SetBounds) remains the
// parent's responsibility, exactly as today.
//
// A child that does not embed [WidgetBase] (and so cannot cache) is laid out
// directly every time — same behavior as a plain child.Layout call.
func LayoutChild(child Widget, ctx Context, constraints geometry.Constraints) geometry.Size {
	if child == nil {
		return geometry.Size{}
	}

	cacher, ok := child.(layoutCacheable)
	if !ok {
		return child.Layout(ctx, constraints)
	}

	if size, hit := cacher.layoutCacheLookup(constraints); hit {
		if layoutDebugEnabled {
			verifyLayoutCacheHit(child, ctx, constraints, size)
		}
		return size
	}

	// Cache miss: run layout, cache it, and mark the subtree's boundary dirty
	// so it repaints (only changed subtrees miss, so only they repaint).
	size := child.Layout(ctx, constraints)
	cacher.layoutCacheStore(constraints, size)
	if r, ok := child.(interface{ SetNeedsRedraw(bool) }); ok {
		r.SetNeedsRedraw(true)
	}

	if layoutDebugEnabled {
		assertConstraintsSatisfied(child, constraints, size)
	}
	return size
}

// verifyLayoutCacheHit re-runs the child's Layout and asserts the result
// matches the cached value. A mismatch means a layout-affecting change was made
// without a corresponding MarkNeedsLayout() call — the exact bug class the
// cache risks. Flutter performs the equivalent assertion in debug builds.
func verifyLayoutCacheHit(child Widget, ctx Context, c geometry.Constraints, cached geometry.Size) {
	fresh := child.Layout(ctx, c)
	if fresh != cached {
		panic(fmt.Sprintf(
			"layout cache stale for %T: cached %v but recomputed %v for constraints %v — "+
				"a layout-affecting mutation is missing a MarkNeedsLayout() call",
			child, cached, fresh, c))
	}
	assertConstraintsSatisfied(child, c, fresh)
}

// assertConstraintsSatisfied panics if the measured size violates the input
// constraints (Flutter debugAssertDoesMeetConstraints). Debug-only.
func assertConstraintsSatisfied(child Widget, c geometry.Constraints, size geometry.Size) {
	if !c.IsSatisfiedBy(size) {
		panic(fmt.Sprintf(
			"%T returned size %v that does not satisfy constraints %v",
			child, size, c))
	}
}
