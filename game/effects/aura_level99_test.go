package effects

import (
	"math"
	"testing"
	"time"
)

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
