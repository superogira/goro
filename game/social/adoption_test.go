package social

import (
	"testing"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

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
	if !CanRequestAdoption(eligible, target) {
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
			if CanRequestAdoption(&s, candidate) {
				t.Fatal("adoption menu remained available")
			}
		})
	}
}
