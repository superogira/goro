package effects

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func (m *System) drawWizardWaterBall(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if now.Before(effect.Starts) || frame >= 40 {
		return
	}
	for i := 11; i >= 0; i-- {
		p := splineProjectilePosition(effect, max(0, frame+1-i), 0)
		m.drawEffectParticleSprite(screen, ctx, projection, component.SpriteFile, p, frame/4+1,
			.75*(1-float64(i)/24), 0, color.RGBA{255, 255, 255, uint8(254 * (1 - float64(i)/12))})
	}
}

func (m *System) drawJupitel(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, origin scene.Point3, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if now.Before(effect.Starts) {
		return
	}
	projectile := effect.EffectID == JupitelThunder
	if projectile {
		if frame >= 20 {
			return
		}
		origin = scene.Add3(effect.ActorOrigin, scene.Mul3(effect.ProjectileDirection, float64(frame+1)/20))
	}
	origin.Y++
	draw := func(file string, halfSize, alpha float64) {
		texture := m.EffectFileTexture(ctx.Resources, file)
		DrawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, origin.X, origin.Z, origin.Y,
			halfSize*2/5, halfSize*2/5, 0, color.RGBA{255, 255, 255, uint8(alpha)},
			TexturedEffectBillboardDrawOptions(true, !projectile))
	}
	if projectile {
		draw(component.TextureFile, 3.5, 170)
		draw(component.TextureFiles[(frame+1)%len(component.TextureFiles)], 4.5, 254)
	} else if frame < 250 {
		age := frame % 20
		if age < 10 {
			draw(component.TextureFile, 2.5*float64(age+1), 254*scene.ClampFloat(float64(9-age)/9, 0, 1))
		}
		if frame >= 10 {
			draw(component.TextureFiles[(frame-10+1)%len(component.TextureFiles)], 7.5, 254)
		}
	}
}

func (m *System) drawSightTrasher(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, origin scene.Point3, now time.Time) {
	if now.Before(effect.Starts) {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if frame >= 85 {
		return
	}
	for birth := 0; birth <= min(frame, 15); birth += 5 {
		n := float64(frame - birth + 1)
		change := float64(35 - birth)
		late := max(0, n-change)
		distance := 5 + float64(birth/5) + min(n, change) + .5*late
		for i := 0; i < 8; i++ {
			s, c := math.Sincos(scene.DegreesToRadians(float64(-45 * i)))
			position := scene.Add3(origin, scene.Point3{
				X: (distance*s - .004*late*(late+1)/2) / 5,
				Y: float64(20-birth/3) / 5,
				Z: (-distance*c + .004*late*(late+1)/2) / 5,
			})
			size, speed := 2.5-float64(birth)/10, 3+HitEffectRandom(effect, birth*8+i, 2)
			if component.ShadowTexture {
				size, speed = 1-float64(birth)/30, 3
				position.Y = scene.TerrainHeightAtRenderPoint(ctx.World, position.X, position.Z) + scene.ActorShadowTerrainLift
			}
			alpha := float64(150-2*birth) * scene.ClampFloat(float64(84-frame)/20, 0, 1)
			m.drawEffectParticleSprite(screen, ctx, projection, component.SpriteFile, position, int(n-1)/speed+1, size, 0, color.RGBA{255, 255, 255, uint8(alpha)})
		}
	}
	age := frame - 10
	if !component.ShadowTexture && age >= 0 && age < 30 {
		texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
		if texture == nil {
			return
		}
		n := float64(age + 1)
		radius := (2.5*n - (2.5/45)*n*(n+1)/2) / 5
		vertices, indices := groundEffectRingMesh(texture, origin, radius+2, radius, uint8(150*scene.ClampFloat(float64(29-age)/10, 0, 1)))
		screen.DrawTriangles3DOwned(vertices, indices, texture, scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat))
	}
}

func (m *System) drawFirePillar(screen *render.Frame, ctx client.Context, component Component, effect Instance, origin scene.Point3, now time.Time) {
	if now.Before(effect.Starts) {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	frame := float64(now.Sub(effect.Starts) / db.EffectFrameDuration)
	rotate := func(p scene.Point3) scene.Point3 {
		// Prim3DCylinder advances longitude twice per tick, drawing between them.
		return rotateModelPointY(p, scene.DegreesToRadians(-3*(2*frame+1)))
	}
	for i := 1; i <= 3; i++ {
		height := float64(2*(4-i)*(5-i)) + 3.5
		drawWorldCylinderBandWithOptions(screen, m.whitePixel, texture, origin.X, origin.Z, origin.Y,
			(float64(i)+1.5)/5, float64(i+3)/5, height/5, color.RGBA{255, 255, 255, 240}, 10,
			rotate(scene.Point3{X: 1}), rotate(scene.Point3{Z: 1}), rotate(scene.Point3{Y: 1}), 2.5, render.BlendSourceOver)
	}
}
