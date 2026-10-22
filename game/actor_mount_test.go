package game

import (
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestVisualJobForPecoRiding(t *testing.T) {
	if got := scene.VisualJobForEffectState(db.JobKnight, db.EffectStateRiding); got != db.JobKnight2 {
		t.Fatalf("visual job = %d, want mounted knight %d", got, db.JobKnight2)
	}
	if got := scene.VisualJobForEffectState(db.JobKnight, 0); got != db.JobKnight {
		t.Fatalf("unmounted visual job = %d, want base knight", got)
	}
}

func TestVisualJobForWeddingCostume(t *testing.T) {
	if got := scene.VisualJobForEffectState(db.JobWizard, db.EffectStateWedding); got != db.JobMarried {
		t.Fatalf("wedding visual job = %d, want married job %d", got, db.JobMarried)
	}
	if got := scene.VisualJobForEffectState(db.JobKnight, db.EffectStateRiding|db.EffectStateWedding); got != db.JobMarried {
		t.Fatalf("wedding riding visual job = %d, want married job %d", got, db.JobMarried)
	}
}

func TestLocalPlayerVisualCharacterKeepsSessionJobUnchanged(t *testing.T) {
	sessionState := &session.Session{
		Selected: session.Character{ID: 200, Job: db.JobKnight, Option: db.EffectStateRiding},
	}
	ctx := client.Context{
		Session: sessionState,
		World:   worldstate.New(),
	}

	visual := localPlayerVisualCharacter(ctx)
	if visual.Job != db.JobKnight2 {
		t.Fatalf("visual character job = %d, want mounted knight %d", visual.Job, db.JobKnight2)
	}
	if sessionState.Selected.Job != db.JobKnight {
		t.Fatalf("session job mutated to %d, want base knight", sessionState.Selected.Job)
	}
}

func TestCharacterWithVisualJobUsesWeddingOptionWithoutMutatingBaseJob(t *testing.T) {
	character := session.Character{ID: 200, Job: db.JobWizard, Option: db.EffectStateWedding}

	visual := characterWithVisualJob(character)
	if visual.Job != db.JobMarried {
		t.Fatalf("visual character job = %d, want married job %d", visual.Job, db.JobMarried)
	}
	if character.Job != db.JobWizard {
		t.Fatalf("source character job mutated to %d, want base wizard", character.Job)
	}
}

func TestAppendActorDrawEntryUsesMountedVisualJob(t *testing.T) {
	world := worldstate.New()
	actor := worldstate.Actor{
		ID:          300,
		Job:         db.JobKnight,
		EffectState: db.EffectStateRiding,
		X:           10,
		Y:           20,
	}
	projection := testutil.Projection(800, 600, 10.5, 20.5, 0)

	entries := appendActorDrawEntry(nil, world, projection, actor, false, time.Now(), 800, 600)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if got := entries[0].Actor.Job; got != db.JobKnight2 {
		t.Fatalf("entry visual job = %d, want mounted knight %d", got, db.JobKnight2)
	}
	if actor.Job != db.JobKnight {
		t.Fatalf("source actor job mutated to %d, want base knight", actor.Job)
	}
}

func TestAppendActorDrawEntryUsesWeddingVisualJob(t *testing.T) {
	world := worldstate.New()
	actor := worldstate.Actor{
		ID:          300,
		Job:         db.JobWizard,
		EffectState: db.EffectStateWedding,
		X:           10,
		Y:           20,
	}
	projection := testutil.Projection(800, 600, 10.5, 20.5, 0)

	entries := appendActorDrawEntry(nil, world, projection, actor, false, time.Now(), 800, 600)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if got := entries[0].Actor.Job; got != db.JobMarried {
		t.Fatalf("entry visual job = %d, want married job %d", got, db.JobMarried)
	}
	if actor.Job != db.JobWizard {
		t.Fatalf("source actor job mutated to %d, want base wizard", actor.Job)
	}
}
