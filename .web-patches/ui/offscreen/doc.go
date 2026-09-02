// Package offscreen renders a [widget.Widget] tree into an [*image.RGBA]
// without a GPU, window, or running application.
//
// This enables headless widget rendering for screenshot testing, multi-process
// compositors (e.g. [github.com/gogpu/compose]), PDF/image export, and CI
// pipelines.
//
// The renderer uses CPU-only rasterization via [github.com/gogpu/gg].
// A Material 3 light theme is applied by default; override with [WithTheme].
//
// # Basic usage
//
//	r := offscreen.NewRenderer(400, 120)
//	r.Render(primitives.Text("Hello, World!").FontSize(24))
//	img := r.Image() // *image.RGBA — ready for png.Encode, testing, compositing
//
// # HiDPI rendering
//
//	r := offscreen.NewRenderer(800, 240, offscreen.WithScale(2.0))
//
// # Custom theme and background
//
//	dark := material3.NewDark(widget.Hex(0x00BFA5))
//	r := offscreen.NewRenderer(400, 120,
//	    offscreen.WithTheme(dark),
//	    offscreen.WithBackground(widget.ColorWhite),
//	)
package offscreen
