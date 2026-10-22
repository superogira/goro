package scene

import (
	"image/color"
	"math"
)

func WithAlpha(c color.RGBA, alpha float64) color.RGBA {
	alpha = ClampFloat(alpha, 0, 1)
	c.A = uint8(math.Round(float64(c.A) * alpha))
	return c
}

func ClampFloat(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
