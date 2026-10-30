package skills

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/social"
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
)

func TestTaekwonMissionAndRankingMessages(t *testing.T) {
	var console gameui.ChatConsole
	mode := &System{}
	mode.Bind(Bindings{Console: &console})
	var socialSystem social.System
	socialSystem.Bind(social.Bindings{Console: &console})
	mode.ApplyTaekwonMission(client.Context{}, network.TaekwonMission{MonsterName: "Spore", MonsterID: 1014, Progress: 37, Result: 1})
	messages := console.Messages()
	if len(messages) != 1 || messages[0].Text != "Taekwon mission: Spore (37%)" || messages[0].Color != taekwonAnnouncementColor {
		t.Fatalf("mission messages = %+v", messages)
	}

	entries := make([]network.FameRankingEntry, 10)
	entries[0] = network.FameRankingEntry{Name: "Kicker", Points: 123}
	socialSystem.ApplyFameRanking(client.Context{}, network.FameRanking{Kind: network.FameRankingTaekwon, Entries: entries})
	messages = console.Messages()
	if len(messages) != 12 {
		t.Fatalf("message count = %d, want mission + header + ten ranks", len(messages))
	}
	if messages[1].Text != "=========== Taekwon Rank ===========" || messages[2].Text != "[1] Kicker : 123 Points" || messages[11].Text != "[10] None : 0 Points" {
		t.Fatalf("ranking messages = %+v", messages[1:])
	}

	socialSystem.ApplyFamePointUpdate(client.Context{}, network.FamePointUpdate{Kind: network.FameRankingTaekwon, GainedPoints: 1, TotalPoints: 42})
	messages = console.Messages()
	if len(messages) != 13 || messages[12].Text != "[POINT] You have been rewarded with 1 Tae-Kwon Mission rank points. Your point total is 42." {
		t.Fatalf("taekwon point message = %+v", messages)
	}
}

func TestStarPlaceCancelDoesNotSendAgreement(t *testing.T) {
	var console gameui.ChatConsole
	var confirm gameui.ConfirmModal
	mode := &System{}
	mode.Bind(Bindings{Console: &console, StarPlaceConfirm: &confirm})
	ctx := client.Context{Network: network.NewClient(20080910, false), ScreenW: 800, ScreenH: 600}

	mode.ApplyStarPlaceRequest(ctx, network.StarPlace{Place: 2})
	confirm.Cancel(ctx)

	if confirm.IsOpen() {
		t.Fatal("star place confirmation stayed open after cancel")
	}
	if messages := console.Messages(); len(messages) != 0 {
		t.Fatalf("cancel attempted a network reply: %+v", messages)
	}
}
