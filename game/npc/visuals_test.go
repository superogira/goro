package npc

import (
	"testing"
	"time"

	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	worldstate "github.com/kivutar/goro/world"
)

func TestSpecialNPCVisualForActor(t *testing.T) {
	tests := []struct {
		name  string
		actor worldstate.Actor
		want  SpecialVisual
	}{
		{
			name:  "guild flag gr2 is normal actor model",
			actor: worldstate.Actor{Job: 722},
			want:  SpecialVisualNone,
		},
		{
			name:  "city flag a remains sprite",
			actor: worldstate.Actor{Job: 1912},
			want:  SpecialVisualNone,
		},
		{
			name:  "city flag b remains sprite",
			actor: worldstate.Actor{Job: 1913},
			want:  SpecialVisualNone,
		},
		{
			name:  "sprite flag remains sprite",
			actor: worldstate.Actor{Job: 973},
			want:  SpecialVisualNone,
		},
		{
			name:  "clear npc torch",
			actor: worldstate.Actor{Job: scene.ActorJobClearNPC, Name: "Bobbing Torch#7"},
			want:  SpecialVisualTorch,
		},
		{
			name:  "clear npc firewood",
			actor: worldstate.Actor{Job: scene.ActorJobClearNPC, Name: "Wet Firewood#moc2"},
			want:  SpecialVisualTorch,
		},
		{
			name:  "flame monster remains sprite",
			actor: worldstate.Actor{Job: 1869, HasObjectType: true, ObjectType: scene.ActorObjectTypeMob},
			want:  SpecialVisualNone,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SpecialVisualForActor(tt.actor); got != tt.want {
				t.Fatalf("SpecialVisualForActor(%+v) = %d, want %d", tt.actor, got, tt.want)
			}
		})
	}
}

func TestPersistentSpecialNPCWorldEffectKeepsActorCellAnchor(t *testing.T) {
	now := time.Unix(100, 0)
	entry := scene.ActorDrawEntry{
		Actor:  worldstate.Actor{ID: 700, X: 30, Y: 40, Job: scene.ActorJobClearNPC},
		WorldX: 30.5,
		WorldY: 40.5,
	}

	effect := persistentWorldEffectForActorEntry(effects.Torch, entry, now)

	if effect.ActorID != 700 || effect.EffectID != effects.Torch {
		t.Fatalf("effect identity = %+v", effect)
	}
	if effect.X != 30 || effect.Y != 40 {
		t.Fatalf("effect cell = %d,%d; want actor cell 30,40", effect.X, effect.Y)
	}
	if !effect.Starts.Equal(now) || !effect.Expires.Equal(now.Add(24*time.Hour)) || effect.Duration != 24*time.Hour {
		t.Fatalf("effect lifetime = starts %s expires %s duration %s", effect.Starts, effect.Expires, effect.Duration)
	}
}
