package game

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestFollowCameraInitializesToRenderedPlayerPosition(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{
		X:            20,
		Y:            30,
		Moving:       true,
		FromX:        10,
		FromY:        20,
		ToX:          20,
		ToY:          30,
		MoveStarted:  now.Add(-750 * time.Millisecond),
		MoveDuration: 1500 * time.Millisecond,
	}
	ctx := client.Context{World: world}

	camera := followCamera{}
	camera.Update(ctx, now)

	if camera.x != 15.5 || camera.y != 25.5 {
		t.Fatalf("camera target = %.2f, %.2f, want rendered player center 15.5, 25.5", camera.x, camera.y)
	}
	if world.Camera.X != camera.x || world.Camera.Y != camera.y {
		t.Fatalf("world camera = %.2f, %.2f, want %.2f, %.2f", world.Camera.X, world.Camera.Y, camera.x, camera.y)
	}
}

func TestFollowCameraEasesTowardRenderedPlayerLikeReferenceView(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	ctx := client.Context{World: world}

	camera := followCamera{}
	now := time.Now()
	camera.Update(ctx, now)
	world.Player = worldstate.Actor{X: 14, Y: 20}
	camera.Update(ctx, now.Add(time.Second/60))

	if math.Abs(camera.x-10.9) > 0.001 || camera.y != 20.5 {
		t.Fatalf("camera target = %.3f, %.3f, want 10.900, 20.5", camera.x, camera.y)
	}
}

func TestCameraFollowLerpClampsLikeReferenceView(t *testing.T) {
	if got := cameraFollowLerp(100 * time.Millisecond); math.Abs(got-0.6) > 0.001 {
		t.Fatalf("camera lerp = %.2f, want 0.60", got)
	}
	if got := cameraFollowLerp(time.Second); got != 1 {
		t.Fatalf("camera lerp = %.2f, want clamped 1.00", got)
	}
}

func TestCameraZoomLerpMatchesRobrowserZoomCurve(t *testing.T) {
	if got := cameraZoomLerp(cameraFollowLerp(time.Second / 60)); math.Abs(got-0.2) > 0.001 {
		t.Fatalf("zoom lerp = %.3f, want 0.200", got)
	}
	if got := cameraZoomLerp(cameraFollowLerp(100 * time.Millisecond)); got != 1 {
		t.Fatalf("large zoom lerp = %.2f, want clamped 1.00", got)
	}
}

func TestFollowCameraZoomEasesTowardTargetLikeRobrowser(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	ctx := client.Context{World: world}

	now := time.Now()
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, zoom: 125, zoomTarget: 125, lastUpdate: now}
	camera.ZoomByDelta(-15)

	if got := camera.currentZoom(); got != 125 {
		t.Fatalf("current zoom before update = %.1f, want unchanged 125.0", got)
	}
	if got := camera.targetZoom(); got != 110 {
		t.Fatalf("target zoom = %.1f, want 110.0", got)
	}

	camera.Update(ctx, now.Add(time.Second/60))
	if math.Abs(camera.currentZoom()-122) > 0.01 {
		t.Fatalf("smoothed zoom = %.2f, want 122.00", camera.currentZoom())
	}
}

func TestIndoorCameraZoomIsLockedWithoutLosingOutdoorZoom(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "indoorrswtable.txt"), []byte("geffen_in.rsw#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	world := worldstate.New()
	world.MapName = "geffen_in"
	ctx := client.Context{
		Resources: manager,
		World:     world,
	}
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, z: 0, zoom: 150}

	indoorProjection := camera.Projection(ctx, 800, 600, time.Now())
	if got := indoorProjection.CameraZoom; got != scene.CameraZoom() {
		t.Fatalf("indoor projection zoom = %.1f, want %.1f", got, scene.CameraZoom())
	}

	ctx.World.MapName = "prontera"
	outdoorProjection := camera.Projection(ctx, 800, 600, time.Now())
	if got := outdoorProjection.CameraZoom; got != 150 {
		t.Fatalf("restored outdoor projection zoom = %.1f, want 150.0", got)
	}
}

func TestFollowCameraTrackingResetKeepsUserView(t *testing.T) {
	camera := followCamera{
		initialized: true,
		x:           10.5,
		y:           20.5,
		z:           3,
		lastUpdate:  time.Now(),
		yawOffset:   73,
		pitch:       245,
		zoom:        148,
		zoomTarget:  152,
	}

	camera.ResetTracking()

	if camera.initialized || camera.x != 0 || camera.y != 0 || camera.z != 0 || !camera.lastUpdate.IsZero() {
		t.Fatalf("tracking state was not reset: %+v", camera)
	}
	if camera.yawOffset != 73 || camera.pitch != 245 || camera.zoom != 148 || camera.zoomTarget != 152 {
		t.Fatalf("user view = yaw %.1f pitch %.1f zoom %.1f target %.1f", camera.yawOffset, camera.pitch, camera.zoom, camera.zoomTarget)
	}
}

func TestFollowCameraProjectionIncludesRuntimeYawOffset(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	ctx := client.Context{World: world}
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, z: 0}

	camera.RotateImmediate(90)
	projection := camera.Projection(ctx, 800, 600, time.Now())
	if got := projection.CameraYaw; got != 90 {
		t.Fatalf("projection yaw = %.1f, want 90.0", got)
	}

	camera.ResetRotation()
	// Reset eases home like the rotate buttons; step time forward until
	// the settle completes.
	now := time.Now()
	for i := 0; i < 240; i++ {
		now = now.Add(16 * time.Millisecond)
		camera.Update(ctx, now)
	}
	projection = camera.Projection(ctx, 800, 600, now)
	if got := projection.CameraYaw; got != scene.DefaultSceneCameraYaw {
		t.Fatalf("reset projection yaw = %.1f, want %.1f", got, scene.DefaultSceneCameraYaw)
	}
}

func TestIndoorCameraYawIsLockedWithoutLosingOutdoorRotation(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "indoorrswtable.txt"), []byte("geffen_in.rsw#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := client.Context{
		Resources: manager,
		World:     &worldstate.World{MapName: "geffen_in"},
	}
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, z: 0}

	camera.RotateImmediate(90)
	camera.Tilt(30)
	projection := camera.Projection(ctx, 800, 600, time.Now())
	if got := projection.CameraYaw; got != -45 {
		t.Fatalf("indoor projection yaw = %.1f, want -45.0", got)
	}
	if got := projection.CameraPitch; got != scene.CameraPitch() {
		t.Fatalf("indoor projection pitch = %.1f, want fixed %.1f", got, scene.CameraPitch())
	}
	if camera.yawOffset != 90 {
		t.Fatalf("stored outdoor yaw offset = %.1f, want 90.0", camera.yawOffset)
	}
	if got := camera.currentPitch(); got != scene.DefaultCameraMaxPitch {
		t.Fatalf("stored outdoor pitch = %.1f, want %.1f", got, scene.DefaultCameraMaxPitch)
	}

	ctx.World.MapName = "prontera"
	projection = camera.Projection(ctx, 800, 600, time.Now())
	if got := projection.CameraYaw; got != 90 {
		t.Fatalf("restored outdoor projection yaw = %.1f, want 90.0", got)
	}
	if got := projection.CameraPitch; got != scene.DefaultCameraMaxPitch {
		t.Fatalf("restored outdoor projection pitch = %.1f, want %.1f", got, scene.DefaultCameraMaxPitch)
	}
}

func TestCameraRotationIsDisabledOnIndoorMap(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "indoorrswtable.txt"), []byte("geffen_in.rsw#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	inputState := input.NewState()
	inputState.SetMousePosition(100, 100)
	inputState.SetMouseButton(input.MouseButtonRight, true)
	inputState.SetKey(input.KeyShift, true)
	inputState.SetMousePosition(200, 160)
	mode := &WorldMode{}
	mode.camera.RotateImmediate(90)
	mode.camera.Tilt(15)
	ctx := client.Context{
		Resources: manager,
		World:     &worldstate.World{MapName: "geffen_in"},
		Input:     inputState,
		ScreenW:   800,
		ScreenH:   600,
	}

	mode.updateCameraRotation(ctx)
	if mode.camera.yawOffset != 90 {
		t.Fatalf("stored outdoor camera yaw offset = %.1f, want 90.0", mode.camera.yawOffset)
	}
	if got := mode.camera.currentPitch(); got != 245 {
		t.Fatalf("stored outdoor camera pitch = %.1f, want 245.0", got)
	}
}

func TestCameraDragYawDeltaMatchesRobrowserScale(t *testing.T) {
	if got := cameraDragYawDelta(100, 1000); got != -72 {
		t.Fatalf("drag yaw delta = %.1f, want -72.0", got)
	}
	if got := cameraDragYawDelta(-100, 1000); got != 72 {
		t.Fatalf("reverse drag yaw delta = %.1f, want 72.0", got)
	}
	if got := cameraDragYawDelta(100, 0); got != 0 {
		t.Fatalf("zero width drag yaw delta = %.1f, want 0", got)
	}
}

func TestCameraDragPitchDeltaMatchesReferenceClientScale(t *testing.T) {
	if got := cameraDragPitchDelta(60); got != 18 {
		t.Fatalf("drag pitch delta = %.1f, want 18.0", got)
	}
	if got := cameraDragPitchDelta(-60); got != -18 {
		t.Fatalf("reverse drag pitch delta = %.1f, want -18.0", got)
	}
}

func TestShiftRightDragTiltsCameraWithoutRotating(t *testing.T) {
	inputState := input.NewState()
	inputState.SetMousePosition(100, 100)
	inputState.SetMouseButton(input.MouseButtonRight, true)
	inputState.SetKey(input.KeyShift, true)
	inputState.SetMousePosition(200, 160)
	mode := &WorldMode{}
	mode.camera.RotateImmediate(45)
	ctx := client.Context{
		World:   &worldstate.World{MapName: "prontera"},
		Input:   inputState,
		ScreenW: 800,
		ScreenH: 600,
	}

	mode.updateCameraRotation(ctx)

	if got := mode.camera.currentPitch(); got != scene.DefaultCameraMaxPitch {
		t.Fatalf("camera pitch = %.1f, want %.1f", got, scene.DefaultCameraMaxPitch)
	}
	if mode.camera.yawOffset != 45 {
		t.Fatalf("camera yaw = %.1f, want unchanged 45.0", mode.camera.yawOffset)
	}
}

func TestCameraPitchIsClampedToReferenceClientOutdoorLimits(t *testing.T) {
	camera := followCamera{}
	camera.Tilt(-1000)
	if got := camera.currentPitch(); got != scene.DefaultCameraMinPitch {
		t.Fatalf("minimum camera pitch = %.1f, want %.1f", got, scene.DefaultCameraMinPitch)
	}
	camera.Tilt(1000)
	if got := camera.currentPitch(); got != scene.DefaultCameraMaxPitch {
		t.Fatalf("maximum camera pitch = %.1f, want %.1f", got, scene.DefaultCameraMaxPitch)
	}
}

func TestCameraWheelZoomDeltaMatchesRobrowserStep(t *testing.T) {
	if got := cameraWheelZoomDelta(1); got != -15 {
		t.Fatalf("wheel up delta = %.1f, want -15", got)
	}
	if got := cameraWheelZoomDelta(-2); got != 15 {
		t.Fatalf("wheel down delta = %.1f, want 15", got)
	}
	if got := cameraWheelZoomDelta(0.25); got != -15 {
		t.Fatalf("trackpad wheel delta = %.1f, want one notch", got)
	}
}

func TestCameraPinchZoomFactorZoomsInWhenFingersSpread(t *testing.T) {
	if got := cameraPinchZoomFactor(25); got >= 1 {
		t.Fatalf("pinch spread factor = %.3f, want zoom-in factor below 1", got)
	}
	if got := cameraPinchZoomFactor(-25); got <= 1 {
		t.Fatalf("pinch close factor = %.3f, want zoom-out factor above 1", got)
	}
}

func TestFollowCameraZoomIsClampedAndProjected(t *testing.T) {
	world := worldstate.New()
	ctx := client.Context{World: world}
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, z: 0}

	camera.ZoomBy(0.1)
	if got := camera.currentZoom(); got != scene.DefaultCameraMinZoom {
		t.Fatalf("zoom in clamp = %.1f, want %.1f", got, scene.DefaultCameraMinZoom)
	}
	camera.ZoomBy(10)
	if got := camera.currentZoom(); got != scene.DefaultCameraMaxZoom {
		t.Fatalf("zoom out clamp = %.1f, want %.1f", got, scene.DefaultCameraMaxZoom)
	}
	projection := camera.Projection(ctx, 800, 600, time.Now())
	if got := projection.CameraZoom; got != scene.DefaultCameraMaxZoom {
		t.Fatalf("projection zoom = %.1f, want %.1f", got, scene.DefaultCameraMaxZoom)
	}
}

func TestCursorRotateInfoMatchesRobrowser(t *testing.T) {
	info := cursorInfo(cursorActionRotate)
	if info.delayMult != 1 {
		t.Fatalf("rotate cursor info = %+v", info)
	}
}
