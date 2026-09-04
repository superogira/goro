package ui

import (
	"testing"
	"time"
)

func TestPoptipsDeduplicateCapAndFade(t *testing.T) {
	now := time.Unix(100, 0)
	var tips Poptips
	tips.Show("First", now)
	tips.Show("First", now.Add(time.Second))
	if len(tips.items) != 1 || tips.items[0].shownAt != now.Add(time.Second) {
		t.Fatalf("deduplicated poptips = %+v", tips.items)
	}
	tips.Show("Second", now.Add(2*time.Second))
	tips.Show("Third", now.Add(3*time.Second))
	if len(tips.items) != poptipMaxItems || tips.items[0].text != "Third" || tips.items[1].text != "Second" {
		t.Fatalf("capped poptips = %+v", tips.items)
	}
	if got := poptipAlpha(3*time.Second + 500*time.Millisecond); got != 0.5 {
		t.Fatalf("fade alpha = %v, want 0.5", got)
	}
	if got := poptipAlpha(4 * time.Second); got != 0 {
		t.Fatalf("expired alpha = %v, want 0", got)
	}
}
