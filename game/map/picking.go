package maps

import (
	"math"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
)

func ClickedWalkTarget(ctx client.Context, projection scene.Projection, mouseX, mouseY int) (int, int, bool) {
	return rayWalkCell(ctx, projection, mouseX, mouseY)
}

func HoveredWalkCell(ctx client.Context, projection scene.Projection, mouseX, mouseY int) (int, int, bool) {
	return rayWalkCell(ctx, projection, mouseX, mouseY)
}

const (
	walkRayStep        = 0.5
	walkRayMaxDistance = 300.0
	walkRayHitEpsilon  = 0.5
)

func rayWalkCell(ctx client.Context, projection scene.Projection, mouseX, mouseY int) (int, int, bool) {
	if ctx.World == nil || ctx.World.GAT == nil {
		return 0, 0, false
	}
	eye, dir, ok := projection.ScreenRay(mouseX, mouseY)
	if !ok || dir.Y >= 0 {
		return 0, 0, false
	}
	previousDelta := math.Inf(1)
	for distance := 1.0; distance <= walkRayMaxDistance; distance += walkRayStep {
		point := scene.Add3(eye, scene.Mul3(dir, distance))
		x, y := rayPointGATCell(point)
		if !ctx.World.GAT.InBounds(x, y) {
			previousDelta = math.Inf(1)
			continue
		}
		height := scene.TerrainHeightAtRenderPoint(ctx.World, point.X, point.Z)
		delta := point.Y - height
		if math.Abs(delta) <= walkRayHitEpsilon || (previousDelta > 0 && delta < 0) {
			if ctx.World.GAT.Walkable(x, y) {
				return x, y, true
			}
			return 0, 0, false
		}
		previousDelta = delta
	}
	return 0, 0, false
}

func rayPointGATCell(point scene.Point3) (int, int) {
	return int(math.Floor(point.X)), int(math.Floor(point.Z))
}
