package widget

// DrawStats holds statistics from a draw tree traversal.
//
// These statistics provide observability into the retained-mode rendering
// system. They track how many widgets were drawn versus skipped, enabling
// performance monitoring and validation that the dirty-tracking system
// is working correctly.
//
// In Sub-Phase 1 (frame-level skip), all widgets in a dirty frame are
// drawn because gg clears the pixmap each frame. The stats still track
// which widgets WERE dirty vs clean, providing the foundation for
// Sub-Phase 2 (RepaintBoundary) where clean subtrees will be composited
// from cached textures instead of re-drawn.
type DrawStats struct {
	// TotalWidgets is the total number of widgets visited during traversal.
	TotalWidgets int

	// DrawnWidgets is the number of widgets that had their Draw called.
	DrawnWidgets int

	// SkippedWidgets is the number of widgets skipped (invisible or nil).
	SkippedWidgets int

	// DirtyWidgets is the number of widgets that had their needsRedraw flag set.
	// In Sub-Phase 1 this equals DrawnWidgets for visible widgets.
	// In Sub-Phase 2 with pixel caching, clean widgets will be composited
	// from cache instead of re-drawn, so DirtyWidgets < DrawnWidgets.
	DirtyWidgets int

	// CleanWidgets is the number of visible widgets that did NOT have
	// their needsRedraw flag set. These are candidates for pixel caching
	// in Sub-Phase 2.
	CleanWidgets int

	// CachedWidgets is the number of RepaintBoundary widgets that served
	// their content from a cached pixmap instead of re-rendering the
	// child subtree. This is the primary metric for Sub-Phase 2 pixel
	// caching effectiveness.
	CachedWidgets int
}

// DrawStatsProvider is an optional interface implemented by Context
// implementations that support draw statistics collection.
//
// During a draw pass, widgets like RepaintBoundary type-assert the Context
// to DrawStatsProvider to record cache hits in the frame's DrawStats.
// This uses the established "interface extension via type assertion" pattern
// (same as Focusable, redrawChecker) to avoid adding methods to the
// Context interface (which would be a breaking change).
//
// Example usage in a widget's Draw method:
//
//	if provider, ok := ctx.(widget.DrawStatsProvider); ok {
//	    if stats := provider.DrawStats(); stats != nil {
//	        stats.CachedWidgets++
//	    }
//	}
type DrawStatsProvider interface {
	DrawStats() *DrawStats
}

// DrawTree performs a draw traversal of the widget tree rooted at w,
// collecting statistics about dirty/clean widget state.
//
// All visible widgets are drawn regardless of their dirty state because
// the current rendering backend (gg) clears the pixmap each frame.
// The statistics track which widgets were dirty vs clean, providing
// observability and the foundation for future pixel-caching optimizations.
//
// DrawTree clears the needsRedraw flag on each widget after drawing it,
// so subsequent frames will see the tree as clean unless new signal
// changes mark widgets dirty again.
//
// If w is nil, DrawTree returns zero stats and does nothing.
func DrawTree(w Widget, ctx Context, canvas Canvas) DrawStats {
	var stats DrawStats

	// Make stats accessible to widgets (e.g., RepaintBoundary) via type
	// assertion on the context during the draw pass.
	type drawStatsSetter interface {
		SetDrawStats(*DrawStats)
	}
	if setter, ok := ctx.(drawStatsSetter); ok {
		setter.SetDrawStats(&stats)
		defer setter.SetDrawStats(nil)
	}

	drawTreeRecursive(w, ctx, canvas, &stats)
	return stats
}

// DrawChild draws a child widget with RepaintBoundary support.
//
// Container widgets (BoxWidget, VBox, ListView) should call this instead
// of child.Draw() directly. If the child has IsRepaintBoundary=true
// (ADR-024 WidgetBase property), drawing is routed through scene caching
// via drawBoundaryWidget. Otherwise, child.Draw() is called directly.
//
// This is the Flutter PaintingContext.paintChild pattern: the parent
// checks child.isRepaintBoundary before painting.
func DrawChild(child Widget, ctx Context, canvas Canvas) {
	if child == nil {
		return
	}

	type boundaryChecker interface {
		IsRepaintBoundary() bool
	}
	if bc, ok := child.(boundaryChecker); ok && bc.IsRepaintBoundary() {
		// Flutter paintChild: skip child boundaries during parent recording.
		// Each child boundary has own scene + offscreen texture (depth > 1).
		// Compositor blits all textures. StampScreenOrigin already called
		// by container before DrawChild for hitTest correctness.
		if br, ok2 := canvas.(BoundaryRecorder); ok2 && br.IsBoundaryRecording() {
			StampScreenOrigin(child, canvas)
			stampCompositorClip(child, canvas)
			return
		}
		drawBoundaryWidget(child, ctx, canvas, nil)
		return
	}

	child.Draw(ctx, canvas)
}

// drawTreeRecursive draws the root widget and collects dirty/clean statistics.
//
// It does NOT recurse into children because Widget.Draw() is responsible for
// drawing its own children (e.g., BoxWidget.Draw draws all children internally).
// If we recursed, children would be drawn twice. Statistics for the full tree
// should be collected separately via [CollectDrawStats].
//
// ADR-024: If the widget has isRepaintBoundary == true (WidgetBase property),
// the draw is handled by drawBoundaryWidget which implements scene caching.
func drawTreeRecursive(w Widget, ctx Context, canvas Canvas, stats *DrawStats) {
	if w == nil {
		return
	}

	stats.TotalWidgets++

	// Check dirty state for statistics.
	type redrawChecker interface {
		NeedsRedraw() bool
	}
	if rc, ok := w.(redrawChecker); ok {
		if rc.NeedsRedraw() {
			stats.DirtyWidgets++
		} else {
			stats.CleanWidgets++
		}
	} else {
		// Widget without NeedsRedraw is treated as always dirty.
		stats.DirtyWidgets++
	}

	// Stamp screen origin on the root widget before drawing.
	// Container widgets stamp their children in their own Draw methods.
	StampScreenOrigin(w, canvas)

	// ADR-024: Check if this widget is a WidgetBase-based repaint boundary.
	// If so, route through drawBoundaryWidget for scene caching.
	type boundaryChecker interface {
		IsRepaintBoundary() bool
	}
	if bc, ok := w.(boundaryChecker); ok && bc.IsRepaintBoundary() {
		drawBoundaryWidget(w, ctx, canvas, stats)
		stats.DrawnWidgets++
		return
	}

	// Normal draw path: the widget's own Draw method handles visibility
	// checks and child drawing.
	w.Draw(ctx, canvas)
	stats.DrawnWidgets++
}

// CollectDrawStats walks the widget tree and counts dirty/clean widgets
// WITHOUT drawing anything. This is useful for diagnostics and testing.
//
// Unlike [DrawTree], this function does not call Draw and does not clear
// any redraw flags.
func CollectDrawStats(w Widget) DrawStats {
	var stats DrawStats
	collectStatsRecursive(w, &stats)
	return stats
}

// collectStatsRecursive walks the tree collecting dirty/clean counts.
func collectStatsRecursive(w Widget, stats *DrawStats) {
	if w == nil {
		return
	}

	stats.TotalWidgets++

	// Check visibility.
	type visibilityChecker interface {
		IsVisible() bool
	}
	if vc, ok := w.(visibilityChecker); ok && !vc.IsVisible() {
		stats.SkippedWidgets++
		return
	}

	// Check dirty state.
	type redrawChecker interface {
		NeedsRedraw() bool
	}
	if rc, ok := w.(redrawChecker); ok {
		if rc.NeedsRedraw() {
			stats.DirtyWidgets++
		} else {
			stats.CleanWidgets++
		}
	} else {
		stats.DirtyWidgets++
	}

	// Recurse into children for full tree stats.
	for _, child := range w.Children() {
		collectStatsRecursive(child, stats)
	}
}
