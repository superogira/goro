package app

import (
	"time"

	"github.com/gogpu/ui/widget"
)

// FrameStats holds timing information for a single frame.
//
// This is useful for performance monitoring and debugging.
// FrameStats is passed to the [FrameCallback] after each call to
// [App.Frame] or [Window.Frame].
type FrameStats struct {
	// FrameStart is the time when Frame() was called.
	FrameStart time.Time

	// LayoutDuration is how long the layout pass took.
	// Zero if layout was not performed this frame.
	LayoutDuration time.Duration

	// DrawDuration is how long the draw pass took.
	DrawDuration time.Duration

	// TotalDuration is the total time spent in Frame().
	TotalDuration time.Duration

	// LayoutPerformed indicates whether layout was recalculated this frame.
	LayoutPerformed bool

	// DrawSkipped indicates that the draw pass was skipped because no
	// widgets in the tree needed re-rendering. When true, the host
	// application can reuse the previous frame's GPU framebuffer output.
	//
	// This is the primary retained-mode optimization: idle UIs skip
	// rendering entirely, consuming zero CPU for the draw phase.
	DrawSkipped bool

	// DrawStats provides per-widget statistics from the draw traversal.
	//
	// When DrawSkipped is true, DrawStats is zero-valued (no traversal
	// was performed). When DrawSkipped is false, DrawStats shows how many
	// widgets were drawn, how many were dirty vs clean, and how many were
	// skipped due to invisibility.
	//
	// This is primarily useful for performance monitoring and for
	// validating that the retained-mode dirty-tracking system is
	// working correctly.
	DrawStats widget.DrawStats
}

// FrameCallback is called after each frame completes with timing statistics.
//
// This is useful for frame time monitoring and adaptive rendering.
// The callback is invoked synchronously at the end of each Frame() call.
type FrameCallback func(stats FrameStats)

// SetFrameCallback sets a callback that is invoked after each Frame call
// with timing statistics.
//
// Set to nil to disable statistics tracking. The callback is called
// synchronously on the same goroutine as Frame().
func (a *App) SetFrameCallback(cb FrameCallback) {
	if a.window == nil {
		return
	}
	a.window.frameCallback = cb
}
