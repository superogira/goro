package scrollview

// ScrollDirection controls which axes the scroll view scrolls along.
type ScrollDirection uint8

// ScrollDirection constants.
const (
	// Vertical enables vertical scrolling only. This is the default.
	Vertical ScrollDirection = iota

	// Horizontal enables horizontal scrolling only.
	Horizontal

	// Both enables scrolling on both axes.
	Both
)

// String returns a human-readable name for the scroll direction.
func (d ScrollDirection) String() string {
	switch d {
	case Vertical:
		return directionVertical
	case Horizontal:
		return directionHorizontal
	case Both:
		return directionBoth
	default:
		return directionUnknown
	}
}

// String constants for ScrollDirection.String to satisfy goconst.
const (
	directionVertical   = "Vertical"
	directionHorizontal = "Horizontal"
	directionBoth       = "Both"
	directionUnknown    = "Unknown"
)

// ScrollbarVisibility controls when scrollbars are displayed.
type ScrollbarVisibility uint8

// ScrollbarVisibility constants.
const (
	// ScrollbarAuto shows scrollbars only when content overflows. This is the default.
	ScrollbarAuto ScrollbarVisibility = iota

	// ScrollbarAlways shows scrollbars regardless of content size.
	ScrollbarAlways

	// ScrollbarNever hides scrollbars completely.
	ScrollbarNever
)

// String returns a human-readable name for the scrollbar visibility mode.
func (v ScrollbarVisibility) String() string {
	switch v {
	case ScrollbarAuto:
		return visibilityAuto
	case ScrollbarAlways:
		return visibilityAlways
	case ScrollbarNever:
		return visibilityNever
	default:
		return visibilityUnknown
	}
}

// String constants for ScrollbarVisibility.String to satisfy goconst.
const (
	visibilityAuto    = "Auto"
	visibilityAlways  = "Always"
	visibilityNever   = "Never"
	visibilityUnknown = "Unknown"
)
