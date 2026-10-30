package battle

import (
	"math"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestApplyActorHPUpdateStoresExactLife(t *testing.T) {
	mode := &System{}
	mode.ApplyActorHPUpdate(network.ActorHPUpdate{ID: 300, HP: 12, MaxHP: 48})

	life, ok := mode.ActorLife[300]
	if !ok {
		t.Fatal("life missing")
	}
	if life.HP != 12 || life.MaxHP != 48 {
		t.Fatalf("life = %+v, want exact 12/48", life)
	}
}

func TestActorLifeForDisplayUsesLocalPlayerHPAndSP(t *testing.T) {
	mode := &System{}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
			Vitals: session.Vitals{
				HP:    75,
				MaxHP: 100,
				SP:    8,
				MaxSP: 20,
			},
		},
	}

	life, ok := mode.ActorLifeForDisplay(ctx, worldstate.Actor{ID: 150000})
	if !ok {
		t.Fatal("local player life missing")
	}
	if life.HP != 75 || life.MaxHP != 100 || life.SP != 8 || life.MaxSP != 20 || !life.HasSP || !life.Player {
		t.Fatalf("local player life = %+v", life)
	}
}

func TestActorLifeForDisplayHidesMonsterHPBarsFor2008Client(t *testing.T) {
	mode := &System{ActorLife: map[uint32]Life{
		300: {HP: 12, MaxHP: 48},
	}}
	ctx := client.Context{
		Session: &session.Session{CharID: 150000},
	}

	actor := worldstate.Actor{ID: 300, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	if _, ok := mode.ActorLifeForDisplay(ctx, actor); ok {
		t.Fatal("monster HP bar should be hidden for the 2008 client profile")
	}
	if life, ok := mode.monsterLifeForSense(300); !ok || life.HP != 12 || life.MaxHP != 48 {
		t.Fatalf("sense life cache = %+v ok=%v, want 12/48", life, ok)
	}
}

func TestActorOverlayBarFillWidthRoundsAndClamps(t *testing.T) {
	if got, want := actorOverlayBarFillWidth(0.333), math.Round((actorOverlayBarWidth-2)*0.333); got != want {
		t.Fatalf("fill width = %.1f, want %.1f", got, want)
	}
	if got := actorOverlayBarFillWidth(1.5); got != actorOverlayBarWidth-2 {
		t.Fatalf("clamped fill width = %.1f, want %.1f", got, actorOverlayBarWidth-2)
	}
	if got := actorOverlayBarFillWidth(math.NaN()); got != 0 {
		t.Fatalf("nan fill width = %.1f, want 0", got)
	}
}

func TestLocalPlayerNameIsBelowHPAndSPBars(t *testing.T) {
	life := Life{HasSP: true}
	barY := actors.ActorLifeBarY(100, 1.2)
	nameY := ActorNameBelowLifeBarY(100, 1.2, life)
	if nameY <= barY+actorLifeBarHeight(life) {
		t.Fatalf("name y = %.1f, bar y = %.1f; want name below hp/sp bars", nameY, barY)
	}
}

func TestActorLifeBarHeightAddsHomunculusHungerRow(t *testing.T) {
	if got := actorLifeBarHeight(Life{HasSP: true, HasHunger: true}); got != 13 {
		t.Fatalf("life bar height = %.1f, want hp/sp/hunger height", got)
	}
}

func TestHomunculusNameYUsesThreeBarHeight(t *testing.T) {
	life := Life{HasSP: true, HasHunger: true}
	barY := actors.ActorLifeBarY(100, 1.2)
	nameY := ActorNameBelowLifeBarY(100, 1.2, life)
	if got := nameY - (barY + actorLifeBarHeight(life)); got != 3 {
		t.Fatalf("name gap = %.1f, want 3px below hp/sp/hunger bars", got)
	}
}
