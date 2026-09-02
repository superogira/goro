package listview

// SelectionMode defines how items can be selected in the list.
type SelectionMode uint8

// SelectionMode constants.
const (
	// SelectionNone disables item selection. This is the default.
	SelectionNone SelectionMode = iota

	// SelectionSingle allows at most one item to be selected at a time.
	SelectionSingle
)

// String returns a human-readable name for the selection mode.
func (m SelectionMode) String() string {
	switch m {
	case SelectionNone:
		return "None"
	case SelectionSingle:
		return "Single"
	default:
		return "Unknown"
	}
}

// ItemContext provides contextual information to the item builder callback.
//
// The builder receives an ItemContext for each visible item, allowing it to
// customize the returned widget based on selection, focus, and hover state.
type ItemContext struct {
	// Index is the zero-based item index in the data source.
	Index int

	// Selected is true if this item is the currently selected item.
	Selected bool

	// Focused is true if this item has keyboard focus within the list.
	Focused bool

	// Hovered is true if the mouse cursor is over this item.
	Hovered bool
}
