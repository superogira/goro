package slider

// Orientation controls the direction of the slider track.
type Orientation uint8

// Orientation constants.
const (
	// Horizontal renders the slider with the thumb moving left to right.
	// This is the default orientation.
	Horizontal Orientation = iota

	// Vertical renders the slider with the thumb moving bottom to top.
	Vertical
)

// String returns a human-readable name for the orientation.
func (o Orientation) String() string {
	switch o {
	case Horizontal:
		return orientationHorizontal
	case Vertical:
		return orientationVertical
	default:
		return orientationUnknown
	}
}

// String constants for Orientation.String to satisfy goconst.
const (
	orientationHorizontal = "Horizontal"
	orientationVertical   = "Vertical"
	orientationUnknown    = "Unknown"
)

// Mark represents a labeled position on the slider track.
// Marks are purely visual annotations; they do not constrain the slider value.
type Mark struct {
	// Value is the position on the slider where the mark appears.
	Value float32

	// Label is the optional text label displayed near the mark.
	// If empty, only a tick mark is drawn.
	Label string
}
