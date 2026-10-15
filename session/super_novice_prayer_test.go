package session

import (
	"testing"
	"time"
)

func TestSuperNovicePrayerHistoryMatchesSakexe(t *testing.T) {
	const first, last = "Dear angel, can you hear my voice?", "Help me out~ Please~ T_T"
	for _, tt := range []struct {
		name   string
		warmup int
		lines  [4]string
		first  string
		last   string
		step   time.Duration
		wantAt int
	}{
		{"first fill", 7, [4]string{first, "anything", last, "thanks"}, first, last, 2 * time.Second, 2},
		{"rolling history", 10, [4]string{first, "anything", last, "thanks"}, first, last, 2 * time.Second, 3},
		{"no warmup", 0, [4]string{first, "anything", last, "thanks"}, first, last, 2 * time.Second, -1},
		{"initial history gap", 8, [4]string{first, "anything", last, "thanks"}, first, last, 2 * time.Second, -1},
		{"wrong first line", 10, [4]string{"wrong", first, last, "thanks"}, first, last, 2 * time.Second, -1},
		{"wrong last line", 10, [4]string{first, "anything", "wrong", "thanks"}, first, last, 2 * time.Second, -1},
		{"localized", 10, [4]string{"Cher ange", "anything", "Aide-moi", "merci"}, "Cher ange", "Aide-moi", 2 * time.Second, 3},
		{"missing text", 10, [4]string{first, "anything", last, "thanks"}, "", "", 2 * time.Second, -1},
		{"chat too fast", 7, [4]string{first, "anything", last, "thanks"}, first, last, time.Second, -1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var prayer SuperNovicePrayer
			now := time.Unix(100, 0)
			for range tt.warmup {
				prayer.Record("hello", now)
				now = now.Add(tt.step)
			}
			for i, line := range tt.lines {
				prayer.Record(line, now)
				if got := prayer.Matches(tt.first, tt.last, now); got != (i == tt.wantAt) {
					t.Fatalf("line %d: prayer matched=%t, want match only at %d", i, got, tt.wantAt)
				}
				now = now.Add(tt.step)
			}
		})
	}
}

func TestSuperNovicePrayerTenSecondBoundary(t *testing.T) {
	var prayer SuperNovicePrayer
	start := time.Unix(100, 0)
	for range 7 {
		prayer.Record("hello", start)
	}
	prayer.Record("first", start.Add(8*time.Second))
	prayer.Record("middle", start.Add(9*time.Second))
	prayer.Record("last", start.Add(10*time.Second))
	if !prayer.Matches("first", "last", start.Add(10*time.Second)) {
		t.Fatal("prayer was suppressed at exactly ten seconds")
	}
}
