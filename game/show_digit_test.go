package game

import (
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/network"
)

func TestShowDigitClockModes(t *testing.T) {
	now := time.Unix(100, 0)
	static := newShowDigitState(network.ShowDigit{Mode: network.ShowDigitStatic, Value: 3661}, now)
	actions, visible := static.actions(now)
	want := []int{0, 1, showDigitSeparator, 0, 1, showDigitSeparator, 0, 1}
	if !visible || !reflect.DeepEqual(actions, want) {
		t.Fatalf("3661 actions = %v, %t, want %v", actions, visible, want)
	}
	if _, visible := static.actions(now.Add(showDigitStatic)); visible {
		t.Fatal("static clock remained visible after five seconds")
	}

	up := newShowDigitState(network.ShowDigit{Mode: network.ShowDigitCountUp, Value: -58}, now)
	if value, _ := up.value(now.Add(3 * time.Second)); value != 61 {
		t.Fatalf("count-up value = %d, want 61", value)
	}
	down := newShowDigitState(network.ShowDigit{Mode: network.ShowDigitCountDown, Value: -1}, now)
	wantWrapped := ^uint32(0)
	if value, _ := down.value(now.Add(2 * time.Second)); value != wantWrapped {
		t.Fatalf("wrapped countdown = %d, want %d", value, wantWrapped)
	}
	fast := newShowDigitState(network.ShowDigit{Mode: network.ShowDigitFastCountDown, Value: 3}, now)
	if value, visible := fast.value(now.Add(time.Second)); !visible || value != 1 {
		t.Fatalf("fast countdown = %d, %t, want 1, true", value, visible)
	}
	if _, visible := fast.value(now.Add(1500 * time.Millisecond)); visible {
		t.Fatal("fast countdown remained visible at zero")
	}
}
