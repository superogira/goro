package scene

import (
	"math"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/res"
)

const (
	DefaultCameraFollowLerpPerMS   = 0.006
	DefaultCameraWheelZoomUnits    = 15
	DefaultCameraPinchZoomScale    = 240
	DefaultCameraMinZoom           = 65.0
	DefaultCameraMaxZoom           = 165.0
	DefaultCameraMinPitch          = 205.0
	DefaultCameraMaxPitch          = 245.0
	DefaultCameraPitchDragPerPixel = 0.3
)

func CameraYawForMap(ctx client.Context) float64 {
	if viewPoint, ok := lockedCameraViewPointForMap(ctx); ok {
		return -float64(viewPoint.InitialLongitude)
	}
	if CameraRotationLockedForMap(ctx) {
		return -45
	}
	return SceneCameraYaw()
}

func CameraRotationLockedForMap(ctx client.Context) bool {
	if ctx.Resources == nil || ctx.World == nil {
		return false
	}
	if _, ok := lockedCameraViewPointForMap(ctx); ok {
		return true
	}
	return ctx.Resources.IsIndoorMap(ctx.World.MapName)
}

func lockedCameraViewPointForMap(ctx client.Context) (res.CameraViewPoint, bool) {
	if ctx.Resources == nil || ctx.World == nil {
		return res.CameraViewPoint{}, false
	}
	viewPoint, ok := ctx.Resources.CameraViewPoint(ctx.World.MapName)
	if !ok || !viewPoint.LocksLongitude() {
		return res.CameraViewPoint{}, false
	}
	return viewPoint, true
}

func ClampCameraZoom(zoom float64) float64 {
	if !IsFinite(zoom) || zoom <= 0 {
		zoom = DefaultSceneCameraZoom
	}
	return math.Max(DefaultCameraMinZoom, math.Min(DefaultCameraMaxZoom, zoom))
}

func ClampCameraPitch(pitch float64) float64 {
	if !IsFinite(pitch) || pitch == 0 {
		pitch = DefaultSceneCameraPitch
	}
	return math.Max(DefaultCameraMinPitch, math.Min(DefaultCameraMaxPitch, pitch))
}
