package scene

import (
	"image/color"

	"github.com/kivutar/goro/render"
)

var (
	QuadIndices012023 = []uint16{0, 1, 2, 0, 2, 3}
	QuadIndices012213 = []uint16{0, 1, 2, 2, 1, 3}
)

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func AbsInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

type Point2 struct {
	X float32
	Y float32
}

type UV struct {
	U float32
	V float32
}

func ClampColor(value float64) uint8 {
	return uint8(min(255, max(0, int(value))))
}

func DrawTexturedSurface3DWithOptions(screen *render.Frame, texture *render.Image, verts [4]Point3, uvs [4]UV, indices []uint16, tints [4]color.RGBA, options *render.DrawTrianglesOptions) {
	bounds := texture.Bounds()
	w := float32(bounds.Dx())
	h := float32(bounds.Dy())
	vertices := []render.Vertex3D{
		TexturedSurfaceVertex3D(verts[0], uvs[0], tints[0], w, h),
		TexturedSurfaceVertex3D(verts[1], uvs[1], tints[1], w, h),
		TexturedSurfaceVertex3D(verts[2], uvs[2], tints[2], w, h),
		TexturedSurfaceVertex3D(verts[3], uvs[3], tints[3], w, h),
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, options)
}

func TexturedSurfaceVertex3D(point Point3, uv UV, tint color.RGBA, textureWidth, textureHeight float32) render.Vertex3D {
	return render.Vertex3D{
		X:      float32(point.X),
		Y:      float32(point.Y),
		Z:      float32(point.Z),
		SrcX:   uv.U * textureWidth,
		SrcY:   uv.V * textureHeight,
		ColorR: float32(tint.R) / 255,
		ColorG: float32(tint.G) / 255,
		ColorB: float32(tint.B) / 255,
		ColorA: float32(tint.A) / 255,
		DepthX: float32(point.X),
		DepthY: float32(point.Y),
		DepthZ: float32(point.Z),
	}
}
