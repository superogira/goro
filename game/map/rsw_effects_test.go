package maps

import (
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/res"
)

func TestRSWEffectWorldPositionUsesMapOffset(t *testing.T) {
	gnd := &res.GND{Width: 300, Height: 200}
	effect := res.RSWEffect{Position: res.RSWVector3{X: 12, Y: -3, Z: 34}}
	x, y, z := rswEffectWorldPosition(gnd, effect)
	if x != 312 || y != 234 || z != -2 {
		t.Fatalf("rswEffectWorldPosition = %.1f, %.1f, %.1f; want 312, 234, -2", x, y, z)
	}
}

func TestLoopingRSWEffectStartUsesDelayPhase(t *testing.T) {
	now := time.Unix(10, 250*int64(time.Millisecond))
	effect := res.RSWEffect{Delay: 100}
	start := loopingRSWEffectStart(effect, 0, 600*time.Millisecond, now)
	if !start.Equal(time.Unix(9, 700*int64(time.Millisecond))) {
		t.Fatalf("loopingRSWEffectStart = %s", start)
	}
}

func TestMapEffectComponentUsesWorldSizedSpriteWithoutChangingPlacement(t *testing.T) {
	component := effects.Component{
		Kind:       effects.Component3D,
		SpriteFile: "torch_01",
		PosX:       0.1,
		PosY:       -0.2,
		PosZ:       0.8,
	}
	got := MapEffectComponent(component)
	if !got.WorldSizedSprite {
		t.Fatalf("mapEffectComponent did not mark sprite as world sized: %+v", got)
	}
	if got.PosX != 0.1 || got.PosY != -0.2 || got.PosZ != 0.8 {
		t.Fatalf("mapEffectComponent = %+v", got)
	}
}

func TestSmokeEffectSpecMatchesReferenceMapEffect(t *testing.T) {
	spec, ok := effects.Lookup(effects.Smoke)
	if !ok {
		t.Fatal("smoke effect spec missing")
	}
	if spec.Duration != 10*time.Second || len(spec.Components) != 1 {
		t.Fatalf("smoke spec = %+v", spec)
	}
	component := spec.Components[0]
	if component.Kind != effects.Component3D || component.SpriteFile != "굴뚝연기" {
		t.Fatalf("smoke component sprite = %+v", component)
	}
	if component.Duplicate != 10 || component.DuplicateDelay != time.Second || component.Duration != 10*time.Second {
		t.Fatalf("smoke timing = duplicate %d delay %s duration %s", component.Duplicate, component.DuplicateDelay, component.Duration)
	}
	if component.PosZ != 0 || component.PosZEnd != 20 || component.PosXEndRand != 3 || !component.PosXSmooth {
		t.Fatalf("smoke placement = %+v", component)
	}
	if component.SizeStart != testutil.EffectTableSize(70) || component.SizeEnd != testutil.EffectTableSize(300) || !component.SizeSmooth {
		t.Fatalf("smoke size = %.3f %.3f smooth=%t", component.SizeStart, component.SizeEnd, component.SizeSmooth)
	}
	if component.AlphaMax != 0.8 || !component.FadeOut || !component.BlendAdditive || !component.Rotate || !component.RotateWithCamera {
		t.Fatalf("smoke alpha/rotation = %+v", component)
	}
}

func TestReferenceRSWMapEffectIDsHaveSpecs(t *testing.T) {
	for _, effectID := range []int{
		effects.Smoke,
		effects.Firefly,
		effects.Torch,
		effects.Bubble,
		effects.DragonSmoke,
		effects.Banjjakii,
		effects.MapPillar,
		effects.MapPillar2,
		effects.MapPillar3,
		effects.MapPillar4,
		effects.TorchRed,
		effects.TorchGreen,
		effects.TorchPurple,
		effects.MapGhost,
		effects.Glow1,
		effects.Glow2,
		effects.Glow4,
		effects.BubbleDrop,
		effects.Rainbow,
	} {
		if _, ok := effects.Lookup(effectID); !ok {
			t.Fatalf("missing RSW map effect spec %d", effectID)
		}
	}
}

func TestMapPillarEffectSpecsMatchReferenceVariants(t *testing.T) {
	type variant struct {
		name        string
		id          int
		texture     string
		radiusStart float64
		alpha       float64
		tint        color.RGBA
	}
	for _, tc := range []variant{
		{"EF_MAPPILLAR", effects.MapPillar, "ring_blue", 2, 50.0 / 255.0, color.RGBA{R: 110, G: 175, B: 255, A: 255}},
		{"EF_MAPPILLAR2", effects.MapPillar2, "ring_blue", 11, 70.0 / 255.0, color.RGBA{R: 110, G: 175, B: 255, A: 255}},
		{"EF_MAPPILLAR3", effects.MapPillar3, "magic_green", 2, 50.0 / 255.0, color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{"EF_MAPPILLAR4", effects.MapPillar4, "ring_red", 2, 50.0 / 255.0, color.RGBA{R: 255, G: 255, B: 255, A: 255}},
	} {
		spec, ok := effects.Lookup(tc.id)
		if !ok || spec.Duration != 16*time.Second || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != effects.ComponentFUNC || component.FuncName != "MapPillar" || component.FuncAdapter != effects.EffectFuncMapPillar || component.TextureName != tc.texture || component.Duration != 16*time.Second {
			t.Fatalf("%s component timing/texture = %+v", tc.name, component)
		}
		if component.AlphaMax != tc.alpha || component.Color != tc.tint || component.BlendMode != 2 || !component.BlendAdditive || component.AttachedEntity {
			t.Fatalf("%s component alpha/color/blend = %+v", tc.name, component)
		}
		if component.BottomSize != tc.radiusStart*0.2 || component.Height != 24 {
			t.Fatalf("%s component geometry = %+v", tc.name, component)
		}
	}
}

func TestMapPillarBandHeightMatchesReferenceLifecycle(t *testing.T) {
	maxHeight := 24.0
	if got := effects.MapPillarBandHeight(199, maxHeight); got != 0 {
		t.Fatalf("height before grow = %.3f, want 0", got)
	}
	if got := effects.MapPillarBandHeight(245, maxHeight); math.Abs(got-maxHeight*math.Sin(45*math.Pi/180)) > 0.0001 {
		t.Fatalf("height mid grow = %.3f", got)
	}
	if got := effects.MapPillarBandHeight(291, maxHeight); got != maxHeight {
		t.Fatalf("height after grow = %.3f, want %.3f", got, maxHeight)
	}
	if got := effects.MapPillarBandHeight(800, maxHeight); math.Abs(got-23.8) > 0.0001 {
		t.Fatalf("height shrink start = %.3f, want 23.8", got)
	}
	if got := effects.MapPillarBandHeight(919, maxHeight); got != 0 {
		t.Fatalf("height after shrink = %.3f, want 0", got)
	}
}
