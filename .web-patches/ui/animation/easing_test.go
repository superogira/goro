package animation

import (
	"math"
	"testing"
)

func TestLinear(t *testing.T) {
	tests := []struct {
		input, expected float32
	}{
		{0, 0},
		{0.25, 0.25},
		{0.5, 0.5},
		{0.75, 0.75},
		{1, 1},
	}
	for _, tt := range tests {
		got := Linear(tt.input)
		if got != tt.expected {
			t.Errorf("Linear(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestEasingBoundaries(t *testing.T) {
	easings := []struct {
		name string
		fn   Easing
	}{
		{"EaseInQuad", EaseInQuad},
		{"EaseOutQuad", EaseOutQuad},
		{"EaseInOutQuad", EaseInOutQuad},
		{"EaseInCubic", EaseInCubic},
		{"EaseOutCubic", EaseOutCubic},
		{"EaseInOutCubic", EaseInOutCubic},
	}
	for _, e := range easings {
		t.Run(e.name, func(t *testing.T) {
			start := e.fn(0)
			end := e.fn(1)
			if math.Abs(float64(start)) > 1e-6 {
				t.Errorf("%s(0) = %v, want 0", e.name, start)
			}
			if math.Abs(float64(end-1)) > 1e-6 {
				t.Errorf("%s(1) = %v, want 1", e.name, end)
			}
		})
	}
}

func TestEasingMonotonicity(t *testing.T) {
	// All standard easings should be monotonically non-decreasing.
	easings := []struct {
		name string
		fn   Easing
	}{
		{"Linear", Linear},
		{"EaseInQuad", EaseInQuad},
		{"EaseOutQuad", EaseOutQuad},
		{"EaseInOutQuad", EaseInOutQuad},
		{"EaseInCubic", EaseInCubic},
		{"EaseOutCubic", EaseOutCubic},
		{"EaseInOutCubic", EaseInOutCubic},
	}

	const steps = 100
	for _, e := range easings {
		t.Run(e.name, func(t *testing.T) {
			prev := e.fn(0)
			for i := 1; i <= steps; i++ {
				tt := float32(i) / float32(steps)
				curr := e.fn(tt)
				if curr < prev-1e-6 {
					t.Errorf("%s not monotonic: f(%v)=%v < f(%v)=%v",
						e.name, tt, curr, float32(i-1)/float32(steps), prev)
				}
				prev = curr
			}
		})
	}
}

func TestEaseInOutQuadSymmetry(t *testing.T) {
	// f(0.5-x) + f(0.5+x) should equal 1.0
	for i := 0; i <= 50; i++ {
		x := float32(i) / 100
		a := EaseInOutQuad(0.5 - x)
		b := EaseInOutQuad(0.5 + x)
		sum := a + b
		if math.Abs(float64(sum-1)) > 1e-5 {
			t.Errorf("EaseInOutQuad symmetry: f(%v)+f(%v)=%v, want 1.0",
				0.5-x, 0.5+x, sum)
		}
	}
}

func TestEaseInOutCubicSymmetry(t *testing.T) {
	for i := 0; i <= 50; i++ {
		x := float32(i) / 100
		a := EaseInOutCubic(0.5 - x)
		b := EaseInOutCubic(0.5 + x)
		sum := a + b
		if math.Abs(float64(sum-1)) > 1e-5 {
			t.Errorf("EaseInOutCubic symmetry: f(%v)+f(%v)=%v, want 1.0",
				0.5-x, 0.5+x, sum)
		}
	}
}

func TestEaseInQuadKnownValues(t *testing.T) {
	// EaseInQuad(t) = t^2
	tests := []struct {
		t, want float32
	}{
		{0.1, 0.01},
		{0.5, 0.25},
		{0.9, 0.81},
	}
	for _, tt := range tests {
		got := EaseInQuad(tt.t)
		if math.Abs(float64(got-tt.want)) > 1e-6 {
			t.Errorf("EaseInQuad(%v) = %v, want %v", tt.t, got, tt.want)
		}
	}
}

func TestEaseOutQuadKnownValues(t *testing.T) {
	// EaseOutQuad(t) = 1 - (1-t)^2
	tests := []struct {
		t, want float32
	}{
		{0.1, 0.19},
		{0.5, 0.75},
		{0.9, 0.99},
	}
	for _, tt := range tests {
		got := EaseOutQuad(tt.t)
		if math.Abs(float64(got-tt.want)) > 1e-5 {
			t.Errorf("EaseOutQuad(%v) = %v, want %v", tt.t, got, tt.want)
		}
	}
}
