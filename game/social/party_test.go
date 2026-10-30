package social

import (
	"reflect"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestPartyCreationFailurePreservesServerState(t *testing.T) {
	for _, party := range []session.Party{
		{RefuseInvites: true},
		{Name: "Existing", Members: []session.PartyMember{{AccountID: 123}}},
	} {
		for _, result := range []uint8{1, 2, 3} {
			s := &session.Session{Party: party}
			mode := newTestSystem()
			mode.HandlePartyCreateResult(client.Context{Session: s}, network.PartyCreateResult{Result: result})
			if !reflect.DeepEqual(s.Party, party) {
				t.Fatalf("result %d changed party state from %+v to %+v", result, party, s.Party)
			}
		}
	}
}

func TestPartyVitalsDoNotCreateMembers(t *testing.T) {
	s := &session.Session{}
	ctx := client.Context{Session: s}

	ApplyPartyMemberHP(ctx, network.PartyMemberHP{AccountID: 123, HP: 10, MaxHP: 20})
	ApplyPartyMemberPosition(ctx, network.PartyMemberPosition{AccountID: 123, X: 5, Y: 6})

	if len(s.Party.Members) != 0 {
		t.Fatalf("party vitals created members: %+v", s.Party.Members)
	}
}

func TestPartyVitalsUpdateKnownMembers(t *testing.T) {
	s := &session.Session{
		Party: session.Party{
			Name:    "party",
			Members: []session.PartyMember{{AccountID: 123, Name: "Alice"}},
		},
	}
	ctx := client.Context{Session: s}

	ApplyPartyMemberHP(ctx, network.PartyMemberHP{AccountID: 123, HP: 10, MaxHP: 20})
	ApplyPartyMemberPosition(ctx, network.PartyMemberPosition{AccountID: 123, X: 5, Y: 6})

	member := s.Party.Members[0]
	if member.HP != 10 || member.MaxHP != 20 || member.X != 5 || member.Y != 6 {
		t.Fatalf("party vitals not applied: %+v", member)
	}
}

func TestApplyPartyListSyncsLocalVitals(t *testing.T) {
	s := &session.Session{
		AccountID: 2000000,
		Selected:  session.Character{Name: "Kivutar"},
		Vitals:    session.Vitals{HP: 75, MaxHP: 100},
	}
	ctx := client.Context{Session: s}

	ApplyPartyList(ctx, network.PartyList{
		Name: "party",
		Members: []network.PartyMember{
			{AccountID: 2000000, Name: "Kivutar"},
			{AccountID: 3000000, Name: "Alice"},
		},
	})

	member := findPartyMember(&s.Party, 2000000)
	if member == nil || member.HP != 75 || member.MaxHP != 100 {
		t.Fatalf("local party member = %+v, want 75/100", member)
	}
}

func TestSyncLocalPartyVitalsFallsBackToSelectedCharacter(t *testing.T) {
	s := &session.Session{
		AccountID: 2000000,
		Selected:  session.Character{Name: "Kivutar", HP: 40, MaxHP: 55},
		Party:     session.Party{Name: "party"},
	}
	ctx := client.Context{Session: s}

	SyncLocalPartyVitals(ctx)

	member := findPartyMember(&s.Party, 2000000)
	if member == nil || member.HP != 40 || member.MaxHP != 55 {
		t.Fatalf("local party member = %+v, want selected character 40/55", member)
	}
}
