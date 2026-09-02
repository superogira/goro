package tabview

import "github.com/gogpu/ui/state"

// Option configures a tabview during construction.
type Option func(*config)

// PositionOpt sets the tab bar position (top or bottom).
func PositionOpt(p TabPosition) Option {
	return func(c *config) {
		c.position = p
	}
}

// Closeable enables close buttons on all tabs.
// Individual tabs can override this via [Tab.Closeable].
func Closeable(v bool) Option {
	return func(c *config) {
		c.closeable = v
	}
}

// OnSelect sets the callback invoked when a tab is selected.
// The callback receives the index of the newly selected tab.
func OnSelect(fn func(index int)) Option {
	return func(c *config) {
		c.onSelect = fn
	}
}

// OnClose sets the callback invoked when a tab's close button is clicked.
// The callback receives the index of the tab being closed.
func OnClose(fn func(index int)) Option {
	return func(c *config) {
		c.onClose = fn
	}
}

// SelectedIndex sets the initially selected tab index.
func SelectedIndex(idx int) Option {
	return func(c *config) {
		c.selected = idx
	}
}

// SelectedSignalOpt binds the selected tab index to a writable reactive signal.
// When set, the signal value takes precedence over [SelectedIndex]
// but not over [SelectedReadonlySignalOpt].
func SelectedSignalOpt(sig state.Signal[int]) Option {
	return func(c *config) {
		c.selectedSignal = sig
	}
}

// SelectedReadonlySignalOpt binds the selected tab index to a read-only signal.
// This is useful for computed signals. When set, it takes highest precedence.
func SelectedReadonlySignalOpt(sig state.ReadonlySignal[int]) Option {
	return func(c *config) {
		c.readonlySelectedSignal = sig
	}
}

// PainterOpt sets the painter used to render the tab bar.
// Each design system provides its own painter. If not set,
// [DefaultPainter] is used.
func PainterOpt(p Painter) Option {
	return func(c *config) {
		c.painter = p
	}
}
