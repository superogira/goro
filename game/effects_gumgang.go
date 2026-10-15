package game

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

// GumGang / PrimGumGang in Sakexe keep four lightning flashes around the
// caster. Each fades independently, then chooses another position and texture.
func (m *WorldMode) drawGumgangEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	if len(component.textureFiles) == 0 || now.Before(effect.starts) {
		return
	}
	frame := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	right, _, _, ok := projection.BillboardBasis(x, y, z)
	if !ok {
		return
	}
	flashes := 4 * maxInt(1, component.duplicate)
	for i := 0; i < flashes; i++ {
		alpha, cycle := gumgangFlash(frame, i%4)
		if alpha <= 0 {
			continue
		}
		seed := (cycle*flashes + i) * 3
		angle := deterministicAngle(effect, seed)
		textureIndex := hitEffectRandom(effect, seed+1, len(component.textureFiles))
		texture := m.effectFileTexture(ctx.Resources, component.textureFiles[textureIndex])
		if texture == nil {
			continue
		}
		// Original world units are one fifth of a GAT cell. RenderGumGang's
		// square has radius 3.5 and its center is 4 units from the actor.
		size := math.Sqrt2 * 3.5 / 5
		sizeX := size
		sin, cos := math.Sincos(angle)
		onRight := cos*right.x+sin*right.z > 0
		if onRight == (textureIndex == 0 || textureIndex == 2) {
			sizeX = -sizeX // lightning tips face inward, toward the caster
		}
		height := float64(3+hitEffectRandom(effect, seed+2, 8)) / 5
		if cycle == 0 {
			height = float64(hitEffectRandom(effect, seed+2, 10)) / 5
		}
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture,
			x+cos*4/5, y+sin*4/5, z+height, sizeX, size, 0,
			effectComponentTint(component, alpha), false)
	}
}

// The initial pulses are 70, 130, 190, 250. A fresh flash grows by 25 for
// nine ticks, then loses 12 per tick until below 12 (a 27-tick cycle).
func gumgangFlash(frame, index int) (alpha float64, cycle int) {
	initial := 70 + index*60
	firstReset := (initial-12)/12 + 1
	if frame < firstReset {
		return float64(initial-frame*12) / 255, 0
	}
	frame -= firstReset
	cycle = 1 + frame/27
	phase := frame % 27
	if phase <= 9 {
		return float64(phase*25) / 255, cycle
	}
	return float64(225-(phase-9)*12) / 255, cycle
}
