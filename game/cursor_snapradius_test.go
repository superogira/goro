package game

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/session"
)

func TestInputPickMultiplierSources(t *testing.T) {
	cases := []struct {
		name    string
		session float64
		config  float64
		want    float64
	}{
		{"session wins", 2, 1, 2},
		{"config fallback", 0, 1.5, 1.5},
		{"missing everywhere clamps up to 0.5", 0, 0, 0.5},
		{"clamped low", 0.1, 0, 0.5},
		{"clamped high", 9, 0, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := client.Context{
				Config:  config.Config{},
				Session: &session.Session{SnapRadius: tc.session},
			}
			ctx.Config.Gameplay.SnapRadius = tc.config
			if got := inputPickMultiplier(ctx); got != tc.want {
				t.Fatalf("inputPickMultiplier = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPickBoundsGrowWithMultiplier(t *testing.T) {
	// A touch 25px right of an item center misses the base pick box
	// (half-width 18) but lands once the radius multiplier doubles it.
	// Actors are wider (half-width 44), so their probe sits at 60px.
	const itemX = 125
	const actorX = 160
	if pointInGroundItemPickBounds(itemX, 100, 100, 100, 1) {
		t.Fatal("25px offset should miss the base item pick box")
	}
	if !pointInGroundItemPickBounds(itemX, 100, 100, 100, 1*2) {
		t.Fatal("25px offset should hit the doubled item pick box")
	}
	if pointInActorPickBounds(actorX, 100, 100, 100, 1) {
		t.Fatal("60px offset should miss the base actor pick box")
	}
	if !pointInActorPickBounds(actorX, 100, 100, 100, 1*2) {
		t.Fatal("60px offset should hit the doubled actor pick box")
	}
}
