package app

// RenderMode controls who owns the window background and pixmap lifecycle.
//
// This is the primary configuration point for choosing between host-managed
// and framework-managed rendering. The mode affects whether DrawTo clears
// the canvas, whether frame skip is possible, and whether incremental
// dirty-region rendering is active.
//
// Enterprise GUI frameworks (Qt QBackingStore, Flutter, Win32) formalize
// background ownership through explicit API. RenderMode provides the same
// clarity for gogpu/ui.
type RenderMode int

const (
	// RenderModeHostManaged is the default rendering mode where the host
	// application owns the window background and pixmap lifecycle.
	//
	// In this mode:
	//   - The host draws its own background before calling DrawTo.
	//   - DrawTo does NOT call canvas.Clear (the host already painted the background).
	//   - DrawTo always performs a full widget tree draw via DrawTree.
	//   - DrawTo always returns true (a valid frame is always produced).
	//   - Frame skip is NOT possible (host may have cleared the pixmap).
	//   - dirty.Tracker is still populated (for RepaintBoundary Intersects fast path).
	//
	// This is backward compatible with existing host applications that
	// clear the canvas themselves before calling DrawTo.
	RenderModeHostManaged RenderMode = iota

	// RenderModeFrameworkManaged is the rendering mode where the framework
	// owns the pixmap lifecycle and draws the theme background.
	//
	// In this mode:
	//   - The host does NOT draw background before calling DrawTo.
	//   - DrawTo calls canvas.Clear(themeBackground) on full repaint.
	//   - Frame skip: DrawTo returns false when nothing changed (Level 1, ADR-004).
	//   - Incremental rendering: only dirty regions are redrawn (Level 2, ADR-004).
	//   - Full repaint on: first frame, resize, theme change, SetRoot.
	//
	// This mode enables the full three-level incremental rendering pipeline
	// described in ADR-004, achieving near-zero CPU for idle UIs.
	RenderModeFrameworkManaged
)

// Render mode string constants.
const (
	renderModeHostManagedStr      = "HostManaged"
	renderModeFrameworkManagedStr = "FrameworkManaged"
	renderModeUnknownStr          = "Unknown"
)

// String returns a human-readable name for the render mode.
func (m RenderMode) String() string {
	switch m {
	case RenderModeHostManaged:
		return renderModeHostManagedStr
	case RenderModeFrameworkManaged:
		return renderModeFrameworkManagedStr
	default:
		return renderModeUnknownStr
	}
}
