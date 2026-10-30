package scene_test

import (
	"testing"
	"time"

	"github.com/kivutar/goro/game/scene"
	worldstate "github.com/kivutar/goro/world"
)

func TestPendingAttackReadyAtWaitsForWalkEnd(t *testing.T) {
	now := time.Unix(100, 0)
	player := worldstate.Actor{
		Moving:       true,
		MoveStarted:  now.Add(-100 * time.Millisecond),
		MoveDuration: 600 * time.Millisecond,
	}

	got := scene.PendingAttackReadyAt(player, now)
	want := now.Add(560 * time.Millisecond)
	if !got.Equal(want) {
		t.Fatalf("readyAt = %s, want %s", got.Sub(now), want.Sub(now))
	}
}

func TestAttackRetryDueUsesOpenMidgardInterval(t *testing.T) {
	now := time.Unix(100, 0)
	if !scene.AttackRetryDue(time.Time{}, now) {
		t.Fatal("zero last attack should be due")
	}
	if scene.AttackRetryDue(now.Add(-scene.AttackRetryInterval+time.Millisecond), now) {
		t.Fatal("attack should not retry before the interval")
	}
	if !scene.AttackRetryDue(now.Add(-scene.AttackRetryInterval), now) {
		t.Fatal("attack should retry at the interval")
	}
}
