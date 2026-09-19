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
