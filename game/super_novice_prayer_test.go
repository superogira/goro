package game

import (
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

func TestSuperNovicePrayerChatRouting(t *testing.T) {
	const first, last = "Cher ange", "Aide-moi"
	resources, err := res.NewManagerFS(fstest.MapFS{
		"data/msgstringtable.txt": &fstest.MapFile{Data: []byte(strings.Repeat("unused#\n", 790) + first + "#\nunused#\nunused#\n" + last + "#\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name      string
		job       int16
		dead      bool
		resources *res.Manager
		want      bool
		chatRoom  bool
	}{
		{"super novice", db.JobSuperNovice, false, resources, true, false},
		{"super baby", db.JobSuperNoviceB, false, resources, true, false},
		{"chat room", db.JobSuperNovice, false, resources, true, true},
		{"ordinary novice", db.JobNovice, false, resources, false, false},
		{"dead", db.JobSuperNovice, true, resources, false, false},
		{"no translations", db.JobSuperNovice, false, nil, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			netClient, server := newBotTestConnection(t, 20080910)
			s := session.New()
			s.Selected = session.Character{Name: "Novice", Job: tt.job}
			s.Dead = tt.dead
			ctx := client.Context{Session: s, Network: netClient, Resources: tt.resources}
			// No EXP or learned skills are supplied: eligibility is server-owned.
			now := time.Now().Add(-time.Minute)
			for range 10 {
				s.SuperNovicePrayer.Record("hello", now)
			}
			for _, line := range []string{first, "any middle line", last} {
				if err := client.SendChat(ctx, line); err != nil {
					t.Fatal(err)
				}
				readBotTestPackets(t, server, network.BuildGlobalChatPacketForClientDate("Novice", line, 20080910))
			}
			before := s.SuperNovicePrayer
			for _, text := range []string{"%party", "$guild", "/w Friend whisper"} {
				if err := client.SendChat(ctx, text); err != nil {
					t.Fatal(err)
				}
			}
			readBotTestPackets(t, server, network.BuildPartyMessagePacket("Novice : party"))
			readBotTestPackets(t, server, network.BuildGuildMessagePacket("Novice : guild"))
			readBotTestPackets(t, server, network.BuildWhisperPacket("Friend", "whisper"))
			if s.SuperNovicePrayer != before {
				t.Fatal("non-public chat changed prayer history")
			}
			if tt.chatRoom {
				NewWorldMode().handleChatRoomAction(ctx, gameui.ChatRoomWindowAction{Message: "thanks"})
			} else if err := client.SendChat(ctx, "thanks"); err != nil {
				t.Fatal(err)
			}
			readBotTestPackets(t, server, network.BuildGlobalChatPacketForClientDate("Novice", "thanks", 20080910))
			if tt.want {
				readBotTestPackets(t, server, []byte{0xED, 0x01})
			}
			// A subsequent packet also detects an unwanted or duplicate prayer.
			if err := client.SendChat(ctx, "done"); err != nil {
				t.Fatal(err)
			}
			readBotTestPackets(t, server, network.BuildGlobalChatPacketForClientDate("Novice", "done", 20080910))
			before = s.SuperNovicePrayer
			netClient.Close()
			if err := client.SendChat(ctx, first); err == nil {
				t.Fatal("disconnected chat succeeded")
			}
			if s.SuperNovicePrayer != before {
				t.Fatal("failed send changed prayer history")
			}
		})
	}
}

func TestSuperNovicePrayerResetsOnMapEntry(t *testing.T) {
	ctx := mapRecoveryTestContext(t)
	ctx.Session.SuperNovicePrayer.Record("previous map", time.Now())
	NewWorldMode().Enter(ctx)
	if ctx.Session.SuperNovicePrayer != (session.SuperNovicePrayer{}) {
		t.Fatal("prayer history leaked into the next map")
	}
}

func TestSuperNovicePrayerUsesServerFuryStatus(t *testing.T) {
	for _, job := range []int16{db.JobSuperNovice, db.JobSuperNoviceB} {
		ctx := testutil.MonkEffectContext()
		ctx.Session.Selected.Job = job
		ctx.World.Player.Job = job
		m := &WorldMode{}
		change := network.StatusEffectChange{ActorID: ctx.Session.AccountID, StatusID: db.StatusExplosionspirits, Active: true}
		m.applyStatusEffectChange(ctx, change)
		if !m.effects().HasActive(effects.Gumgang, ctx.Session.AccountID, time.Now().Add(time.Hour)) {
			t.Fatalf("job %d did not get persistent Fury lightning", job)
		}
		change.Active = false
		m.applyStatusEffectChange(ctx, change)
		if m.effects().HasActive(effects.Gumgang, ctx.Session.AccountID, time.Now()) {
			t.Fatalf("job %d kept Fury lightning after the server cleared it", job)
		}
	}
}
