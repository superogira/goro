package effects

import (
	"image/color"
	"math"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func appendBasilicaWalls(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, center scene.Point3, tick int) {
	// Two concentric walls and two moving strips per layer (PP_RECT_UP2).
	for layer := 0; layer < 2; layer++ {
		for i := 0; i < 4; i++ {
			maxHeight, half, fade := 30.0+float64(i), 12.5, 2415
			tint := color.RGBA{200, 250, 200, 65}
			if layer == 1 {
				maxHeight, half, fade = 20+float64(i), 14, 2385
				tint.A = 32
			}
			half += .4 * float64(min(i, 2))
			start, length := 0.0, 8*half
			if i >= 2 {
				start = math.Mod(float64(i-2)*53.2+float64(layer)*26.6+.5*float64(tick), 8*half)
				length = 26.6
				tint = color.RGBA{150, 250, 150, 15}
				fade = 2415
			}
			angle := min(tick, 90)
			if tick > fade {
				angle = max(0, min(90, 2505-layer*30-tick))
			}
			height := maxHeight * math.Sin(scene.DegreesToRadians(float64(angle)))
			appendSquareEffectBand(vertices, indices, texture, center, half/5, height/5, start/5, length/5, tint)
		}
	}
}
