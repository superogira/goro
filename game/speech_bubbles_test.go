package game

import (
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestSpeechBubbleTextUsesMessagePart(t *testing.T) {
	if got := speechBubbleText("Kivutar : hello there"); got != "hello there" {
		t.Fatalf("speechBubbleText = %q, want hello there", got)
	}
}

func TestApplySpeechBubbleUsesLocalActorIDsForLocalEcho(t *testing.T) {
	mode := NewWorldMode()
	ctx := client.Context{Session: &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Selected:  session.Character{ID: 150000, Name: "Kivutar"},
	}}
	mode.applySpeechBubble(ctx, network.ChatMessage{Text: "Kivutar : hello"}, time.Now())
	if mode.speechBubbles[150000].text != "hello" {
		t.Fatalf("char bubble = %+v, want hello", mode.speechBubbles[150000])
	}
	if mode.speechBubbles[2000000].text != "hello" {
		t.Fatalf("account bubble = %+v, want hello", mode.speechBubbles[2000000])
	}
}

func TestPetTalkUsesPetNameAsSpeaker(t *testing.T) {
	mode := NewWorldMode()
	mode.petProperty = network.PetProperty{Name: "Sakurai"}
	mode.applyPetTalk(client.Context{}, 123, "foo", time.Now())
	if mode.speechBubbles[123].text != "foo" {
		t.Fatalf("pet bubble = %+v, want foo", mode.speechBubbles[123])
	}
}

func TestSkillCastNotifyShowsSkillNameBubble(t *testing.T) {
	mode := NewWorldMode()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
			Skills: session.Skills{List: []session.Skill{
				{ID: db.SkillMGFirebolt, Name: "Fire Bolt"},
			}},
		},
		World: world,
	}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{
		SourceID:  2000000,
		TargetID:  1100,
		SkillID:   db.SkillMGFirebolt,
		DelayTime: 1000,
	})

	if got := mode.speechBubbles[2000000].text; got != "Fire Bolt !!" {
		t.Fatalf("account skill bubble = %q, want Fire Bolt !!", got)
	}
	if got := mode.speechBubbles[150000].text; got != "Fire Bolt !!" {
		t.Fatalf("char skill bubble = %q, want Fire Bolt !!", got)
	}
}

func TestActorActionNotifyShowsSkillNameBubble(t *testing.T) {
	mode := NewWorldMode()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: 1}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
			Skills: session.Skills{List: []session.Skill{
				{ID: db.SkillSMBash, Name: "Bash"},
			}},
		},
		World: world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    2000000,
		TargetID:    1100,
		SourceSpeed: 500,
		TargetSpeed: 500,
		SkillID:     db.SkillSMBash,
		SkillLevel:  1,
		Action:      network.ActorActionSkill,
	})

	if got := mode.speechBubbles[150000].text; got != "Bash !!" {
		t.Fatalf("skill action bubble = %q, want Bash !!", got)
	}
}

func TestSkillNameBubbleSkipsMobsAndExcludedSkills(t *testing.T) {
	mode := NewWorldMode()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applySkillNameBubble(ctx, 300, db.SkillSMBash, time.Now())
	if len(mode.speechBubbles) != 0 {
		t.Fatalf("mob skill bubble = %+v, want none", mode.speechBubbles)
	}

	mode.applySkillNameBubble(ctx, 2000000, db.SkillTFHiding, time.Now())
	if len(mode.speechBubbles) != 0 {
		t.Fatalf("excluded skill bubble = %+v, want none", mode.speechBubbles)
	}
}
