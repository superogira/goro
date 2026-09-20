package game

import (
	"testing"
)

func TestStorageDepositDialogStepping(t *testing.T) {
	var dialog storageDepositDialog
	dialog.begin(7, 501, "Red Potion", 30)
	if !dialog.open || dialog.amount != 1 || dialog.max != 30 {
		t.Fatalf("begin state = open=%v amount=%d max=%d", dialog.open, dialog.amount, dialog.max)
	}

	dialog.adjust(1)
	dialog.adjust(1)
	if dialog.amount != 3 {
		t.Fatalf("after two +1 steps: amount = %d, want 3", dialog.amount)
	}
	dialog.adjust(10)
	if dialog.amount != 13 {
		t.Fatalf("after +10: amount = %d, want 13", dialog.amount)
	}
	dialog.adjust(-10)
	dialog.adjust(-1)
	if dialog.amount != 2 {
		t.Fatalf("after -10 -1: amount = %d, want 2", dialog.amount)
	}

	// Clamps at both ends.
	dialog.adjust(-100)
	if dialog.amount != 1 {
		t.Fatalf("lower clamp: amount = %d, want 1", dialog.amount)
	}
	dialog.adjust(100)
	if dialog.amount != 30 {
		t.Fatalf("upper clamp: amount = %d, want 30", dialog.amount)
	}
}

func TestStorageDepositDialogBeginClampsMax(t *testing.T) {
	var dialog storageDepositDialog
	dialog.begin(7, 501, "Red Potion", 0)
	if dialog.max != 1 {
		t.Fatalf("zero max clamped to %d, want 1", dialog.max)
	}
}

func TestStorageDepositDialogDirection(t *testing.T) {
	// A withdrawal opened through beginWithdraw must carry the withdraw
	// direction: it picks both the packet (MoveFromStorage) and the
	// dialog title. begin must reset it — the two dialog structs are
	// otherwise identical and reused across operations.
	var d storageDepositDialog
	d.beginWithdraw(12, 501, "Red Potion", 30)
	if !d.open || !d.withdraw {
		t.Fatalf("beginWithdraw: open=%v withdraw=%v, want true/true", d.open, d.withdraw)
	}
	if d.itemIndex != 12 || d.itemID != 501 || d.itemName != "Red Potion" || d.max != 30 || d.amount != 1 {
		t.Fatalf("beginWithdraw fields = %+v", d)
	}

	d.begin(4, 502, "Apple", 4)
	if d.withdraw {
		t.Fatal("begin must reset the direction to deposit")
	}
	if d.itemIndex != 4 || d.itemID != 502 || d.max != 4 {
		t.Fatalf("begin fields = %+v", d)
	}
}
