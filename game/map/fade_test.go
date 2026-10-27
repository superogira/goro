package maps

import (
	"testing"
	"time"

	"github.com/kivutar/goro/network"
)

func TestMapFadeAlphaTransitionsThroughBlack(t *testing.T) {
	start := time.Unix(100, 0)
	mode := &Fade{}
	mode.StartOut(network.MapChange{MapName: "geffen"}, start)

	if got := mode.Alpha(start); got != 0 {
		t.Fatalf("fade-out start alpha = %d, want 0", got)
	}
	if got := mode.Alpha(start.Add(FadeOutDuration)); got != 255 {
		t.Fatalf("fade-out end alpha = %d, want 255", got)
	}

	*mode = Fade{Phase: FadeHold, Started: start}
	if got := mode.Alpha(start.Add(time.Second)); got != 255 {
		t.Fatalf("hold alpha = %d, want 255", got)
	}
	mode.StartPrewarm()
	if got := mode.Alpha(start.Add(time.Second)); got != 255 {
		t.Fatalf("prewarm alpha = %d, want 255", got)
	}

	mode.StartIn(start)
	if got := mode.Alpha(start); got != 255 {
		t.Fatalf("fade-in start alpha = %d, want 255", got)
	}
	if got := mode.Alpha(start.Add(FadeInDuration)); got != 0 {
		t.Fatalf("fade-in end alpha = %d, want 0", got)
	}
}

func TestMapPrewarmCompletesBeforeFadeInStarts(t *testing.T) {
	start := time.Unix(200, 0)
	mode := &Fade{}
	mode.StartPrewarm()

	mode.AdvancePrewarm(start, false)
	if mode.Phase != FadePrewarm {
		t.Fatalf("fade phase without a rendered frame = %d, want prewarm", mode.Phase)
	}

	mode.RecordCoveredFrame()
	mode.AdvancePrewarm(start, false)
	if mode.Phase != FadePrewarm {
		t.Fatalf("fade phase after one rendered frame = %d, want prewarm", mode.Phase)
	}

	mode.RecordCoveredFrame()
	mode.AdvancePrewarm(start, false)
	if mode.Phase != FadeIn {
		t.Fatalf("fade phase after prewarming = %d, want fade-in", mode.Phase)
	}
	if !mode.Started.Equal(start) {
		t.Fatalf("fade-in started = %s, want %s", mode.Started, start)
	}
}
