package game

import (
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/pet"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/input"
)

func TestActivePetSlotMachineBlocksMapPointer(t *testing.T) {
	mode := &WorldMode{petSystem: pet.System{SlotMachine: pet.SlotMachineState{Active: true}}}
	if !mode.mapPointerBlocked(client.Context{}) {
		t.Fatal("active pet slot machine left the map pointer available")
	}
}

func TestActivePetSlotMachineUsesClickCursor(t *testing.T) {
	mode := &WorldMode{petSystem: pet.System{SlotMachine: pet.SlotMachineState{Active: true}}}
	ctx := client.Context{Input: input.NewState()}

	if got := mode.cursorDesiredAction(ctx, scene.Projection{}, time.Now()); got != cursorActionClick {
		t.Fatalf("cursor action = %d, want click", got)
	}
}
