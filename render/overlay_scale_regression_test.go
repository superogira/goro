package render

import (
	"image/color"
	"testing"

	"github.com/gogpu/gpucontext"
	"github.com/gogpu/gputypes"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/text"
)

// TestGGContextScaleSwitchReRaster reproduces the user's exact sequence at the gg
// raster level: a context created at device scale 0.5 (session at 50%), text
// drawn (hover label appears), then SetDeviceScale(1.0) (switch to 100%) and
// the same text redrawn. If the re-raster is correct the ink density must
// match a context created fresh at 1.0.
func TestGGContextScaleSwitchReRaster(t *testing.T) {
	fontData := sarabunRegularTTF
	source, err := text.NewFontSource(fontData)
	if err != nil {
		t.Skipf("font source: %v", err)
	}
	face := source.Face(12)

	ink := func(ctx *gg.Context) int {
		n := 0
		b := ctx.Image().Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bl, _ := ctx.Image().At(x, y).RGBA()
				lum := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bl)) / 257
				if lum > 60 {
					n++
				}
			}
		}
		return n
	}
	draw := func(ctx *gg.Context) {
		ctx.Clear()
		ctx.SetColor(color.White)
		ctx.DrawStringAnchored("Kafra Employee tester", 2, 2, 0, 0)
	}

	switched := gg.NewContextWithScale(300, 50, 0.5)
	switched.SetFont(face)
	draw(switched)
	ink50 := ink(switched)

	// production sequence: measure pass resizes to 1x1, draw pass resizes to
	// the label size, THEN the scale is switched on the next label render.
	switched.SetDeviceScale(1.0)
	switched.Resize(1, 1)
	switched.Resize(300, 50)
	draw(switched)
	inkSwitched := ink(switched)

	fresh := gg.NewContextWithScale(300, 50, 1.0)
	fresh.SetFont(face)
	draw(fresh)
	inkFresh := ink(fresh)

	t.Logf("created@0.5: ink=%d", ink50)
	t.Logf("after 0.5->1.0: ink=%d", inkSwitched)
	t.Logf("fresh@1.0: ink=%d", inkFresh)
	if inkSwitched == ink50 {
		t.Errorf("switch to 1.0 did not re-raster (ink unchanged from 50%%)")
	}
	if inkSwitched < inkFresh*8/10 {
		t.Errorf("switched ink %d far below fresh %d — blurry upscale", inkSwitched, inkFresh)
	}
}

type stubProvider struct{}

func (stubProvider) Device() gpucontext.Device             { return gpucontext.Device{} }
func (stubProvider) Queue() gpucontext.Queue               { return gpucontext.Queue{} }
func (stubProvider) SurfaceFormat() gputypes.TextureFormat { return gputypes.TextureFormatRGBA8Unorm }
func (stubProvider) Adapter() gpucontext.Adapter           { return gpucontext.Adapter{} }
func (stubProvider) AdapterInfo() gpucontext.AdapterInfo   { return gpucontext.AdapterInfo{} }
func (stubProvider) Features() gputypes.Features           { return 0 }
func (stubProvider) DownlevelCapabilities() gputypes.DownlevelCapabilities {
	return gputypes.DownlevelCapabilities{}
}

// TestOverlayLabelScaleSwitchRegression runs the production label raster function
// at deviceScale 0.5 then 1.0 (the user's hover-at-50-then-switch sequence)
// and checks the produced image resolution follows the scale.
func TestOverlayLabelScaleSwitchRegression(t *testing.T) {
	r := &runner{}
	label := UIActorLabelCommand{
		Labels:     []string{"tester (Swordman)"},
		Size:       12,
		Foreground: color.RGBA{R: 255, G: 255, B: 255, A: 255},
	}
	c50, err := r.cachedActorLabelImage(stubProvider{}, label, 0.5)
	if err != nil {
		t.Skipf("needs GPU provider: %v", err)
	}
	// simulate the scale-change invalidation drawUIOverlay performs
	r.uiOverlayScale = 1.0
	r.uiTextCache = nil
	r.uiBubbleCache = nil
	c100, err := r.cachedActorLabelImage(stubProvider{}, label, 1.0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("at 0.5: cached %dx%d image %dx%d", c50.width, c50.height, c50.image.Bounds().Dx(), c50.image.Bounds().Dy())
	t.Logf("at 1.0: cached %dx%d image %dx%d", c100.width, c100.height, c100.image.Bounds().Dx(), c100.image.Bounds().Dy())
	if c100.image.Bounds().Dy() <= c50.image.Bounds().Dy() {
		t.Errorf("label at 1.0 reused the 0.5 raster: image height %d not > %d", c100.image.Bounds().Dy(), c50.image.Bounds().Dy())
	}
}

func BenchmarkOverlayLabelCacheHit(b *testing.B) {
	r := &runner{}
	label := UIActorLabelCommand{
		Labels:     []string{"tester (Swordman)"},
		Size:       12,
		Foreground: color.RGBA{R: 255, G: 255, B: 255, A: 255},
	}
	if _, err := r.cachedActorLabelImage(stubProvider{}, label, 1.0); err != nil {
		b.Skip(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := r.cachedActorLabelImage(stubProvider{}, label, 1.0); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOverlayLabelRaster(b *testing.B) {
	label := UIActorLabelCommand{
		Labels:     []string{"tester (Swordman)"},
		Size:       12,
		Foreground: color.RGBA{R: 255, G: 255, B: 255, A: 255},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := &runner{}
		if _, err := r.cachedActorLabelImage(stubProvider{}, label, 1.0); err != nil {
			b.Fatal(err)
		}
	}
}
