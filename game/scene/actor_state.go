package scene

import "image/color"

func MultiplyTint(tint color.RGBA, red, green, blue, alpha float64) color.RGBA {
	return color.RGBA{
		R: uint8(ClampFloat(float64(tint.R)*red, 0, 255)),
		G: uint8(ClampFloat(float64(tint.G)*green, 0, 255)),
		B: uint8(ClampFloat(float64(tint.B)*blue, 0, 255)),
		A: uint8(ClampFloat(float64(tint.A)*alpha, 0, 255)),
	}
}
