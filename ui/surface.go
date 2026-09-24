package ui

import (
	"image/color"
	"math"

	uiwidget "github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/render"
)

type surfaceKey struct {
	w        int
	h        int
	top      color.RGBA
	bottom   color.RGBA
	border   color.RGBA
	radius   float32
	gradient bool
}

var surfaceCache = map[surfaceKey]*render.Image{}

const HungerBarLowThreshold = 0.25

var (
	WindowRadius      = float32(6)
	WindowBorderColor = color.RGBA{R: 118, G: 160, B: 206, A: 255}
	PanelAltColor     = color.RGBA{R: 236, G: 244, B: 252, A: 255}
	TextColor         = color.RGBA{R: 38, G: 48, B: 58, A: 255}
	MutedTextColor    = color.RGBA{R: 98, G: 112, B: 126, A: 255}
	TitleTextColor    = color.RGBA{R: 22, G: 54, B: 88, A: 255}
	GoodTextColor     = color.RGBA{R: 34, G: 142, B: 72, A: 255}
	ErrorTextColor    = color.RGBA{R: 204, G: 48, B: 48, A: 255}
	PlayerHPBarColor  = color.RGBA{R: 16, G: 239, B: 33, A: 255}
	PlayerSPBarColor  = color.RGBA{R: 24, G: 99, B: 222, A: 255}
	HungerBarColor    = color.RGBA{R: 255, G: 231, B: 231, A: 255}
	HungerBarLowColor = color.RGBA{R: 255, G: 255, B: 0, A: 255}
)

func HungerBarFillColor(current, maxValue int) color.RGBA {
	ratio := 0.0
	if maxValue > 0 {
		ratio = clampUnit(float64(current) / float64(maxValue))
	}
	if ratio < HungerBarLowThreshold {
		return HungerBarLowColor
	}
	return HungerBarColor
}

func LerpColor(a, b color.RGBA, t float64) color.RGBA {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t + 0.5),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t + 0.5),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t + 0.5),
		A: uint8(float64(a.A) + (float64(b.A)-float64(a.A))*t + 0.5),
	}
}

type imageDrawTarget interface {
	DrawImage(*render.Image, *render.DrawImageOptions)
}

func DrawRoundedSurface(screen imageDrawTarget, x, y, w, h int, bg, border color.RGBA, radius float32) {
	DrawRoundedGradientSurface(screen, x, y, w, h, bg, bg, border, radius)
}

func DrawRoundedGradientSurface(screen imageDrawTarget, x, y, w, h int, top, bottom, border color.RGBA, radius float32) {
	if screen == nil || w <= 0 || h <= 0 {
		return
	}
	img := cachedSurface(w, h, top, bottom, border, radius)
	if img == nil {
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Translate(float64(x), float64(y))
	opts.Filter = render.FilterNearest
	screen.DrawImage(img, &opts)
}

func cachedSurface(w, h int, top, bottom, border color.RGBA, radius float32) *render.Image {
	key := surfaceKey{w: w, h: h, top: top, bottom: bottom, border: border, radius: radius, gradient: top != bottom}
	if img, ok := surfaceCache[key]; ok {
		return img
	}
	img := render.NewImage(w, h)
	drawCachedSurface(img, w, h, top, bottom, border, float64(radius))
	if len(surfaceCache) > 256 {
		surfaceCache = map[surfaceKey]*render.Image{}
	}
	surfaceCache[key] = img
	return img
}

func drawCachedSurface(img *render.Image, w, h int, top, bottom, border color.RGBA, radius float64) {
	if img == nil || w <= 0 || h <= 0 {
		return
	}
	maxRadius := float64(minInt(w, h)) * 0.5
	if radius > maxRadius {
		radius = maxRadius
	}
	const samples = 4
	for py := 0; py < h; py++ {
		t := 0.0
		if h > 1 {
			t = float64(py) / float64(h-1)
		}
		fill := LerpColor(top, bottom, t)
		for px := 0; px < w; px++ {
			var outer, inner float64
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					x := float64(px) + (float64(sx)+0.5)/samples
					y := float64(py) + (float64(sy)+0.5)/samples
					if roundedSampleInside(x, y, float64(w), float64(h), radius, 0) {
						outer++
					}
					if roundedSampleInside(x, y, float64(w), float64(h), radius, 1) {
						inner++
					}
				}
			}
			outer /= samples * samples
			inner /= samples * samples
			if border.A != 0 {
				drawSurfacePixel(img, px, py, border, math.Max(0, outer-inner))
				drawSurfacePixel(img, px, py, fill, inner)
			} else {
				drawSurfacePixel(img, px, py, fill, outer)
			}
		}
	}
}

func roundedSampleInside(x, y, w, h, radius float64, inset float64) bool {
	if x < inset || y < inset || x >= w-inset || y >= h-inset {
		return false
	}
	r := radius - inset
	if r <= 0 {
		return true
	}
	left := inset + r
	right := w - inset - r
	top := inset + r
	bottom := h - inset - r
	cx := clampFloat(x, left, right)
	cy := clampFloat(y, top, bottom)
	dx := x - cx
	dy := y - cy
	return dx*dx+dy*dy <= r*r
}

func drawSurfacePixel(img *render.Image, x, y int, c color.RGBA, coverage float64) {
	if c.A == 0 || coverage <= 0 {
		return
	}
	if coverage > 1 {
		coverage = 1
	}
	c.A = uint8(float64(c.A)*coverage + 0.5)
	if c.A == 0 {
		return
	}
	blendStraightAlphaPixel(img, x, y, c)
}

func blendStraightAlphaPixel(img *render.Image, x, y int, src color.RGBA) {
	rgba := img.RGBA()
	if rgba == nil {
		return
	}
	bounds := rgba.Bounds()
	if x < 0 || y < 0 || x >= bounds.Dx() || y >= bounds.Dy() {
		return
	}
	off := rgba.PixOffset(x+bounds.Min.X, y+bounds.Min.Y)
	sa := float64(src.A) / 255
	da := float64(rgba.Pix[off+3]) / 255
	outA := sa + da*(1-sa)
	if outA <= 0 {
		rgba.Pix[off], rgba.Pix[off+1], rgba.Pix[off+2], rgba.Pix[off+3] = 0, 0, 0, 0
		return
	}
	dr := float64(rgba.Pix[off])
	dg := float64(rgba.Pix[off+1])
	db := float64(rgba.Pix[off+2])
	rgba.Pix[off] = uint8((float64(src.R)*sa+dr*da*(1-sa))/outA + 0.5)
	rgba.Pix[off+1] = uint8((float64(src.G)*sa+dg*da*(1-sa))/outA + 0.5)
	rgba.Pix[off+2] = uint8((float64(src.B)*sa+db*da*(1-sa))/outA + 0.5)
	rgba.Pix[off+3] = uint8(outA*255 + 0.5)
}

func Color(c color.RGBA) uiwidget.Color {
	return uiwidget.RGBA8(c.R, c.G, c.B, c.A)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampUnit(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func clampFloat(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
