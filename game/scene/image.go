package scene

import (
	"image"

	"github.com/kivutar/goro/render"
)

func VisibleImageBounds(img *render.Image) image.Rectangle {
	rgba := img.RGBA()
	if rgba == nil {
		return image.Rectangle{}
	}
	bounds := rgba.Bounds()
	visible := image.Rectangle{Min: bounds.Max, Max: bounds.Min}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if rgba.RGBAAt(x, y).A == 0 {
				continue
			}
			if x < visible.Min.X {
				visible.Min.X = x
			}
			if y < visible.Min.Y {
				visible.Min.Y = y
			}
			if x+1 > visible.Max.X {
				visible.Max.X = x + 1
			}
			if y+1 > visible.Max.Y {
				visible.Max.Y = y + 1
			}
		}
	}
	if visible.Min.X > visible.Max.X || visible.Min.Y > visible.Max.Y {
		return image.Rectangle{}
	}
	return visible.Inset(-1).Intersect(bounds)
}
