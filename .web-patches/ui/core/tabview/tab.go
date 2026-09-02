package tabview

import "github.com/gogpu/ui/widget"

// Tab represents a single tab with a label, content widget, and options.
type Tab struct {
	// Label is the display text shown in the tab bar.
	Label string

	// Content is the widget displayed when this tab is selected.
	// If nil, the tab area is empty when selected.
	Content widget.Widget

	// Closeable enables the close button for this specific tab.
	// This overrides the widget-level closeable setting when true.
	Closeable bool

	// Disabled prevents the tab from being selected.
	Disabled bool
}

// TabPosition controls where the tab bar is rendered relative to the content.
type TabPosition uint8

// Tab position constants.
const (
	// Top places the tab bar above the content area.
	Top TabPosition = iota

	// Bottom places the tab bar below the content area.
	Bottom
)

// String returns a human-readable name for the tab position.
func (p TabPosition) String() string {
	switch p {
	case Top:
		return positionTop
	case Bottom:
		return positionBottom
	default:
		return positionUnknown
	}
}

// String constants for TabPosition.String to satisfy goconst.
const (
	positionTop     = "Top"
	positionBottom  = "Bottom"
	positionUnknown = "Unknown"
)

// Tab bar layout constants.
const (
	// tabBarHeight is the fixed height of the tab bar in logical pixels.
	tabBarHeight float32 = 48

	// tabPaddingX is the horizontal padding inside each tab.
	tabPaddingX float32 = 16

	// tabFontSize is the default font size for tab labels.
	tabFontSize float32 = 14

	// closeButtonSize is the size of the close button area.
	closeButtonSize float32 = 16

	// closeButtonPadding is the padding between the label and close button.
	closeButtonPadding float32 = 8

	// indicatorHeight is the height of the selected tab indicator.
	indicatorHeight float32 = 3
)
