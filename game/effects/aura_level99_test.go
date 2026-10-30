package effects

import (
	"math"
	"testing"
	"time"

	"github.com/kivutar/goro/game/internal/testutil"
	worldstate "github.com/kivutar/goro/world"
)

func TestAuraPreferenceUpdatesVisibleActorsWithoutChangingOtherEffects(t *testing.T) {
	ctx := testutil.Level99AuraTestContext(99, false)
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 12, Y: 20, Level: 99, HasLevel: true}
	now := time.Unix(10, 0)
	m := &System{}
	m.AddCellLifetime(ctx, Heal, 300, 12, 20, now, time.Minute, false)
	for _, step := range []struct {
		simple, less bool
		wantFull     bool
	}{
		{false, false, true},
		{true, false, false},
		{false, false, true},
		{false, true, false},
		{true, true, false},
	} {
		ctx.Session.SimpleAura = step.simple
		ctx.Session.LessEffects = step.less
		m.SyncLevel99AuraEffects(ctx, now)
		for _, actorID := range []uint32{2000000, 300} {
			for _, effectID := range Level99AuraNormalEffects {
				want := step.wantFull || effectID == Level99Bubble
				if got := m.HasActive(effectID, actorID, now); got != want {
					t.Fatalf("simple=%t less=%t actor=%d effect=%d: active=%t, want %t", step.simple, step.less, actorID, effectID, got, want)
				}
			}
		}
		if !m.HasActive(Heal, 300, now) {
			t.Fatal("changing aura preference removed a skill effect")
		}
	}
}

func TestLevel99BubbleSampleStartsHiddenThenRisesFromFeet(t *testing.T) {
	effect := Instance{EffectID: Level99Bubble, ActorID: 2000000, Starts: time.Unix(10, 0)}

	if sample := level99BubbleSample(effect, 0, 0, 0); sample.visible {
		t.Fatalf("fresh bubble sample is visible: %+v", sample)
	}

	firstVisibleFrame := -1.0
	for frame := 0.0; frame < 900; frame++ {
		if level99BubbleSample(effect, 0, 0, frame).visible {
			firstVisibleFrame = frame
			break
		}
	}
	if firstVisibleFrame < 0 {
		t.Fatal("bubble never reached its visible phase")
	}

	born := level99BubbleSample(effect, 0, 0, firstVisibleFrame)
	later := level99BubbleSample(effect, 0, 0, firstVisibleFrame+80)
	if !born.visible || !later.visible {
		t.Fatalf("visible samples missing: born=%+v later=%+v", born, later)
	}
	if born.height > level99BubbleRiseSpeed*level99BubbleGameToWorld+1e-9 {
		t.Fatalf("bubble starts at height %.4f, want near feet", born.height)
	}
	if later.height <= born.height {
		t.Fatalf("bubble height = %.4f after %.4f, want rising motion", later.height, born.height)
	}
	if math.Hypot(later.offsetX, later.offsetY) <= math.Hypot(born.offsetX, born.offsetY) {
		t.Fatalf("bubble drift = %.4f after %.4f, want wind spread while rising", math.Hypot(later.offsetX, later.offsetY), math.Hypot(born.offsetX, born.offsetY))
	}
}
