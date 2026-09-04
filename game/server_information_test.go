package game

import (
	"testing"
	"time"
)

func TestSkillMessageText(t *testing.T) {
	if got, ok := skillMessageText(0x1f); !ok || got != "ATK +100%." {
		t.Fatalf("skill message = %q, %t", got, ok)
	}
	if _, ok := skillMessageText(0x1a); ok {
		t.Fatal("unknown skill message unexpectedly resolved")
	}
}

func TestBossRespawnMessage(t *testing.T) {
	if got := bossRespawnMessage("Baphomet", time.Hour+5*time.Minute, 0); got != "Baphomet will respawn in 1 hour(s) and 5 minute(s)." {
		t.Fatalf("single respawn time = %q", got)
	}
	if got := bossRespawnMessage("Baphomet", 5*time.Minute, 10*time.Minute); got != "Baphomet will respawn between 5 minute(s) and 10 minute(s)." {
		t.Fatalf("respawn range = %q", got)
	}
	if got := bossRespawnMessage("", 0, 0); got != "The boss will respawn shortly." {
		t.Fatalf("immediate respawn = %q", got)
	}
}
