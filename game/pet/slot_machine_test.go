package pet

import (
	"testing"
	"time"

	"github.com/kivutar/goro/res"
)

func TestPetSlotMachineKeepsResultVisibleDuringPause(t *testing.T) {
	started := time.Unix(10, 0)
	act := &res.ACT{Actions: make([]res.ACTAction, int(petSlotMachineFail)+1)}
	act.Actions[petSlotMachineSuccess] = res.ACTAction{
		DelayMS:    100,
		Animations: []res.ACTAnimation{{}},
	}
	mode := &System{SlotMachine: SlotMachineState{
		Active:  true,
		Phase:   petSlotMachineSuccess,
		Started: started,
	}}

	action, motion, visible := mode.slotMachineFrame(act, started.Add(200*time.Millisecond))
	if action != int(petSlotMachineSuccess) || motion != 0 || !visible {
		t.Fatalf("result frame = action %d motion %d visible %t", action, motion, visible)
	}
	if !mode.SlotMachine.Active {
		t.Fatal("result pause closed the slot machine too early")
	}

	_, _, visible = mode.slotMachineFrame(act, started.Add(601*time.Millisecond))
	if visible || mode.SlotMachine.Active {
		t.Fatal("slot machine stayed visible after the result pause")
	}
}
