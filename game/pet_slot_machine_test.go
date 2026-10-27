package game

import (
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/res"
)

func TestActivePetSlotMachineBlocksMapPointer(t *testing.T) {
	mode := &WorldMode{petSlotMachine: petSlotMachineState{active: true}}
	if !mode.mapPointerBlocked(client.Context{}) {
		t.Fatal("active pet slot machine left the map pointer available")
	}
}

func TestActivePetSlotMachineUsesClickCursor(t *testing.T) {
	mode := &WorldMode{petSlotMachine: petSlotMachineState{active: true}}
	ctx := client.Context{Input: input.NewState()}

	if got := mode.cursorDesiredAction(ctx, scene.Projection{}, time.Now()); got != cursorActionClick {
		t.Fatalf("cursor action = %d, want click", got)
	}
}

func TestPetSlotMachineKeepsResultVisibleDuringPause(t *testing.T) {
	started := time.Unix(10, 0)
	act := &res.ACT{Actions: make([]res.ACTAction, int(petSlotMachineFail)+1)}
	act.Actions[petSlotMachineSuccess] = res.ACTAction{
		DelayMS:    100,
		Animations: []res.ACTAnimation{{}},
	}
	mode := &WorldMode{petSlotMachine: petSlotMachineState{
		active:  true,
		phase:   petSlotMachineSuccess,
		started: started,
	}}

	action, motion, visible := mode.petSlotMachineFrame(act, started.Add(200*time.Millisecond))
	if action != int(petSlotMachineSuccess) || motion != 0 || !visible {
		t.Fatalf("result frame = action %d motion %d visible %t", action, motion, visible)
	}
	if !mode.petSlotMachine.active {
		t.Fatal("result pause closed the slot machine too early")
	}

	_, _, visible = mode.petSlotMachineFrame(act, started.Add(601*time.Millisecond))
	if visible || mode.petSlotMachine.active {
		t.Fatal("slot machine stayed visible after the result pause")
	}
}
