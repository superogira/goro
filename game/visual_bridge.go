package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/render"
	worldstate "github.com/kivutar/goro/world"
)

func (m *WorldMode) effects() *effects.System {
	if m.mapScene.Textures == nil {
		m.mapScene.Textures = make(map[string]*render.Image)
	}
	if m.mapScene.TextureMiss == nil {
		m.mapScene.TextureMiss = make(map[string]struct{})
	}
	m.effectSystem.Bind(m, m.mapScene.Textures, m.mapScene.TextureMiss)
	return &m.effectSystem
}

func (m *WorldMode) EffectSound(at time.Time, effect effects.Instance, paths ...string) {
	m.scheduleSoundAtWorldEffect(at, effect, paths...)
}

func (m *WorldMode) EffectSoundAtPosition(at time.Time, volume, x, y float64, paths ...string) {
	m.scheduleSoundVolumeAtPosition(at, volume, x, y, paths...)
}

func (m *WorldMode) EffectSoundGlobal(at time.Time, paths ...string) { m.scheduleSound(at, paths...) }

func (m *WorldMode) EffectShake(at time.Time, duration time.Duration) {
	m.startCameraShake(at, duration)
}

func (m *WorldMode) EffectStarted(ctx client.Context, effect effects.Instance, at time.Time) {
	m.applyWorldEffectSideEffects(ctx, effect, at)
}

func (m *WorldMode) EffectCameraYaw(ctx client.Context) float64 {
	yaw := scene.CameraYawForMap(ctx)
	if !scene.CameraRotationLockedForMap(ctx) {
		yaw += m.camera.yawOffset
	}
	return yaw
}

func (m *WorldMode) EffectSpriteTexture(image *render.Image) *render.Image {
	return m.spriteTexture(image)
}

func (m *WorldMode) starFusionHeight(actor worldstate.Actor, now time.Time) float64 {
	active := false
	job := scene.ActorVisualJob(actor)
	if job == db.JobStar2 || job == db.JobStar2B {
		anim, ok := m.actorAnimation(actor.ID, now)
		active = ok && anim.actionFamily != scene.SpriteActionIdle && anim.actionFamily != scene.SpriteActionWalk
	}
	return m.effects().FusionHeight(actor, now, active)
}

func (m *WorldMode) spriteTexture(src *render.Image) *render.Image {
	if !m.mapScene.UpscaleTextures || src == nil {
		return src
	}
	if m.spriteUpscaler == nil {
		m.spriteUpscaler = sprites.NewUpscaler()
	}
	return m.spriteUpscaler.Texture(src, m.mapScene.Images.Own)
}
