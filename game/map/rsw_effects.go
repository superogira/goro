package maps

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func (m *System) DrawRSWEffects(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time, fx func() *effects.System) {
	if !effects.Enabled(ctx) {
		return
	}
	if screen == nil || ctx.World == nil || ctx.World.RSW == nil || ctx.World.GND == nil {
		return
	}
	radius := RSMRenderRadius()
	for index, effect := range ctx.World.RSW.Effects {
		effectID := int(effect.ID)
		spec, ok := effects.Lookup(effectID)
		if !ok {
			continue
		}
		worldX, worldY, worldZ := rswEffectWorldPosition(ctx.World.GND, effect)
		if math.Abs(worldX-projection.PlayerX) > radius*2 || math.Abs(worldY-projection.PlayerY) > radius*2 {
			continue
		}
		m.drawRSWEffect(screen, ctx, projection, spec, effectID, index, effect, worldX, worldY, worldZ, now, fx)
	}
}

func (m *System) drawRSWEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, spec effects.Spec, effectID int, index int, rswEffect res.RSWEffect, worldX, worldY, worldZ float64, now time.Time, fx func() *effects.System) {
	effect := effects.Instance{
		EffectID: effectID,
		X:        int(math.Round(worldX)),
		Y:        int(math.Round(worldY)),
		Expires:  now.Add(24 * time.Hour),
		Duration: 24 * time.Hour,
	}
	for componentIndex, component := range spec.Components {
		component = MapEffectComponent(component)
		duration := fx().WorldEffectResolvedComponentDuration(ctx, spec, component)
		if duration <= 0 {
			duration = 600 * time.Millisecond
		}
		starts := loopingRSWEffectStart(rswEffect, index, duration, now)
		effect.Starts = starts
		progress := effects.WorldEffectComponentProgress(starts, duration, now)
		fx().DrawWorldEffectComponent(screen, ctx, projection, effect, component, componentIndex, worldX, worldY, worldZ, progress, duration, now)
	}
}

func MapEffectComponent(component effects.Component) effects.Component {
	if component.Kind == effects.Component3D && component.SpriteFile != "" {
		component.WorldSizedSprite = true
	}
	return component
}

func rswEffectWorldPosition(gnd *res.GND, effect res.RSWEffect) (float64, float64, float64) {
	// robr subtracts 0.5 from authored map effect objects, then SpriteRenderer.vs
	// adds +0.5 back. Goro projects directly, so use the final world position.
	if gnd == nil {
		return float64(effect.Position.X), float64(effect.Position.Z), float64(effect.Position.Y) + 1
	}
	return float64(effect.Position.X) + float64(gnd.Width), float64(effect.Position.Z) + float64(gnd.Height), float64(effect.Position.Y) + 1
}

func loopingRSWEffectStart(effect res.RSWEffect, index int, duration time.Duration, now time.Time) time.Time {
	if duration <= 0 {
		return now
	}
	phase := time.Duration(effect.Delay*float32(time.Millisecond)) + time.Duration(index%17)*17*time.Millisecond
	origin := time.Unix(0, 0).Add(phase)
	elapsed := now.Sub(origin)
	if elapsed < 0 {
		return origin
	}
	return now.Add(-(elapsed % duration))
}
