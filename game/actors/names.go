package actors

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
)

func (m *System) SiegeGuildEmblemAnchor(ctx client.Context, projection scene.Projection, now time.Time, entry scene.ActorDrawEntry) (float64, float64) {
	centerX := entry.ScreenX
	topY := ActorSpriteTopY(entry.ScreenY, entry.Scale)
	if entry.IsPlayer || !m.nonPCActorHasGR2Model(ctx, entry.Actor) {
		return centerX, topY
	}

	view := m.NonPCGR2ModelView(ctx, entry.Actor)
	if view == nil || view.geometry == nil {
		return centerX, topY
	}
	scale := gr2ModelWorldScale * m.ActorBodySizeMultiplier(entry.Actor.ID, now)
	top := scene.Point3{
		X: float64(view.geometry.Center[0]),
		Y: float64(view.geometry.Center[1]),
		Z: float64(view.geometry.Center[2] + view.geometry.Size[2]/2),
	}
	matrix := gr2ActorModelMatrix(entry.WorldX, entry.WorldY, entry.WorldZ, entry.Actor.Dir, scale)
	worldTop := scene.Mat4TransformPoint(matrix, top)
	projected := projection.Project(worldTop.X, worldTop.Z, worldTop.Y)
	if !scene.IsFinite(float64(projected.X)) || !scene.IsFinite(float64(projected.Y)) {
		return centerX, topY
	}
	return float64(projected.X), float64(projected.Y)
}
