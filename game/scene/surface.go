package scene

import (
	"image/color"
	"math"

	"github.com/kivutar/goro/render"
)

func ClampUnit(value float64) float64 {
	return math.Max(0, math.Min(1, value))
}

func DrawColoredSurfaceTints3DAlpha(screen *render.Frame, white *render.Image, verts [4]Point3, indices []uint16, colors [4]color.RGBA) {
	drawColoredSurfaceTints3DWithOptions(screen, white, verts, indices, colors, TriangleDrawOptions(render.FilterNearest, render.AddressUnsafe))
}

func drawColoredSurfaceTints3DWithOptions(screen *render.Frame, white *render.Image, verts [4]Point3, indices []uint16, colors [4]color.RGBA, options *render.DrawTrianglesOptions) {
	vertices := []render.Vertex3D{
		ColoredSurfaceVertex3D(verts[0], 0, 0, colors[0]),
		ColoredSurfaceVertex3D(verts[1], 1, 0, colors[1]),
		ColoredSurfaceVertex3D(verts[2], 1, 1, colors[2]),
		ColoredSurfaceVertex3D(verts[3], 0, 1, colors[3]),
	}
	screen.DrawTriangles3DOwned(vertices, indices, white, options)
}

func DrawTexturedSurface3DAlpha(screen *render.Frame, texture *render.Image, verts [4]Point3, uvs [4]UV, indices []uint16, tints [4]color.RGBA) {
	DrawTexturedSurface3DWithOptions(screen, texture, verts, uvs, indices, tints, TriangleDrawOptions(render.FilterLinear, render.AddressRepeat))
}

func ColoredSurfaceVertex3D(point Point3, u, v float32, tint color.RGBA) render.Vertex3D {
	return render.Vertex3D{
		X:      float32(point.X),
		Y:      float32(point.Y),
		Z:      float32(point.Z),
		SrcX:   u,
		SrcY:   v,
		ColorR: float32(tint.R) / 255,
		ColorG: float32(tint.G) / 255,
		ColorB: float32(tint.B) / 255,
		ColorA: float32(tint.A) / 255,
		DepthX: float32(point.X),
		DepthY: float32(point.Y),
		DepthZ: float32(point.Z),
	}
}
