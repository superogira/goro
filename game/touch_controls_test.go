package game

import (
	"testing"

	"github.com/kivutar/goro/client"
)

func TestParseTouchStickVector(t *testing.T) {
	cases := []struct {
		in   string
		x, y float64
		ok   bool
	}{
		{"0.50,-0.75", 0.5, -0.75, true},
		{"1.00,0.00", 1, 0, true},
		{"0,0", 0, 0, true},
		{"junk", 0, 0, false},
		{"1.5", 0, 0, false},
		{"a,b", 0, 0, false},
	}
	for _, tc := range cases {
		x, y, ok := parseTouchStickVector(tc.in)
		if ok != tc.ok || (ok && (x != tc.x || y != tc.y)) {
			t.Fatalf("parseTouchStickVector(%q) = (%v,%v,%v), want (%v,%v,%v)", tc.in, x, y, ok, tc.x, tc.y, tc.ok)
		}
	}
}

func TestTouchActionRangeScalesWithSnapRadius(t *testing.T) {
	cases := []struct {
		multiplier float64
		want       float64
	}{
		{1, 2.5},
		{2.5, 6.25},
		{0.5, 1.25},
	}
	for _, tc := range cases {
		ctx := client.Context{}
		ctx.Config.Gameplay.SnapRadius = tc.multiplier
		if got := touchActionBaseCells * inputPickMultiplier(ctx); got != tc.want {
			t.Fatalf("action range at Snap Radius %v = %v, want %v", tc.multiplier, got, tc.want)
		}
	}
}
