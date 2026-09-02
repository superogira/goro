package dialog

// alertOKLabel is the default button label for alert dialogs.
const alertOKLabel = "OK"

// confirmCancelLabel is the default cancel button label for confirm dialogs.
const confirmCancelLabel = "Cancel"

// Alert creates a simple informational dialog with a title, message, and
// a single OK button. The onOK callback is invoked when the OK button is
// clicked or when the dialog is dismissed.
//
// The returned dialog must still be shown with [Widget.Show].
//
// Example:
//
//	d := dialog.Alert("Error", "File not found.", func() { log.Println("dismissed") })
//	d.Show(ctx)
func Alert(title, message string, onOK func()) *Widget {
	return New(
		Title(title),
		Actions(Action{
			Label:   alertOKLabel,
			OnClick: onOK,
			Variant: VariantFilled,
			Default: true,
		}),
		OnClose(onOK),
	)
}

// Confirm creates a confirmation dialog with Cancel and OK buttons.
// The onCancel callback is invoked when Cancel is clicked or the dialog
// is dismissed. The onConfirm callback is invoked when OK is clicked.
//
// The returned dialog must still be shown with [Widget.Show].
//
// Example:
//
//	d := dialog.Confirm("Delete?", "This cannot be undone.",
//	    func() { log.Println("canceled") },
//	    func() { deleteItem() },
//	)
//	d.Show(ctx)
func Confirm(title, message string, onCancel, onConfirm func()) *Widget {
	return New(
		Title(title),
		Actions(
			Action{
				Label:   confirmCancelLabel,
				OnClick: onCancel,
				Variant: VariantTextOnly,
			},
			Action{
				Label:   alertOKLabel,
				OnClick: onConfirm,
				Variant: VariantFilled,
				Default: true,
			},
		),
		OnClose(onCancel),
	)
}
