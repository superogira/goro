package session_test

import (
	"testing"

	"github.com/kivutar/goro/session"
)

func TestSelectCharacterSeedsInventoryZeny(t *testing.T) {
	sessionState := &session.Session{}

	sessionState.SelectCharacter(session.Character{ID: 1234, Money: 95000})

	if sessionState.Inventory.Zeny != 95000 {
		t.Fatalf("zeny = %d, want 95000", sessionState.Inventory.Zeny)
	}
}

func TestSessionProgressFromCharacterUsesBaseLevel(t *testing.T) {
	progress := session.ProgressFromCharacter(session.Character{Level: 12, JobLevel: 7})
	if progress.BaseLevel != 12 || progress.JobLevel != 7 {
		t.Fatalf("progress = %+v", progress)
	}
}
