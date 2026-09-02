// Package tabview provides a tabbed navigation widget.
//
// Construction uses functional options for immutable configuration:
//
//	tv := tabview.New(
//	    []tabview.Tab{
//	        {Label: "Home", Content: homeWidget},
//	        {Label: "Settings", Content: settingsWidget},
//	    },
//	    tabview.PositionOpt(tabview.Top),
//	    tabview.Closeable(true),
//	    tabview.OnSelect(func(idx int) { fmt.Println("selected:", idx) }),
//	)
//
// # Tab Position
//
// Tabs can be placed at the top or bottom of the content area:
//   - [Top] (default) -- tab bar above content
//   - [Bottom] -- tab bar below content
//
// # Keyboard Navigation
//
// When the tab bar is focused, the following keys are supported:
//   - Left/Right arrows -- navigate between tabs (skipping disabled tabs)
//   - Home -- select first enabled tab
//   - End -- select last enabled tab
//
// # Closeable Tabs
//
// When closeable is enabled (globally or per-tab), a close button appears
// on each tab. Clicking it triggers the [OnClose] callback.
//
// # Signal Binding
//
// The selected tab index can be bound to a reactive signal:
//
//	selected := state.NewSignal(0)
//	tv := tabview.New(tabs, tabview.SelectedSignalOpt(selected))
//	selected.Set(2) // switches to third tab
//
// # Visual Style
//
// The visual rendering is provided by a [Painter] implementation.
// Each design system (Material 3, Fluent, Cupertino) supplies its own
// painter to render the tab bar in the appropriate visual style.
//
// If no painter is set, [DefaultPainter] is used.
//
// # Focus
//
// TabView implements [widget.Focusable] and participates in tab navigation.
// A focus ring is drawn around the tab bar when focused.
package tabview
