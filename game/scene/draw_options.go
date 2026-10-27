package scene

import (
	"github.com/kivutar/goro/render"
)

const GroundDecalDepthBias = 1.0 / 32768.0

func TriangleDrawOptions(filter render.Filter, address render.Address) *render.DrawTrianglesOptions {
	return &render.DrawTrianglesOptions{
		Filter:    filter,
		Address:   address,
		DepthTest: true,
	}
}
