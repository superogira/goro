package effects

import (
	"testing"
	"time"

	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/network"
)

func TestDevotionSnapshotAndRemoval(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	m := &System{}
	m.AddAt(ctx, Devotion, 2000000, time.Now())
	notify := network.ActorDevotion{SourceID: 2000000, Targets: [5]uint32{300, 300}}
	m.ApplyActorDevotion(ctx, notify)
	m.ApplyActorDevotion(ctx, notify)
	if len(m.Active) != 2 || !m.Active[1].Expires.IsZero() || m.Active[1].TargetID != 300 {
		t.Fatalf("snapshots must replace links without duplicating them: %+v", m.Active)
	}
	m.RemoveActor(300)
	if len(m.Active) != 1 || m.Active[0].EffectID != Devotion {
		t.Fatal("vanishing target left a link or removed the cast animation")
	}
	m.ApplyActorDevotion(ctx, notify)
	m.ApplyActorDevotion(ctx, network.ActorDevotion{SourceID: 2000000})
	if len(m.Active) != 1 {
		t.Fatal("empty snapshot did not clear the caster's links")
	}
	m.ApplyActorDevotion(ctx, notify)
	m.RemoveActor(2000000)
	if len(m.Active) != 0 {
		t.Fatal("vanishing caster left Devotion effects")
	}
}
