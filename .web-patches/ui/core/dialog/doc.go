// Package dialog provides a modal dialog widget for confirmations, alerts,
// and custom content.
//
// Construction uses functional options for immutable configuration:
//
//	d := dialog.New(
//	    dialog.Title("Confirm Delete"),
//	    dialog.Actions(
//	        dialog.Action{Label: "Cancel"},
//	        dialog.Action{Label: "Delete", Variant: VariantFilled, Default: true},
//	    ),
//	)
//
// # Show/Close Lifecycle
//
// A dialog is created in a hidden state. Call [Widget.Show] to push it
// as a modal overlay, and [Widget.Close] to remove it:
//
//	d.Show(ctx)  // pushes to overlay stack
//	d.Close(ctx) // removes from overlay stack
//
// # Visual Style
//
// The visual rendering is provided by a [Painter] implementation.
// Each design system (Material 3, Fluent, Cupertino) supplies its own
// painter to render dialogs in the appropriate visual style.
//
// If no painter is set, [DefaultPainter] is used, which draws a minimal
// dialog suitable for testing and prototyping.
//
// # Dismissal
//
// By default, dialogs are dismissible by clicking the backdrop and
// pressing the Escape key. This behavior can be configured:
//
//	d := dialog.New(
//	    dialog.Title("Processing..."),
//	    dialog.DismissibleOpt(false),   // cannot click backdrop to close
//	    dialog.EscapeToCloseOpt(false), // cannot press Escape to close
//	)
//
// # Convenience Constructors
//
// [Alert] creates a simple informational dialog with one button:
//
//	dialog.Alert("Error", "Something went wrong.", func() {})
//
// [Confirm] creates a confirmation dialog with Cancel and OK buttons:
//
//	dialog.Confirm("Delete?", "This cannot be undone.", onCancel, onConfirm)
//
// # Signal Binding
//
// The dialog title can be bound to reactive signals from the [state] package:
//
//	title := state.NewSignal("Loading...")
//	d := dialog.New(dialog.TitleSignal(title))
//
// # Focus
//
// Dialogs implement focus trapping: Tab/Shift+Tab cycles between action
// buttons within the dialog, and focus does not escape to the background.
package dialog
