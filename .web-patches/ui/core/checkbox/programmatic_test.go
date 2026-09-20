package checkbox

import "testing"

func TestProgrammaticToggleMatchesClick(t *testing.T) {
	// Gamepad-driven UIs toggle the checkbox through ToggleChecked; it must
	// match a click exactly: state flips, OnToggle fires once per toggle.
	toggles := 0
	cb := New(
		LabelOpt("Keep"),
		OnToggle(func(checked bool) { toggles++ }),
	)
	if cb.IsChecked() {
		t.Fatal("new checkbox should start unchecked")
	}
	cb.ToggleChecked()
	if !cb.IsChecked() || toggles != 1 {
		t.Fatalf("after ToggleChecked: checked=%v toggles=%d, want true/1", cb.IsChecked(), toggles)
	}
	// SetChecked with the same value is a no-op (no spurious callbacks).
	cb.SetChecked(true)
	if !cb.IsChecked() || toggles != 1 {
		t.Fatalf("SetChecked(true) on checked box: toggles=%d, want 1", toggles)
	}
	cb.SetChecked(false)
	if cb.IsChecked() || toggles != 2 {
		t.Fatalf("SetChecked(false): checked=%v toggles=%d, want false/2", cb.IsChecked(), toggles)
	}
}
