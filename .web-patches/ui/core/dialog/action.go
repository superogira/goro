package dialog

// Variant constants for action button styling.
// These are uint8 values that match button.Variant values but do not
// import the button package, avoiding tight coupling between core widgets.
const (
	// VariantTextOnly renders an action as text-only (no background).
	VariantTextOnly uint8 = 2

	// VariantFilled renders an action with a solid background.
	VariantFilled uint8 = 0

	// VariantOutlined renders an action with a border.
	VariantOutlined uint8 = 1

	// VariantTonal renders an action with a tinted background.
	VariantTonal uint8 = 3
)

// Action describes a button displayed in the dialog's action area.
//
// Action is a value struct; it is passed by value and does not require
// pointer semantics. The Variant field uses uint8 constants defined in
// this package (e.g. [VariantFilled]) that correspond to button variant
// values without importing the button package.
type Action struct {
	// Label is the text displayed on the action button.
	Label string

	// OnClick is called when the action button is activated.
	// After OnClick returns, the dialog is automatically closed.
	OnClick func()

	// Variant controls the visual style of the button.
	// Use the Variant* constants in this package (e.g. VariantTextOnly).
	// Zero value is VariantFilled.
	Variant uint8

	// Default indicates this action receives initial focus when the
	// dialog is shown. If multiple actions have Default set, the last
	// one wins.
	Default bool
}
