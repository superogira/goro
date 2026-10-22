package testutil

import (
	"math"

	"github.com/kivutar/goro/game/scene"
)

func ModelPointNear(a, b scene.Point3, epsilon float64) bool {
	return math.Abs(a.X-b.X) <= epsilon && math.Abs(a.Y-b.Y) <= epsilon && math.Abs(a.Z-b.Z) <= epsilon
}
