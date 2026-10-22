package effects

import (
	"testing"

	"github.com/kivutar/goro/game/scene"
)

func TestBardPangVoiceFacing(t *testing.T) {
	for _, tc := range []struct {
		dir    scene.Point3
		yaw    float64
		action int
	}{
		{scene.Point3{X: 1}, 0, 2}, {scene.Point3{Z: 1}, 0, 1},
		{scene.Point3{X: -1}, 0, 0}, {scene.Point3{Z: -1}, 0, 3},
		{scene.Point3{X: 1}, 90, 3}, {scene.Point3{X: 1}, -90, 1},
	} {
		if got := directedSpriteAction(tc.dir, tc.yaw); got != tc.action {
			t.Fatalf("direction %+v yaw %v: action %d, want %d", tc.dir, tc.yaw, got, tc.action)
		}
	}
}
