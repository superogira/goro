package game

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestAdoptionRequestPacketOpensConfirmation(t *testing.T) {
	data := make([]byte, 34)
	binary.LittleEndian.PutUint16(data[0:2], network.PacketZCReqBaby)
	binary.LittleEndian.PutUint32(data[2:6], 2000001)
	binary.LittleEndian.PutUint32(data[6:10], 2000000)
	copy(data[10:34], "Zambla")

	mode := &WorldMode{}
	ctx := client.Context{ScreenW: 800, ScreenH: 600}
	mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCReqBaby, Data: data}, time.Now())

	if !mode.ui.adoptionRequest.IsOpen() {
		t.Fatal("adoption request did not open a confirmation")
	}
}

func TestAdoptionMenuRequiresEligibleParentAndPartyChild(t *testing.T) {
	target := worldstate.Actor{ID: 2000002, Name: "Baby", Job: db.JobNovice}
	eligible := &session.Session{
		Selected: session.Character{ID: 150004, Job: db.JobMerchant},
		Progress: session.Progress{BaseLevel: 70},
		Skills: session.Skills{List: []session.Skill{{
			ID:    db.SkillWECallpartner,
			Level: 1,
		}}},
		Party: session.Party{Members: []session.PartyMember{{AccountID: target.ID, Name: target.Name}}},
	}
	if !canRequestAdoption(eligible, target) {
		t.Fatal("eligible married parent could not request adoption")
	}

	for _, tc := range []struct {
		name   string
		mutate func(*session.Session, *worldstate.Actor)
	}{
		{"under level 70", func(s *session.Session, _ *worldstate.Actor) { s.Progress.BaseLevel = 69 }},
		{"not married", func(s *session.Session, _ *worldstate.Actor) { s.Skills.List = nil }},
		{"already a parent", func(s *session.Session, _ *worldstate.Actor) {
			s.Skills.List = append(s.Skills.List, session.Skill{ID: db.SkillWECallbaby, Level: 1})
		}},
		{"prospective baby", func(s *session.Session, _ *worldstate.Actor) { s.Selected.Job = db.JobNoviceB }},
		{"outside party", func(s *session.Session, _ *worldstate.Actor) { s.Party.Members = nil }},
		{"ineligible target job", func(_ *session.Session, target *worldstate.Actor) { target.Job = db.JobKnight }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := *eligible
			s.Skills.List = append([]session.Skill(nil), eligible.Skills.List...)
			s.Party.Members = append([]session.PartyMember(nil), eligible.Party.Members...)
			candidate := target
			tc.mutate(&s, &candidate)
			if canRequestAdoption(&s, candidate) {
				t.Fatal("adoption menu remained available")
			}
		})
	}
}

func TestBabyPlayerBodyScale(t *testing.T) {
	if got := playerBodyScaleForJob(db.JobNovice); got != 1 {
		t.Fatalf("adult scale = %v, want 1", got)
	}
	if got := playerBodyScaleForJob(db.JobNoviceB); got != scene.BabyPlayerBodyScale {
		t.Fatalf("baby scale = %v, want %v", got, scene.BabyPlayerBodyScale)
	}
}

func TestRemoteBabyDrawEntryUsesBabyScale(t *testing.T) {
	world := worldstate.New()
	projection := testutil.Projection(800, 600, 10.5, 20.5, 0)
	baseActor := worldstate.Actor{
		ID:         300,
		Job:        db.JobNovice,
		X:          10,
		Y:          20,
		Appearance: true,
	}
	adult := appendActorDrawEntry(nil, world, projection, baseActor, false, time.Now(), 800, 600)
	baseActor.Job = db.JobNoviceB
	baby := appendActorDrawEntry(nil, world, projection, baseActor, false, time.Now(), 800, 600)
	if len(adult) != 1 || len(baby) != 1 {
		t.Fatalf("draw entries adult=%d baby=%d, want one each", len(adult), len(baby))
	}
	if got, want := baby[0].Scale, adult[0].Scale*scene.BabyPlayerBodyScale; got != want {
		t.Fatalf("remote baby scale = %v, want %v", got, want)
	}
}

func TestBodySizeEffectUsesAbsoluteScaleForBaby(t *testing.T) {
	starts := time.Unix(10, 0)
	mode := WorldMode{effectSystem: effects.System{Active: []effects.Instance{{
		EffectID: effects.BabyBody,
		ActorID:  42,
		Starts:   starts,
		Expires:  starts.Add(300 * time.Millisecond),
		Duration: 300 * time.Millisecond,
	}}}}
	ctx := client.Context{}
	actor := worldstate.Actor{ID: 42, Job: db.JobNoviceB}
	baseScale := 10.0 * scene.BabyPlayerBodyScale

	for _, tc := range []struct {
		name string
		at   time.Duration
		want float64
	}{
		{"start", 0, 8},
		{"middle", 150 * time.Millisecond, 6.5},
		{"end", 300 * time.Millisecond, 5},
	} {
		if got := mode.playerRenderScale(ctx, actor, baseScale, starts.Add(tc.at)); got != tc.want {
			t.Fatalf("baby effect scale at %s = %v, want %v", tc.name, got, tc.want)
		}
	}
}
