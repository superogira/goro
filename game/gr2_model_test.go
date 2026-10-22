package game

import (
	"math"
	"testing"

	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/res"
)

func TestGR2ModelFacingYawMatchesClassicRODirections(t *testing.T) {
	tests := []struct {
		dir  int
		want scene.Point3
	}{
		{dir: 0, want: scene.Point3{Z: 1}},
		{dir: 2, want: scene.Point3{X: -1}},
		{dir: 4, want: scene.Point3{Z: -1}},
		{dir: 6, want: scene.Point3{X: 1}},
	}
	for _, tt := range tests {
		matrix := scene.Mat4RotateY(scene.Mat4Identity(), gr2ModelFacingYaw(tt.dir))
		front := scene.Mat4TransformVector(matrix, scene.Point3{Z: -1})
		if math.Abs(front.X-tt.want.X) > 1e-6 || math.Abs(front.Z-tt.want.Z) > 1e-6 {
			t.Fatalf("dir %d front = %+v, want %+v", tt.dir, front, tt.want)
		}
	}
}

func TestGR2ActorModelMatrixUsesCellAnchorAndPositiveHeight(t *testing.T) {
	matrix := gr2ActorModelMatrix(10.5, 20.5, 3, 0, 0.2)
	base := scene.Mat4TransformPoint(matrix, scene.Point3{})
	if math.Abs(base.X-10.5) > 1e-6 || math.Abs(base.Y-3) > 1e-6 || math.Abs(base.Z-20.5) > 1e-6 {
		t.Fatalf("base = %+v, want actor anchor 10.5, height 3, 20.5", base)
	}
	up := scene.Mat4TransformVector(matrix, scene.Point3{Z: 1})
	if up.Y <= 0 {
		t.Fatalf("local GR2 +Z should point upward in Goro, got %+v", up)
	}
}

func TestGR2ActorModelMatrixPreservesFacingWithPositiveHeight(t *testing.T) {
	tests := []struct {
		dir  int
		want scene.Point3
	}{
		{dir: 0, want: scene.Point3{Z: 1}},
		{dir: 2, want: scene.Point3{X: -1}},
		{dir: 4, want: scene.Point3{Z: -1}},
		{dir: 6, want: scene.Point3{X: 1}},
	}
	for _, tt := range tests {
		matrix := gr2ActorModelMatrix(0, 0, 0, tt.dir, 1)
		front := scene.Normalize3(scene.Mat4TransformVector(matrix, scene.Point3{Y: -1}))
		if math.Abs(front.X-tt.want.X) > 1e-6 || math.Abs(front.Z-tt.want.Z) > 1e-6 {
			t.Fatalf("dir %d front = %+v, want %+v", tt.dir, front, tt.want)
		}
	}
}

func TestGR2ActionForSpriteStateMatchesClassicStates(t *testing.T) {
	tests := []struct {
		name  string
		state sprites.State
		want  res.GR2Action
	}{
		{name: "stand", state: sprites.State{ActionFamily: scene.SpriteActionIdle}, want: res.GR2ActionStand},
		{name: "move", state: sprites.State{ActionFamily: scene.SpriteActionWalk}, want: res.GR2ActionMove},
		{name: "attack", state: sprites.State{ActionFamily: sprites.ActionNonPCAttack}, want: res.GR2ActionAttack},
		{name: "skill", state: sprites.State{ActionFamily: sprites.ActionPCSkill}, want: res.GR2ActionAttack},
		{name: "hurt", state: sprites.State{ActionFamily: sprites.ActionNonPCHurt}, want: res.GR2ActionDamage},
		{name: "dead", state: sprites.State{ActionFamily: sprites.ActionNonPCDeath}, want: res.GR2ActionDead},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gr2ActionForSpriteState(tt.state); got != tt.want {
				t.Fatalf("action = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLoadGR2ModelViewRealData(t *testing.T) {
	manager := testutil.RealDataManager(t)
	for _, name := range []string{"Empelium90_0.gr2", "Guildflag90_1.gr2", "Kguardian90_7.gr2"} {
		t.Run(name, func(t *testing.T) {
			view, status := loadGR2ModelView(manager, name)
			if view == nil {
				t.Fatalf("loadGR2ModelView failed: %s", status)
			}
			if view.geometry == nil || len(view.geometry.Vertices) == 0 || len(view.geometry.Batches) == 0 {
				t.Fatalf("empty geometry: %+v status=%s", view.geometry, status)
			}
			loadedTextures := 0
			for _, texture := range view.textures {
				if texture != nil {
					loadedTextures++
				}
			}
			if loadedTextures == 0 {
				t.Fatalf("no textures loaded: %s", status)
			}
			if view.pose == nil || len(view.bindPalette) != view.pose.boneCount() {
				t.Fatalf("pose/palette not loaded: bones=%d palette=%d status=%s", view.pose.boneCount(), len(view.bindPalette), status)
			}
			if name == "Guildflag90_1.gr2" && len(view.emblemTextures) == 0 {
				t.Fatalf("guild flag emblem texture slot not detected: %s", status)
			}
		})
	}
}

func TestKGuardianGR2AnimationClipsRealData(t *testing.T) {
	manager := testutil.RealDataManager(t)
	view, status := loadGR2ModelView(manager, "Kguardian90_7.gr2")
	if view == nil {
		t.Fatalf("loadGR2ModelView failed: %s", status)
	}
	for _, action := range []res.GR2Action{res.GR2ActionStand, res.GR2ActionMove, res.GR2ActionAttack, res.GR2ActionDead, res.GR2ActionDamage} {
		clip := view.clip(action)
		if clip == nil {
			t.Fatalf("missing clip action=%d status=%s", action, status)
		}
		if clip.duration <= 0 {
			t.Fatalf("clip action=%d duration=%f, want positive", action, clip.duration)
		}
		palette := view.skinningPalette(action, clip.duration/2)
		if len(palette) != view.pose.boneCount() {
			t.Fatalf("palette action=%d len=%d, want bones=%d", action, len(palette), view.pose.boneCount())
		}
	}
}
