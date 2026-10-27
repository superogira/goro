package maps

import (
	"image"
	"image/color"
	"math"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

// RSM walk surfaces can be coplanar with the GAT and use layered depth biases.
// Keep the cursor just above the largest RSM layer while remaining far below
// the old 0.001 bias that let it show through furniture.
const TileCursorDepthBias = rsmModelDepthBias + rsmHorizontalDepthBiasLayers*rsmHorizontalDepthBiasStep

func (m *System) DrawTileCursor(screen *render.Frame, manager *res.Manager, gat *res.GAT, projection scene.Projection, x, y int) {
	verts, ok := tileCursorCellVerts(gat, x, y)
	if !ok {
		return
	}
	points := projectTileCursorVerts(projection, verts)
	if quadHasInvalidPoint(points) || quadOutside(points, float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())) {
		return
	}
	drawTileCursorSurface3D(screen, m.tileCursorTexture(manager), verts)
}

func tileCursorCellVerts(gat *res.GAT, x, y int) ([4]scene.Point3, bool) {
	if gat == nil {
		return [4]scene.Point3{}, false
	}
	cell, ok := gat.Cell(x, y)
	if !ok {
		return [4]scene.Point3{}, false
	}
	return [4]scene.Point3{
		{X: float64(x), Y: float64(cell.Heights[0]), Z: float64(y)},
		{X: float64(x + 1), Y: float64(cell.Heights[1]), Z: float64(y)},
		{X: float64(x), Y: float64(cell.Heights[2]), Z: float64(y + 1)},
		{X: float64(x + 1), Y: float64(cell.Heights[3]), Z: float64(y + 1)},
	}, true
}

func projectTileCursorVerts(projection scene.Projection, verts [4]scene.Point3) [4]scene.Point2 {
	return [4]scene.Point2{
		projection.Project(verts[0].X, verts[0].Z, verts[0].Y),
		projection.Project(verts[1].X, verts[1].Z, verts[1].Y),
		projection.Project(verts[2].X, verts[2].Z, verts[2].Y),
		projection.Project(verts[3].X, verts[3].Z, verts[3].Y),
	}
}

func quadOutside(points [4]scene.Point2, width, height float64) bool {
	minX, minY := float64(points[0].X), float64(points[0].Y)
	maxX, maxY := minX, minY
	for _, point := range points[1:] {
		minX = math.Min(minX, float64(point.X))
		minY = math.Min(minY, float64(point.Y))
		maxX = math.Max(maxX, float64(point.X))
		maxY = math.Max(maxY, float64(point.Y))
	}
	return maxX < -32 || maxY < -32 || minX > width+32 || minY > height+32
}

func quadHasInvalidPoint(points [4]scene.Point2) bool {
	for _, point := range points {
		if !scene.IsFinite(float64(point.X)) || !scene.IsFinite(float64(point.Y)) {
			return true
		}
		if point.X <= -1<<19 && point.Y <= -1<<19 {
			return true
		}
	}
	return false
}

func (m *System) tileCursorTexture(manager *res.Manager) *render.Image {
	if m.tileCursor != nil {
		return m.tileCursor
	}
	if manager != nil {
		if img, _, err := res.LoadImageExact(manager, []string{"data\\texture\\grid.tga", "data/texture/grid.tga"}); err == nil {
			m.tileCursor = render.NewImageFromImage(tintTileCursorImage(img))
			return m.tileCursor
		}
	}
	m.tileCursor = render.NewImageFromImage(fallbackTileCursorImage())
	return m.tileCursor
}

func tintTileCursorImage(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	out := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			alpha := uint8((a >> 8) * 153 / 255)
			out.SetNRGBA(x, y, color.NRGBA{R: 50, G: 240, B: 160, A: alpha})
		}
	}
	return out
}

func fallbackTileCursorImage() image.Image {
	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dist := scene.MinInt(scene.MinInt(x, y), scene.MinInt(size-1-x, size-1-y))
			alpha := uint8(0)
			switch {
			case dist < 3:
				alpha = 190
			case dist < 6:
				alpha = 100
			case dist < 11:
				alpha = 32
			}
			if x == y || x == size-1-y {
				alpha = maxUint8(alpha, 34)
			}
			if alpha > 0 {
				img.SetRGBA(x, y, color.RGBA{R: 50, G: 240, B: 160, A: alpha})
			}
		}
	}
	return img
}

func maxUint8(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}

func drawTileCursorSurface3D(screen *render.Frame, texture *render.Image, verts [4]scene.Point3) {
	if texture == nil {
		return
	}
	tints := [4]color.RGBA{
		{R: 255, G: 255, B: 255, A: 255},
		{R: 255, G: 255, B: 255, A: 255},
		{R: 255, G: 255, B: 255, A: 255},
		{R: 255, G: 255, B: 255, A: 255},
	}
	uvs := [4]scene.UV{
		{U: 0, V: 0},
		{U: 1, V: 0},
		{U: 0, V: 1},
		{U: 1, V: 1},
	}
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	options.DepthBias = TileCursorDepthBias
	scene.DrawTexturedSurface3DWithOptions(screen, texture, verts, uvs, scene.QuadIndices012213, tints, options)
}
