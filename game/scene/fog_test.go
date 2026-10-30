package scene_test

import (
	"math"
	"testing"

	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
)

func TestSceneFogDepthAtTargetMatchesRobrowserDefaultZoom(t *testing.T) {
	projection := testutil.ProjectionYawZoom(1280, 720, 10.5, 20.5, 0, 0, scene.DefaultSceneCameraZoom)
	got := projection.FogDepth(10.5, 20.5, 0)
	const want = 1000 * ((scene.DefaultSceneCameraZoom * 0.5) - 1) / 999
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("target fog depth = %.3f, want %.3f", got, want)
	}
}
