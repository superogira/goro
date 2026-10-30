package social

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
)

func TestFameRankingMessages(t *testing.T) {
	tests := []struct {
		kind            network.FameRankingKind
		profession      string
		pointProfession string
		professionID    int
		pointID         int
		pointPrefix     string
	}{
		{network.FameRankingBlacksmith, "BlackSmith", "Blacksmith", 2386, 901, "[Point]"},
		{network.FameRankingAlchemist, "Alchemist", "Alchemist", 2387, 902, "[Point]"},
		{network.FameRankingTaekwon, "Taekwon", "Tae-Kwon Mission", 2388, 926, "[POINT]"},
	}
	for _, test := range tests {
		presentation := fameRankingPresentationFor(test.kind)
		if presentation.professionMessageID != test.professionID || presentation.pointMessageID != test.pointID {
			t.Fatalf("%s message IDs=(%d, %d) want=(%d, %d)", test.profession, presentation.professionMessageID, presentation.pointMessageID, test.professionID, test.pointID)
		}

		mode := newTestSystem()
		entries := make([]network.FameRankingEntry, 10)
		entries[0] = network.FameRankingEntry{Name: "Crafter", Points: 321}
		mode.ApplyFameRanking(client.Context{}, network.FameRanking{Kind: test.kind, Entries: entries})
		mode.ApplyFamePointUpdate(client.Context{}, network.FamePointUpdate{Kind: test.kind, GainedPoints: 25, TotalPoints: 346})

		messages := mode.bindings.Console.Messages()
		if len(messages) != 12 {
			t.Fatalf("%s message count=%d want=12", test.profession, len(messages))
		}
		if messages[0].Text != "=========== "+test.profession+" Rank ===========" || messages[1].Text != "[1] Crafter : 321 Points" {
			t.Fatalf("%s ranking messages=%+v", test.profession, messages[:2])
		}
		wantPoint := test.pointPrefix + " You have been rewarded with 25 " + test.pointProfession + " rank points. Your point total is 346."
		if messages[11].Text != wantPoint {
			t.Fatalf("%s fame update=%+v", test.profession, messages[11])
		}
	}
}
