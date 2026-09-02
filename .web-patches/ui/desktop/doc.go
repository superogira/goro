// Package desktop provides a managed render loop for gogpu/ui applications
// running inside a [github.com/gogpu/gogpu] desktop window.
//
// Without this package, host applications must manually create a ggcanvas,
// handle resize, draw the background, call [app.App.Frame], translate the
// widget canvas, and present to the GPU surface. This results in 50-60 lines
// of render boilerplate in every example and application.
//
// desktop.Run encapsulates the entire rendering pipeline into a single call:
//
//	gogpuApp := gogpu.NewApp(gogpu.DefaultConfig().WithTitle("My App"))
//	uiApp := app.New(
//	    app.WithWindowProvider(gogpuApp),
//	    app.WithPlatformProvider(gogpuApp),
//	    app.WithEventSource(gogpuApp.EventSource()),
//	    app.WithRenderMode(app.RenderModeFrameworkManaged),
//	)
//	uiApp.SetRoot(buildUI())
//	if err := desktop.Run(gogpuApp, uiApp); err != nil {
//	    log.Fatal(err)
//	}
//
// The framework controls the entire rendering pipeline: ggcanvas lifecycle,
// background clear (in FrameworkManaged mode), widget drawing, and GPU
// surface presentation.
//
// # Render Modes
//
// In [app.RenderModeHostManaged] (default), desktop.Run draws a light-gray
// background before calling DrawTo, matching the behavior of existing examples.
//
// In [app.RenderModeFrameworkManaged], desktop.Run does NOT draw a background.
// The framework's DrawTo handles background clearing via the theme color.
// This mode enables frame skip (zero CPU when idle) and future incremental
// dirty-region rendering.
//
// # GPU Acceleration
//
// If [github.com/gogpu/gg/gpu] is imported via blank import, GPU SDF
// acceleration is active. desktop.Run configures the surface target for
// zero-copy GPU-direct rendering when the backend supports it, and falls
// back to the universal CPU-readback path otherwise.
//
// # Resource Cleanup
//
// desktop.Run registers an OnClose callback that releases the GPU accelerator
// and ggcanvas resources. The ggcanvas is also auto-tracked by gogpu's
// ResourceTracker for additional safety.
package desktop
