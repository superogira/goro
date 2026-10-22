package game

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/input"
	worldstate "github.com/kivutar/goro/world"
)

type followCamera struct {
	initialized bool
	x           float64
	y           float64
	z           float64
	lastUpdate  time.Time
	yawOffset   float64
	yawTarget   float64
	pitch       float64
	zoom        float64
	zoomTarget  float64
}

func (c *followCamera) ResetTracking() {
	yawOffset, yawTarget, pitch, zoom, zoomTarget := c.yawOffset, c.yawTarget, c.pitch, c.zoom, c.zoomTarget
	*c = followCamera{
		yawOffset:  yawOffset,
		yawTarget:  yawTarget,
		pitch:      pitch,
		zoom:       zoom,
		zoomTarget: zoomTarget,
	}
}

func (c *followCamera) Update(ctx client.Context, now time.Time) {
	targetX, targetY, targetZ := playerCameraTarget(ctx.World, now)
	if !c.initialized {
		c.x = targetX
		c.y = targetY
		c.z = targetZ
		c.zoom = c.targetZoom()
		c.lastUpdate = now
		c.initialized = true
		c.store(ctx)
		return
	}
	delta := now.Sub(c.lastUpdate)
	lerp := cameraFollowLerp(delta)
	c.lastUpdate = now
	c.x += (targetX - c.x) * lerp
	c.y += (targetY - c.y) * lerp
	c.z += (targetZ - c.z) * lerp
	zoomLerp := cameraZoomLerp(lerp)
	c.zoom += (c.targetZoom() - c.currentZoom()) * zoomLerp
	c.yawOffset = normalizeCameraYaw(c.yawOffset + cameraYawShortestDelta(c.yawOffset, c.yawTarget)*lerp)
	c.store(ctx)
}

func cameraFollowLerp(delta time.Duration) float64 {
	if delta <= 0 {
		return 0
	}
	return math.Min(float64(delta)/float64(time.Millisecond)*scene.DefaultCameraFollowLerpPerMS, 1.0)
}

// Camera button steps for the on-screen touch controls. Yaw is measured in
// degrees (cameraDragYawDelta spans 720° across the screen; the earlier
// radian constants here were invisible ~1-6° nudges).
const (
	cameraButtonZoomStep   = 1.25
	cameraButtonRotateStep = 45.0
)

// Rotate moves the yaw target; the displayed yaw eases toward it in
// Update (fast at first, slowing as it arrives — the same exponential
// settle the zoom uses).
func (c *followCamera) Rotate(delta float64) {
	c.yawTarget = normalizeCameraYaw(c.yawTarget + delta)
}

// RotateImmediate sets both target and displayed yaw — continuous inputs
// (right-drag, two-finger pan) must track the finger 1:1 with no lag.
func (c *followCamera) RotateImmediate(delta float64) {
	c.yawTarget = normalizeCameraYaw(c.yawTarget + delta)
	c.yawOffset = c.yawTarget
}

func (c *followCamera) Tilt(delta float64) {
	if delta == 0 || !scene.IsFinite(delta) {
		return
	}
	c.pitch = scene.ClampCameraPitch(c.currentPitch() + delta)
}

func (c *followCamera) currentPitch() float64 {
	if c.pitch == 0 || !scene.IsFinite(c.pitch) {
		return scene.CameraPitch()
	}
	return scene.ClampCameraPitch(c.pitch)
}

func (c *followCamera) ZoomBy(factor float64) {
	if factor <= 0 || !scene.IsFinite(factor) {
		return
	}
	c.zoomTarget = scene.ClampCameraZoom(c.targetZoom() * factor)
}

func (c *followCamera) ZoomByDelta(delta float64) {
	if delta == 0 || !scene.IsFinite(delta) {
		return
	}
	c.zoomTarget = scene.ClampCameraZoom(c.targetZoom() + delta)
}

func (c *followCamera) currentZoom() float64 {
	if c.zoom <= 0 || !scene.IsFinite(c.zoom) {
		return c.targetZoom()
	}
	return scene.ClampCameraZoom(c.zoom)
}

func (c *followCamera) targetZoom() float64 {
	if c.zoomTarget <= 0 || !scene.IsFinite(c.zoomTarget) {
		if c.zoom > 0 && scene.IsFinite(c.zoom) {
			return scene.ClampCameraZoom(c.zoom)
		}
		return scene.CameraZoom()
	}
	return scene.ClampCameraZoom(c.zoomTarget)
}

func cameraZoomLerp(followLerp float64) float64 {
	if followLerp <= 0 {
		return 0
	}
	return math.Min(followLerp*2, 1.0)
}

func (c *followCamera) ResetRotation() {
	c.yawTarget = 0
}

func (c *followCamera) Projection(ctx client.Context, width, height int, now time.Time) scene.Projection {
	return c.ProjectionWithOffset(ctx, width, height, now, 0, 0)
}

func (c *followCamera) ProjectionWithOffset(ctx client.Context, width, height int, now time.Time, offsetX, offsetY float64) scene.Projection {
	if !c.initialized {
		c.Update(ctx, now)
	}
	c.store(ctx)
	yawOffset := c.yawOffset
	if scene.CameraRotationLockedForMap(ctx) {
		yawOffset = 0
	}
	outdoorZoom := c.currentZoom()
	projectedZoom := cameraZoomForMap(ctx, outdoorZoom)
	projectedPitch := c.currentPitch()
	if scene.CameraRotationLockedForMap(ctx) {
		projectedPitch = scene.CameraPitch()
	}
	return scene.NewProjection(width, height, c.x+offsetX, c.y+offsetY, c.z, scene.CameraYawForMap(ctx)+yawOffset, projectedPitch, projectedZoom)
}

func (c *followCamera) store(ctx client.Context) {
	if ctx.World == nil {
		return
	}
	ctx.World.Camera.X = c.x
	ctx.World.Camera.Y = c.y
}

func (m *WorldMode) sceneProjection(ctx client.Context, width, height int, now time.Time) scene.Projection {
	offsetX, offsetY := m.cameraShakeOffset(now)
	return m.camera.ProjectionWithOffset(ctx, width, height, now, offsetX, offsetY)
}

func (m *WorldMode) startCameraShake(starts time.Time, duration time.Duration) {
	if duration <= 0 {
		return
	}
	m.cameraShakeStart = starts
	m.cameraShakeEnd = starts.Add(duration)
}

func (m *WorldMode) cameraShakeOffset(now time.Time) (float64, float64) {
	if m.cameraShakeStart.IsZero() || now.Before(m.cameraShakeStart) || !now.Before(m.cameraShakeEnd) {
		return 0, 0
	}
	duration := m.cameraShakeEnd.Sub(m.cameraShakeStart)
	if duration <= 0 {
		return 0, 0
	}
	elapsed := now.Sub(m.cameraShakeStart)
	progress := scene.ClampFloat(float64(elapsed)/float64(duration), 0, 1)
	amplitude := 0.18 * (1 - progress)
	seconds := float64(elapsed) / float64(time.Second)
	return math.Sin(seconds*120*2*math.Pi) * amplitude, math.Cos(seconds*150*2*math.Pi) * amplitude
}

func (m *WorldMode) updateCameraRotation(ctx client.Context) {
	if scene.CameraRotationLockedForMap(ctx) {
		return
	}
	delta := 0.0
	screenW, _ := ctx.ScreenSize()
	if ctx.Input.MousePressed(input.MouseButtonRight) {
		if ctx.Input.Pressed(input.KeyShift) {
			m.camera.Tilt(cameraDragPitchDelta(ctx.Input.MouseDY))
			return
		}
		delta = cameraDragYawDelta(ctx.Input.MouseDX, screenW)
	}
	// Two-finger pan rotates the camera with the same sensitivity as
	// right-drag; the vertical component tilts (bounded as usual).
	if panDX := ctx.Input.GesturePanDX; panDX != 0 {
		delta += cameraDragYawDelta(int(panDX), screenW)
	}
	if panDY := ctx.Input.GesturePanDY; math.Abs(panDY) >= 2 {
		m.camera.Tilt(cameraDragPitchDelta(int(panDY)))
	}
	if delta != 0 {
		m.camera.RotateImmediate(delta)
	}
}

func (m *WorldMode) updateCameraZoom(ctx client.Context) {
	if cameraZoomLockedForMap(ctx) {
		return
	}
	factor := 1.0
	if ctx.Input.WheelY != 0 {
		m.camera.ZoomByDelta(cameraWheelZoomDelta(ctx.Input.WheelY))
	}
	if ctx.Input.PinchDelta != 0 {
		factor *= cameraPinchZoomFactor(ctx.Input.PinchDelta)
	}
	// Two-finger pinch: the gesture reports a distance ratio (>1 = fingers
	// spread = zoom in), which is the inverse of the zoom factor.
	if zoom := ctx.Input.GestureZoom; zoom > 0 && zoom != 1 {
		m.camera.ZoomBy(1 / zoom)
	}
	if factor != 1 {
		m.camera.ZoomBy(factor)
	}
}

func playerCameraTarget(world *worldstate.World, now time.Time) (float64, float64, float64) {
	if world == nil {
		return 0.5, 0.5, 0
	}
	playerX, playerY := scene.ActorRenderPosition(world.Player, now)
	return scene.CellCenter(playerX), scene.CellCenter(playerY), cameraTargetHeightAt(world, playerX, playerY)
}

func cameraTargetHeightAt(world *worldstate.World, x, y float64) float64 {
	return scene.TerrainHeightAt(world, x, y)
}

func cameraZoomForMap(ctx client.Context, outdoorZoom float64) float64 {
	if ctx.Resources == nil || ctx.World == nil {
		return scene.ClampCameraZoom(outdoorZoom)
	}
	if viewPoint, ok := ctx.Resources.CameraViewPoint(ctx.World.MapName); ok && viewPoint.DistanceScope <= 0 {
		if viewPoint.InitialDistance > 0 {
			return scene.ClampCameraZoom(float64(viewPoint.InitialDistance))
		}
		return scene.CameraZoom()
	}
	if ctx.Resources.IsIndoorMap(ctx.World.MapName) {
		return scene.CameraZoom()
	}
	return scene.ClampCameraZoom(outdoorZoom)
}

func cameraZoomLockedForMap(ctx client.Context) bool {
	if ctx.Resources == nil || ctx.World == nil {
		return false
	}
	if viewPoint, ok := ctx.Resources.CameraViewPoint(ctx.World.MapName); ok {
		return viewPoint.DistanceScope <= 0
	}
	return ctx.Resources.IsIndoorMap(ctx.World.MapName)
}

func cameraDragYawDelta(mouseDX, screenWidth int) float64 {
	if mouseDX == 0 || screenWidth <= 0 {
		return 0
	}
	return -(float64(mouseDX) / float64(screenWidth)) * 720
}

func cameraDragPitchDelta(mouseDY int) float64 {
	return float64(mouseDY) * scene.DefaultCameraPitchDragPerPixel
}

func cameraWheelZoomDelta(wheelY float64) float64 {
	if wheelY == 0 || !scene.IsFinite(wheelY) {
		return 0
	}
	return -math.Copysign(cameraZoomWheelUnits(), wheelY)
}

func cameraPinchZoomFactor(delta float64) float64 {
	if delta == 0 || !scene.IsFinite(delta) {
		return 1
	}
	return math.Exp(-delta / cameraPinchZoomScale())
}

func cameraZoomWheelUnits() float64 {
	return scene.DefaultCameraWheelZoomUnits
}

func cameraPinchZoomScale() float64 {
	return scene.DefaultCameraPinchZoomScale
}

// cameraYawShortestDelta returns the signed angular difference taking
// the shorter arc, so easing from 170° to -170° travels 20°, not 340°.
func cameraYawShortestDelta(from, to float64) float64 {
	delta := math.Mod(to-from, 360)
	if delta > 180 {
		delta -= 360
	}
	if delta < -180 {
		delta += 360
	}
	return delta
}

func normalizeCameraYaw(yaw float64) float64 {
	yaw = math.Mod(yaw, 360)
	if yaw <= -180 {
		yaw += 360
	}
	if yaw > 180 {
		yaw -= 360
	}
	return yaw
}
