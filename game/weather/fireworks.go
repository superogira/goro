package weather

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	worldstate "github.com/kivutar/goro/world"
)

const (
	pokJukWeatherSpread         = 10
	pokJukWeatherLaunchInterval = 1200 * time.Millisecond
	pokJukWeatherRiseDuration   = 154 * time.Second / 60
	pokJukWeatherBurstDuration  = 1200 * time.Millisecond
	pokJukWeatherRiseHeight     = 8
	pokJukWeatherBurstParticles = 30
	pokJukWeatherTrailParticles = 2
)

var pokJukWeatherColors = []color.RGBA{
	{R: 100, G: 160, B: 255, A: 255},
	{R: 255, G: 100, B: 100, A: 255},
	{R: 100, G: 255, B: 100, A: 255},
	{R: 255, G: 255, B: 130, A: 255},
	{R: 255, G: 130, B: 255, A: 255},
}

type mapWeatherFireworkState struct {
	key         string
	initialized bool
	rockets     [pokJukWeatherFireworks]mapWeatherPokJukRocket
}

type mapWeatherPokJukRocket struct {
	launchTime   time.Time
	generation   int
	x            float64
	y            float64
	z            float64
	driftX       float64
	driftY       float64
	color        color.RGBA
	textureIndex int
}

func mapWeatherFireworkCenter(world *worldstate.World, projection scene.Projection, now time.Time) (float64, float64) {
	if world == nil {
		return projection.PlayerX, projection.PlayerY
	}
	return scene.ActorRenderPosition(world.Player, now)
}

func (s *mapWeatherFireworkState) reset() {
	*s = mapWeatherFireworkState{}
}

func (s *mapWeatherFireworkState) ensure(key string, world *worldstate.World, centerX, centerY float64, now time.Time) {
	if s.initialized && s.key == key {
		return
	}
	s.reset()
	s.key = key
	s.initialized = true
	for i := range s.rockets {
		launchTime := now.Add(-time.Duration(i) * pokJukWeatherLaunchInterval)
		s.spawn(i, world, centerX, centerY, launchTime)
	}
}

func (s *mapWeatherFireworkState) update(world *worldstate.World, centerX, centerY float64, now time.Time) {
	cycle := pokJukWeatherCycle()
	for i := range s.rockets {
		rocket := &s.rockets[i]
		if rocket.launchTime.IsZero() {
			s.spawn(i, world, centerX, centerY, now)
			continue
		}
		for !now.Before(rocket.launchTime.Add(cycle)) {
			rocket.generation++
			s.spawn(i, world, centerX, centerY, rocket.launchTime.Add(cycle))
		}
	}
}

func (s *mapWeatherFireworkState) spawn(index int, world *worldstate.World, centerX, centerY float64, launchTime time.Time) {
	rocket := &s.rockets[index]
	generation := rocket.generation
	x := centerX + mapWeatherPokJukHashSigned(index, generation, 1)*pokJukWeatherSpread
	y := centerY + mapWeatherPokJukHashSigned(index, generation, 2)*pokJukWeatherSpread
	angle := mapWeatherPokJukHash01(index, generation, 3) * 2 * math.Pi
	drift := 0.8 + mapWeatherPokJukHash01(index, generation, 4)*1.0
	colorIndex := int(mapWeatherPokJukHash01(index, generation, 5) * float64(len(pokJukWeatherColors)))
	if colorIndex >= len(pokJukWeatherColors) {
		colorIndex = len(pokJukWeatherColors) - 1
	}
	textureIndex := int(mapWeatherPokJukHash01(index, generation, 6) * 3)
	if textureIndex > 2 {
		textureIndex = 2
	}
	rocket.launchTime = launchTime
	rocket.x = x
	rocket.y = y
	rocket.z = scene.TerrainHeightAt(world, x-0.5, y-0.5) + 1
	rocket.driftX = math.Cos(angle) * drift
	rocket.driftY = math.Sin(angle) * drift
	rocket.color = pokJukWeatherColors[colorIndex]
	rocket.textureIndex = textureIndex
}

func (s *mapWeatherFireworkState) scheduleSounds(m *System, now time.Time, scheduleSound func(time.Time, ...string)) {
	for i := range s.rockets {
		starts := s.rockets[i].launchTime
		m.scheduleMapWeatherSound(i*2, starts, now, scheduleSound, pokJukLaunchSFX)
		m.scheduleMapWeatherSound(i*2+1, starts.Add(pokJukWeatherRiseDuration), now, scheduleSound, pokJukExplosionSFX, pokJukLaunchSFX)
	}
}

func (s *mapWeatherFireworkState) draw(screen *render.Frame, fx *effects.System, ctx client.Context, projection scene.Projection, now time.Time) {
	textures := [3]*render.Image{
		fx.EffectFileTexture(ctx.Resources, "effect/pok1.tga"),
		fx.EffectFileTexture(ctx.Resources, "effect/pok2.tga"),
		fx.EffectFileTexture(ctx.Resources, "effect/pok3.tga"),
	}
	options := effects.TexturedEffectBillboardDrawOptions(true, true)
	options.DisableFog = true
	for i := range s.rockets {
		rocket := s.rockets[i]
		elapsed := now.Sub(rocket.launchTime)
		if elapsed < 0 {
			continue
		}
		switch {
		case elapsed < pokJukWeatherRiseDuration:
			s.drawRisingRocket(screen, projection, textures, options, rocket, elapsed)
		case elapsed < pokJukWeatherRocketLifetime():
			s.drawBurstRocket(screen, projection, textures, options, i, rocket, elapsed-pokJukWeatherRiseDuration)
		}
	}
}

func (s *mapWeatherFireworkState) drawRisingRocket(screen *render.Frame, projection scene.Projection, textures [3]*render.Image, options *render.DrawTrianglesOptions, rocket mapWeatherPokJukRocket, elapsed time.Duration) {
	progress := scene.ClampFloat(float64(elapsed)/float64(pokJukWeatherRiseDuration), 0, 1)
	x := rocket.x + rocket.driftX*progress*progress
	y := rocket.y + rocket.driftY*progress*progress
	z := rocket.z + pokJukWeatherRiseHeight*progress
	texture := pokJukWeatherTexture(textures, rocket.textureIndex+int(elapsed/(100*time.Millisecond)))
	alpha := scene.ClampFloat(progress*2, 0, 1)
	tint := color.RGBA{R: 255, G: 255, B: 255, A: uint8(alpha * 255)}
	drawPokJukParticle(screen, projection, texture, x, y, z, pokJukWeatherSize(8), 0, tint, options)
	for trail := 1; trail <= pokJukWeatherTrailParticles; trail++ {
		trailProgress := scene.ClampFloat(progress-float64(trail)*0.08, 0, 1)
		trailX := rocket.x + rocket.driftX*trailProgress*trailProgress
		trailY := rocket.y + rocket.driftY*trailProgress*trailProgress
		trailZ := rocket.z + pokJukWeatherRiseHeight*trailProgress
		trailTint := tint
		trailTint.A = uint8(alpha * 255 / float64(trail+1))
		drawPokJukParticle(screen, projection, texture, trailX, trailY, trailZ, pokJukWeatherSize(6), 0, trailTint, options)
	}
}

func (s *mapWeatherFireworkState) drawBurstRocket(screen *render.Frame, projection scene.Projection, textures [3]*render.Image, options *render.DrawTrianglesOptions, index int, rocket mapWeatherPokJukRocket, elapsed time.Duration) {
	progress := scene.ClampFloat(float64(elapsed)/float64(pokJukWeatherBurstDuration), 0, 1)
	centerX := rocket.x + rocket.driftX
	centerY := rocket.y + rocket.driftY
	centerZ := rocket.z + pokJukWeatherRiseHeight
	alpha := 1 - progress
	for particle := 0; particle < pokJukWeatherBurstParticles; particle++ {
		heading := mapWeatherPokJukHash01(index*100+particle, rocket.generation, 11) * 2 * math.Pi
		elevation := mapWeatherPokJukHashSigned(index*100+particle, rocket.generation, 12)
		distance := 1.0 + mapWeatherPokJukHash01(index*100+particle, rocket.generation, 13)*3.5
		radius := distance * math.Sin(progress*math.Pi*0.75)
		x := centerX + math.Cos(heading)*radius
		y := centerY + math.Sin(heading)*radius
		z := centerZ + elevation*radius*0.75 + progress*0.6
		size := pokJukWeatherSize(10 - progress*5 + mapWeatherPokJukHash01(index*100+particle, rocket.generation, 14)*3)
		tint := rocket.color
		tint.A = uint8(alpha * 255)
		texture := pokJukWeatherTexture(textures, rocket.textureIndex+particle)
		angle := mapWeatherPokJukHash01(index*100+particle, rocket.generation, 15)*2*math.Pi + progress*math.Pi
		drawPokJukParticle(screen, projection, texture, x, y, z, size, angle, tint, options)
	}
}

func drawPokJukParticle(screen *render.Frame, projection scene.Projection, texture *render.Image, x, y, z, size, angle float64, tint color.RGBA, options *render.DrawTrianglesOptions) {
	if texture == nil || tint.A == 0 {
		return
	}
	effects.DrawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, x, y, z, size, size, angle, tint, options)
}

func pokJukWeatherTexture(textures [3]*render.Image, index int) *render.Image {
	if index < 0 {
		index = -index
	}
	if texture := textures[index%len(textures)]; texture != nil {
		return texture
	}
	for _, texture := range textures {
		if texture != nil {
			return texture
		}
	}
	return nil
}

func pokJukWeatherSize(value float64) float64 {
	return value * scene.PixelRatio
}

func pokJukWeatherRocketLifetime() time.Duration {
	return pokJukWeatherRiseDuration + pokJukWeatherBurstDuration
}

func pokJukWeatherCycle() time.Duration {
	return pokJukWeatherLaunchInterval * pokJukWeatherFireworks
}

func mapWeatherPokJukHashSigned(index, generation, salt int) float64 {
	return mapWeatherPokJukHash01(index, generation, salt)*2 - 1
}

func mapWeatherPokJukHash01(index, generation, salt int) float64 {
	return weatherCloudHash01(index, generation, salt)
}
