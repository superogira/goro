package dirty

import (
	"fmt"
	"log"
	"os"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

func init() {
	if os.Getenv("GOGPU_DEBUG_COLLECTOR") == "1" {
		collectorDebug = true
	}
}

var collectorDebug bool

// Collector walks the widget tree and collects dirty regions from widgets
// that have NeedsRedraw set. It populates a Tracker with the bounds of
// each dirty widget.
type Collector struct {
	tracker *Tracker
}

// NewCollector creates a new Collector that writes dirty regions to the
// given tracker.
func NewCollector(tracker *Tracker) *Collector {
	return &Collector{tracker: tracker}
}

// Collect walks the widget tree starting from root, adding the bounds of
// any widget with NeedsRedraw() == true to the tracker.
//
// Widgets that do not implement the NeedsRedraw interface (i.e., they do
// not embed [widget.WidgetBase]) are treated as always dirty for safety,
// matching the behavior of [widget.NeedsRedrawInTree].
//
// Invisible widgets and their subtrees are skipped.
func (c *Collector) Collect(root widget.Widget) {
	if root == nil {
		return
	}
	c.collect(root)
}

// collect is the recursive tree walker.
func (c *Collector) collect(w widget.Widget) {
	// Skip invisible widgets entirely.
	if vis, ok := w.(interface{ IsVisible() bool }); ok && !vis.IsVisible() {
		return
	}

	// Viewport containers (ScrollView) act as dirty boundaries.
	// Flutter pattern: Viewport is RepaintBoundary — Collector clips child
	// dirty regions to viewport bounds instead of reporting full content.
	if vc, ok := w.(interface{ IsViewportClip() bool }); ok && vc.IsViewportClip() { //nolint:nestif // viewport dirty collection with leaf-dirty pattern and debug logging
		if c.isWidgetDirty(w) {
			children := w.Children()
			hasDirty := c.hasDirtyChild(children)
			if collectorDebug {
				log.Printf("[COLLECTOR] viewport %T dirty, children=%d, hasDirtyChild=%v",
					w, len(children), hasDirty)
				for i, ch := range children {
					log.Printf("[COLLECTOR]   child[%d] %T dirty=%v", i, ch, c.isWidgetDirty(ch))
				}
			}
			if hasDirty {
				c.collectViewportChildren(w)
			} else {
				c.markWidgetDirty(w)
			}
		} else {
			c.collectViewportChildren(w)
		}
		return
	}

	dirty := c.isWidgetDirty(w)

	if collectorDebug && dirty {
		children := w.Children()
		hasDC := c.hasDirtyChild(children)
		log.Printf("[COLLECT] %T dirty=%v children=%d hasDirtyChild=%v",
			w, dirty, len(children), hasDC)
	}

	// Leaf dirty pattern: if widget dirty AND has dirty children,
	// skip self and report only children (smaller dirty rects).
	children := w.Children()
	if dirty && c.hasDirtyChild(children) {
		for _, child := range children {
			c.collect(child)
		}
		return
	}

	if dirty {
		c.markWidgetDirty(w)
	}

	for _, child := range children {
		c.collect(child)
	}
}

// collectViewportChildren recurses into a viewport container's children,
// collecting dirty regions clipped to the viewport bounds.
func (c *Collector) collectViewportChildren(viewport widget.Widget) {
	type screenBounder interface {
		ScreenBounds() geometry.Rect
	}
	var vpBounds geometry.Rect
	if sb, ok := viewport.(screenBounder); ok {
		vpBounds = sb.ScreenBounds()
	}

	var collectClipped func(w widget.Widget, depth int)
	collectClipped = func(w widget.Widget, depth int) {
		if vis, ok := w.(interface{ IsVisible() bool }); ok && !vis.IsVisible() {
			return
		}
		children := w.Children()
		if c.isWidgetDirty(w) { //nolint:nestif // clipped dirty collection with leaf-dirty pattern and debug logging
			hasDirty := c.hasDirtyChild(children)
			if collectorDebug {
				indent := ""
				for range depth {
					indent += "  "
				}
				log.Printf("[COLLECTOR] %s%T dirty, children=%d, hasDirtyChild=%v",
					indent, w, len(children), hasDirty)
			}
			if hasDirty {
				for _, child := range children {
					collectClipped(child, depth+1)
				}
				return
			}
			if collectorDebug {
				type sb interface{ ScreenBounds() geometry.Rect }
				if s, ok := w.(sb); ok {
					log.Printf("[COLLECTOR] %s→ markClippedDirty bounds=%v", fmt.Sprintf("%*s", depth*2, ""), s.ScreenBounds())
				}
			}
			c.markClippedDirty(w, vpBounds)
		}
		for _, child := range children {
			collectClipped(child, depth+1)
		}
	}

	for _, child := range viewport.Children() {
		collectClipped(child, 0)
	}
}

func intersectRect(a, b geometry.Rect) geometry.Rect {
	r := geometry.Rect{
		Min: geometry.Point{X: max(a.Min.X, b.Min.X), Y: max(a.Min.Y, b.Min.Y)},
		Max: geometry.Point{X: min(a.Max.X, b.Max.X), Y: min(a.Max.Y, b.Max.Y)},
	}
	if r.Min.X >= r.Max.X || r.Min.Y >= r.Max.Y {
		return geometry.Rect{}
	}
	return r
}

// markClippedDirty adds a dirty widget's bounds clipped to the viewport.
func (c *Collector) markClippedDirty(w widget.Widget, vpBounds geometry.Rect) {
	type screenBounder interface {
		ScreenBounds() geometry.Rect
	}
	sb, ok := w.(screenBounder)
	if !ok {
		return
	}
	bounds := sb.ScreenBounds()
	if !vpBounds.IsEmpty() {
		bounds = intersectRect(bounds, vpBounds)
	}
	if !bounds.IsEmpty() {
		c.tracker.MarkDirty(bounds)
	}
}

// hasDirtyChild checks if any immediate child is dirty.
func (c *Collector) hasDirtyChild(children []widget.Widget) bool {
	for _, child := range children {
		if c.isWidgetDirty(child) {
			return true
		}
	}
	return false
}

// isWidgetDirty returns true if the widget needs redrawing.
// Widgets without a NeedsRedraw method (no WidgetBase) are always considered dirty.
func (c *Collector) isWidgetDirty(w widget.Widget) bool {
	type needsRedrawer interface {
		NeedsRedraw() bool
	}
	nr, ok := w.(needsRedrawer)
	if !ok {
		return true // no WidgetBase — assume dirty
	}
	return nr.NeedsRedraw()
}

// markWidgetDirty adds the widget's screen-space bounds to the tracker.
// ScreenBounds (set during the previous Draw pass via StampScreenOrigin)
// gives the global position needed for canvas-level dirty clipping.
// This is safe because layout changes always set needsFullRepaint=true,
// which bypasses dirty-region clipping entirely — so ScreenBounds from
// the previous frame is always valid when this code path executes.
// Follows Qt QWidgetRepaintManager::markDirty pattern: translate
// widget-local rect to top-level window coordinates at collection time.
func (c *Collector) markWidgetDirty(w widget.Widget) {
	if collectorDebug {
		type sb interface{ ScreenBounds() geometry.Rect }
		if s, ok := w.(sb); ok {
			log.Printf("[MARK-DIRTY] %T screenBounds=%v", w, s.ScreenBounds())
		} else {
			log.Printf("[MARK-DIRTY] %T (no ScreenBounds)", w)
		}
	}
	type screenBounder interface {
		ScreenBounds() geometry.Rect
	}
	if sb, ok := w.(screenBounder); ok {
		bounds := sb.ScreenBounds()
		bounds = c.clipToParentViewport(w, bounds)
		c.tracker.MarkDirty(bounds)
		return
	}
	type bounder interface {
		Bounds() geometry.Rect
	}
	if b, ok := w.(bounder); ok {
		c.tracker.MarkDirty(b.Bounds())
	}
}

// clipToParentViewport intersects bounds with parent's screen bounds.
// Scroll content widgets have bounds larger than viewport — clipping
// prevents dirty regions from exceeding the visible area.
func (c *Collector) clipToParentViewport(w widget.Widget, bounds geometry.Rect) geometry.Rect {
	type parentGetter interface {
		Parent() widget.Widget
	}
	pg, ok := w.(parentGetter)
	if !ok {
		return bounds
	}
	parent := pg.Parent()
	if parent == nil {
		return bounds
	}
	type screenBounder interface {
		ScreenBounds() geometry.Rect
	}
	sb, ok := parent.(screenBounder)
	if !ok {
		return bounds
	}
	parentBounds := sb.ScreenBounds()
	if parentBounds.IsEmpty() {
		return bounds
	}
	// Manual intersect (geometry.Rect has no Intersect method).
	clipped := geometry.Rect{
		Min: geometry.Point{
			X: max(bounds.Min.X, parentBounds.Min.X),
			Y: max(bounds.Min.Y, parentBounds.Min.Y),
		},
		Max: geometry.Point{
			X: min(bounds.Max.X, parentBounds.Max.X),
			Y: min(bounds.Max.Y, parentBounds.Max.Y),
		},
	}
	if clipped.Min.X >= clipped.Max.X || clipped.Min.Y >= clipped.Max.Y {
		return geometry.Rect{}
	}
	return clipped
}
