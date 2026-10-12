package game

import (
	"reflect"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

func TestPartyCreationWaitsForServerMembership(t *testing.T) {
	for _, command := range []bool{false, true} {
		name := "window"
		if command {
			name = "organize command"
		}
		t.Run(name, func(t *testing.T) {
			netClient, conn := newBotTestConnection(t, 20080910)
			s := &session.Session{AccountID: 123, Party: session.Party{RefuseInvites: true}}
			ctx := client.Context{Session: s, Network: netClient}
			mode := &WorldMode{}
			request := func() {
				t.Helper()
				if command {
					mode.ui.console.SendText(ctx, `/organize "Friends"`)
					readBotTestPackets(t, conn, network.BuildMakePartyPacket("Friends"))
				} else {
					mode.createPartyFromWindow(ctx, gameui.PartyCreateWindowAction{
						Name: "Friends", ItemPickup: 1, ItemDivision: 1,
					})
					readBotTestPackets(t, conn, network.BuildMakeParty2Packet("Friends", 1, 1))
				}
				if s.Party.Active() {
					t.Fatalf("request created unconfirmed party: %+v", s.Party)
				}
			}
			request()
			// Basic Skill below level 7 is rejected through ZC_ACK_TOUSESKILL,
			// without a party creation response.
			mode.applySkillFailAck(ctx, network.SkillFailAck{SkillID: 1, Number: 4})
			if s.Party.Active() {
				t.Fatalf("rejected party is active: %+v", s.Party)
			}
			if messages := mode.ui.console.Messages(); len(messages) != 1 || messages[0].Text != "Cannot form a party." {
				t.Fatalf("rejection messages = %+v", messages)
			}
			request() // A refused request must not block another attempt.
			mode.handlePartyCreateResult(ctx, network.PartyCreateResult{Result: 0})
			applyPartyList(ctx, network.PartyList{
				Name: "Friends", Members: []network.PartyMember{{AccountID: 123, Name: "Alice"}},
			})
			if !s.Party.Active() || s.Party.Name != "Friends" || len(s.Party.Members) != 1 || s.Party.Members[0].AccountID != 123 {
				t.Fatalf("confirmed party = %+v", s.Party)
			}
			if !s.Party.RefuseInvites {
				t.Fatal("creation changed the invitation preference")
			}
		})
	}
}

func TestPartyCreationFailurePreservesServerState(t *testing.T) {
	for _, party := range []session.Party{
		{RefuseInvites: true},
		{Name: "Existing", Members: []session.PartyMember{{AccountID: 123}}},
	} {
		for _, result := range []uint8{1, 2, 3} {
			s := &session.Session{Party: party}
			mode := &WorldMode{}
			mode.handlePartyCreateResult(client.Context{Session: s}, network.PartyCreateResult{Result: result})
			if !reflect.DeepEqual(s.Party, party) {
				t.Fatalf("result %d changed party state from %+v to %+v", result, party, s.Party)
			}
		}
	}
}

func TestPartyVitalsDoNotCreateMembers(t *testing.T) {
	s := &session.Session{}
	ctx := client.Context{Session: s}

	applyPartyMemberHP(ctx, network.PartyMemberHP{AccountID: 123, HP: 10, MaxHP: 20})
	applyPartyMemberPosition(ctx, network.PartyMemberPosition{AccountID: 123, X: 5, Y: 6})

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

	applyPartyMemberHP(ctx, network.PartyMemberHP{AccountID: 123, HP: 10, MaxHP: 20})
	applyPartyMemberPosition(ctx, network.PartyMemberPosition{AccountID: 123, X: 5, Y: 6})

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

	applyPartyList(ctx, network.PartyList{
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

	syncLocalPartyVitals(ctx)

	member := findPartyMember(&s.Party, 2000000)
	if member == nil || member.HP != 40 || member.MaxHP != 55 {
		t.Fatalf("local party member = %+v, want selected character 40/55", member)
	}
}
