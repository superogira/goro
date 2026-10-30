package scene_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestCameraYawForIndoorMapIsLocked(t *testing.T) {
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
	if got := scene.CameraYawForMap(ctx); got != -45 {
		t.Fatalf("indoor camera yaw = %.1f, want -45.0", got)
	}
	ctx.World.MapName = "prontera"
	if got := scene.CameraYawForMap(ctx); got != scene.DefaultSceneCameraYaw {
		t.Fatalf("outdoor camera yaw = %.1f, want %.1f", got, scene.DefaultSceneCameraYaw)
	}
}

func TestCameraYawForFixedViewPointMapIsLocked(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "viewpointtable.txt"), []byte("fixed_view.rsw#150#50#170#30#30#30#60#30#45#\nfree_view.rsw#150#50#170#-360#360#0#60#30#45#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := client.Context{
		Resources: manager,
		World:     &worldstate.World{MapName: "fixed_view"},
	}
	if got := scene.CameraYawForMap(ctx); got != -30 {
		t.Fatalf("fixed viewpoint yaw = %.1f, want -30.0", got)
	}
	if !scene.CameraRotationLockedForMap(ctx) {
		t.Fatal("fixed viewpoint should lock camera rotation")
	}
	ctx.World.MapName = "free_view"
	if got := scene.CameraYawForMap(ctx); got != scene.DefaultSceneCameraYaw {
		t.Fatalf("free viewpoint yaw = %.1f, want %.1f", got, scene.DefaultSceneCameraYaw)
	}
	if scene.CameraRotationLockedForMap(ctx) {
		t.Fatal("free viewpoint should not lock camera rotation")
	}
}

func TestCameraZoomRangeMatchesRobrowserOutdoorDefaults(t *testing.T) {
	if got := scene.CameraZoom(); got != 125 {
		t.Fatalf("default zoom = %.1f, want reference client default 125", got)
	}
	if scene.DefaultCameraMinZoom != 65 || scene.DefaultCameraMaxZoom != 165 {
		t.Fatalf("zoom range = %.1f..%.1f, want goro outdoor 65..165", scene.DefaultCameraMinZoom, scene.DefaultCameraMaxZoom)
	}
}
