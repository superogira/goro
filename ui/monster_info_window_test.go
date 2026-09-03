package ui

import "testing"

func TestMonsterInfoLabels(t *testing.T) {
	if got := monsterSizeName(1); got != "Medium" {
		t.Fatalf("size = %q", got)
	}
	if got := monsterRaceName(7); got != "Demi-human" {
		t.Fatalf("race = %q", got)
	}
	if got := monsterPropertyName(8); got != "Ghost" {
		t.Fatalf("property = %q", got)
	}
	if got := monsterPropertyName(99); got != "Unknown (99)" {
		t.Fatalf("unknown property = %q", got)
	}
}

func TestMonsterHPTextUsesCachedMaximumWhenAvailable(t *testing.T) {
	if got := monsterHPText(45, 100); got != "45 / 100" {
		t.Fatalf("HP with maximum = %q", got)
	}
	if got := monsterHPText(45, 0); got != "45" {
		t.Fatalf("HP without maximum = %q", got)
	}
	if got := monsterHPText(120, 100); got != "120" {
		t.Fatalf("HP with stale maximum = %q", got)
	}
}
