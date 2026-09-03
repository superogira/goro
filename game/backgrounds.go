package game

import (
	"image/color"
	"math/rand"
	"strings"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

// backgroundPoolCandidates splits a comma-separated pool setting into file
// names, preserving order.
func backgroundPoolCandidates(pool string) []string {
	var out []string
	for _, raw := range strings.Split(pool, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	return out
}

// loadRandomBackground loads one random image from a configured pool. Files
// are looked up in the background folder at the data root (next to bgm\),
// then by their bare name, and may be jpg/png/bmp/tga.
func loadRandomBackground(manager *res.Manager, pool string) *render.Image {
	names := backgroundPoolCandidates(pool)
	if len(names) == 0 {
		return nil
	}
	name := names[rand.Intn(len(names))]
	img, _, err := res.LoadImageExact(manager, []string{
		"background\\" + name,
		"background/" + name,
		"Background\\" + name,
		"Background/" + name,
		name,
	})
	if err != nil {
		return nil
	}
	return render.NewImageFromImage(img)
}

// drawLoadingScreen renders the Now Loading cover: an optional background
// image scaled to cover the screen (fallback plain black), a dim veil so
// the text stays readable, and the centered Now Loading label.
func drawLoadingScreen(screen *render.Frame, img *render.Image) {
	bounds := screen.Bounds()
	width, height := float64(bounds.Dx()), float64(bounds.Dy())
	if img != nil {
		b := img.Bounds()
		if b.Dx() > 0 && b.Dy() > 0 {
			scale := width / float64(b.Dx())
			if s := height / float64(b.Dy()); s > scale {
				scale = s
			}
			var opts render.DrawImageOptions
			opts.GeoM.Scale(scale, scale)
			opts.GeoM.Translate((width-float64(b.Dx())*scale)/2, (height-float64(b.Dy())*scale)/2)
			opts.Filter = render.FilterLinear
			screen.DrawImage(img, &opts)
		}
	} else {
		render.DrawRect(screen, 0, 0, width, height, color.RGBA{A: 255})
	}
	render.DrawRect(screen, 0, 0, width, height, color.RGBA{A: 110})
	if text := render.OutlinedTextImage("Now Loading...", color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBA{A: 190}); text != nil {
		var opts render.DrawImageOptions
		opts.GeoM.Translate((width-float64(text.Bounds().Dx()))/2, (height-float64(text.Bounds().Dy()))/2)
		screen.DrawImage(text, &opts)
	}
}
