package animation

import (
	"math"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

func TestNewFloat32Tween(t *testing.T) {
	tw := NewFloat32Tween(0, 100)

	tests := []struct {
		t    float32
		want float32
	}{
		{0, 0},
		{0.25, 25},
		{0.5, 50},
		{0.75, 75},
		{1, 100},
	}
	for _, tt := range tests {
		got := tw.At(tt.t)
		if math.Abs(float64(got-tt.want)) > 1e-5 {
			t.Errorf("Float32Tween.At(%v) = %v, want %v", tt.t, got, tt.want)
		}
	}
}

func TestNewFloat32TweenBeginEnd(t *testing.T) {
	tw := NewFloat32Tween(10, 20)
	if tw.Begin() != 10 {
		t.Errorf("Begin() = %v, want 10", tw.Begin())
	}
	if tw.End() != 20 {
		t.Errorf("End() = %v, want 20", tw.End())
	}
}

func TestNewColorTween(t *testing.T) {
	red := widget.Color{R: 1, G: 0, B: 0, A: 1}
	blue := widget.Color{R: 0, G: 0, B: 1, A: 1}
	tw := NewColorTween(red, blue)

	mid := tw.At(0.5)
	if math.Abs(float64(mid.R-0.5)) > 1e-5 {
		t.Errorf("ColorTween mid R = %v, want 0.5", mid.R)
	}
	if math.Abs(float64(mid.B-0.5)) > 1e-5 {
		t.Errorf("ColorTween mid B = %v, want 0.5", mid.B)
	}

	// Start and end.
	start := tw.At(0)
	if start != red {
		t.Errorf("ColorTween.At(0) = %v, want red", start)
	}
	end := tw.At(1)
	if end != blue {
		t.Errorf("ColorTween.At(1) = %v, want blue", end)
	}
}

func TestNewPointTween(t *testing.T) {
	p1 := geometry.Pt(0, 0)
	p2 := geometry.Pt(100, 200)
	tw := NewPointTween(p1, p2)

	mid := tw.At(0.5)
	if math.Abs(float64(mid.X-50)) > 1e-5 || math.Abs(float64(mid.Y-100)) > 1e-5 {
		t.Errorf("PointTween.At(0.5) = %v, want (50, 100)", mid)
	}

	start := tw.At(0)
	if start != p1 {
		t.Errorf("PointTween.At(0) = %v, want %v", start, p1)
	}
	end := tw.At(1)
	if end != p2 {
		t.Errorf("PointTween.At(1) = %v, want %v", end, p2)
	}
}

func TestNewSizeTween(t *testing.T) {
	s1 := geometry.Sz(100, 50)
	s2 := geometry.Sz(200, 100)
	tw := NewSizeTween(s1, s2)

	mid := tw.At(0.5)
	if math.Abs(float64(mid.Width-150)) > 1e-5 || math.Abs(float64(mid.Height-75)) > 1e-5 {
		t.Errorf("SizeTween.At(0.5) = %v, want (150, 75)", mid)
	}
}

func TestCustomTween(t *testing.T) {
	// Custom tween that interpolates between two strings via index.
	type indexedStr struct {
		text  string
		index float32
	}
	tw := NewTween(
		indexedStr{text: "hello", index: 0},
		indexedStr{text: "world", index: 1},
		func(begin, end indexedStr, tt float32) indexedStr {
			return indexedStr{
				text:  begin.text, // just keep begin text
				index: begin.index + (end.index-begin.index)*tt,
			}
		},
	)

	got := tw.At(0.5)
	if got.index != 0.5 {
		t.Errorf("Custom tween.At(0.5).index = %v, want 0.5", got.index)
	}
}

func TestLerpFloat32(t *testing.T) {
	tests := []struct {
		begin, end, tt, want float32
	}{
		{0, 100, 0, 0},
		{0, 100, 0.5, 50},
		{0, 100, 1, 100},
		{-50, 50, 0.5, 0},
	}
	for _, tt := range tests {
		got := LerpFloat32(tt.begin, tt.end, tt.tt)
		if math.Abs(float64(got-tt.want)) > 1e-5 {
			t.Errorf("LerpFloat32(%v, %v, %v) = %v, want %v",
				tt.begin, tt.end, tt.tt, got, tt.want)
		}
	}
}

func TestLerpColor(t *testing.T) {
	black := widget.Color{R: 0, G: 0, B: 0, A: 1}
	white := widget.Color{R: 1, G: 1, B: 1, A: 1}
	mid := LerpColor(black, white, 0.5)
	if math.Abs(float64(mid.R-0.5)) > 1e-5 {
		t.Errorf("LerpColor mid R = %v, want 0.5", mid.R)
	}
}

func TestLerpPoint(t *testing.T) {
	p1 := geometry.Pt(0, 0)
	p2 := geometry.Pt(10, 20)
	mid := LerpPoint(p1, p2, 0.5)
	if mid.X != 5 || mid.Y != 10 {
		t.Errorf("LerpPoint mid = %v, want (5, 10)", mid)
	}
}

func TestLerpSize(t *testing.T) {
	s1 := geometry.Sz(0, 0)
	s2 := geometry.Sz(100, 200)
	mid := LerpSize(s1, s2, 0.5)
	if mid.Width != 50 || mid.Height != 100 {
		t.Errorf("LerpSize mid = %v, want (50, 100)", mid)
	}
}
