package offscreen

import (
	"image"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/geometry"
	internalRender "github.com/gogpu/ui/internal/render"
	"github.com/gogpu/ui/render"
	"github.com/gogpu/ui/theme/material3"
	"github.com/gogpu/ui/widget"
)

// m3DefaultSeed is the Material 3 baseline purple used when no theme is provided.
var m3DefaultSeed = widget.Hex(0x6750A4)

// maxFitDimension caps fit-to-content for widgets that expand infinitely.
const maxFitDimension = 4096

// Renderer renders a widget tree into an offscreen image.
//
// Create with [NewRenderer]. Configure with [Option] functions.
// Call [Renderer.Render] to draw a widget, then [Renderer.Image] to
// retrieve the result.
type Renderer struct {
	width      int
	height     int
	scale      float32
	theme      widget.ThemeProvider
	background widget.Color
	img        *image.RGBA
	fitSize    bool
	maxWidth   int
	maxHeight  int
}

// Option configures a [Renderer].
type Option func(*Renderer)

// WithTheme overrides the default Material 3 light theme.
func WithTheme(tp widget.ThemeProvider) Option {
	return func(r *Renderer) {
		r.theme = tp
	}
}

// WithScale sets the display scale factor for HiDPI rendering.
// The default is 1.0. A value of 2.0 renders at Retina density.
func WithScale(s float32) Option {
	return func(r *Renderer) {
		if s > 0 {
			r.scale = s
		}
	}
}

// WithBackground sets the canvas clear color before drawing.
// The default is transparent (zero alpha).
func WithBackground(c widget.Color) Option {
	return func(r *Renderer) {
		r.background = c
	}
}

// WithFitSize enables measure-then-render mode. The renderer measures the
// widget's preferred size before allocating the canvas, so the caller does
// not need to know dimensions upfront. Pass width=0, height=0 to NewRenderer.
//
// This follows the universal enterprise pattern (Flutter IntrinsicWidth,
// Qt6 sizeHint, SwiftUI sizeThatFits, Compose SubcomposeLayout).
//
// Use [WithMaxSize] to constrain expansion for widgets that grow unbounded.
func WithFitSize() Option {
	return func(r *Renderer) {
		r.fitSize = true
	}
}

// WithMaxSize sets maximum dimensions for fit-to-content mode.
// A value of 0 means unlimited in that axis.
// Only effective when [WithFitSize] is also set.
func WithMaxSize(width, height int) Option {
	return func(r *Renderer) {
		r.maxWidth = width
		r.maxHeight = height
	}
}

// NewRenderer creates an offscreen renderer with the given pixel dimensions.
//
// Width and height must be positive; they are clamped to a minimum of 1.
// By default, a Material 3 light theme is applied and the background is
// transparent. Override with [WithTheme], [WithBackground], and [WithScale].
func NewRenderer(width, height int, opts ...Option) *Renderer {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	r := &Renderer{
		width:      width,
		height:     height,
		scale:      1.0,
		theme:      material3.New(m3DefaultSeed),
		background: widget.ColorTransparent,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Render lays out and draws the widget into the offscreen buffer.
//
// In default mode, the widget receives the renderer dimensions as loose
// constraints. In fit-to-content mode ([WithFitSize]), the widget is
// measured first to determine its preferred size, then the canvas is
// allocated at that size. Calling Render again replaces the previous image.
func (r *Renderer) Render(w widget.Widget) {
	ctx := widget.NewContext()
	ctx.SetThemeProvider(r.theme)
	ctx.SetScale(r.scale)

	width, height := r.width, r.height

	if r.fitSize {
		width, height = r.measure(w, ctx)
	}

	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	ctx.SetWindowSize(geometry.Sz(float32(width), float32(height)))

	constraints := geometry.Loose(geometry.Sz(float32(width), float32(height)))
	w.Layout(ctx, constraints)

	dc := gg.NewContext(width, height)
	canvas := render.NewCanvas(dc, width, height)
	canvas.Clear(r.background)

	widget.DrawTree(w, ctx, canvas)

	r.img = internalRender.ToRGBA(dc.Image())
	_ = dc.Close()
}

// measure performs a layout pass with expanded constraints to determine
// the widget's preferred size. Applies maxWidth/maxHeight if set.
func (r *Renderer) measure(w widget.Widget, ctx *widget.ContextImpl) (int, int) {
	maxW := float32(maxFitDimension)
	maxH := float32(maxFitDimension)
	if r.maxWidth > 0 {
		maxW = float32(r.maxWidth)
	}
	if r.maxHeight > 0 {
		maxH = float32(r.maxHeight)
	}

	ctx.SetWindowSize(geometry.Sz(maxW, maxH))
	constraints := geometry.Loose(geometry.Sz(maxW, maxH))
	size := w.Layout(ctx, constraints)

	width := int(size.Width + 0.5)
	height := int(size.Height + 0.5)

	if width > int(maxW) {
		width = int(maxW)
	}
	if height > int(maxH) {
		height = int(maxH)
	}

	return width, height
}

// Image returns the rendered image, or nil if [Render] has not been called.
func (r *Renderer) Image() *image.RGBA {
	return r.img
}
