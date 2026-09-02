package animation

import (
	"math"
	"testing"
)

func TestCubicBezierBoundaries(t *testing.T) {
	curves := []struct {
		name           string
		x1, y1, x2, y2 float32
	}{
		{"ease", 0.25, 0.1, 0.25, 1.0},
		{"ease-in", 0.42, 0, 1, 1},
		{"ease-out", 0, 0, 0.58, 1},
		{"ease-in-out", 0.42, 0, 0.58, 1},
		{"M3Standard", 0.2, 0.0, 0.0, 1.0},
	}
	for _, c := range curves {
		t.Run(c.name, func(t *testing.T) {
			ease := CubicBezier(c.x1, c.y1, c.x2, c.y2)
			start := ease(0)
			end := ease(1)
			if start != 0 {
				t.Errorf("CubicBezier(%s)(0) = %v, want 0", c.name, start)
			}
			if end != 1 {
				t.Errorf("CubicBezier(%s)(1) = %v, want 1", c.name, end)
			}
		})
	}
}

func TestCubicBezierNegativeAndOverOne(t *testing.T) {
	ease := CubicBezier(0.25, 0.1, 0.25, 1.0)
	// Values outside [0,1] should be clamped.
	if got := ease(-0.5); got != 0 {
		t.Errorf("CubicBezier(-0.5) = %v, want 0", got)
	}
	if got := ease(1.5); got != 1 {
		t.Errorf("CubicBezier(1.5) = %v, want 1", got)
	}
}

func TestCubicBezierLinear(t *testing.T) {
	// Linear bezier: cubic-bezier(0,0,1,1) should be identity.
	ease := CubicBezier(0, 0, 1, 1)
	for i := 0; i <= 10; i++ {
		x := float32(i) / 10
		got := ease(x)
		if math.Abs(float64(got-x)) > 0.001 {
			t.Errorf("Linear CubicBezier(%v) = %v, want %v", x, got, x)
		}
	}
}

func TestCubicBezierCSSEaseAccuracy(t *testing.T) {
	// CSS "ease" = cubic-bezier(0.25, 0.1, 0.25, 1.0)
	// Known reference values (computed from browser implementations).
	ease := CubicBezier(0.25, 0.1, 0.25, 1.0)
	tests := []struct {
		x, expected float32
		tolerance   float64
	}{
		{0.0, 0.0, 0.001},
		{0.25, 0.41, 0.02},
		{0.5, 0.80, 0.02},
		{0.75, 0.96, 0.02},
		{1.0, 1.0, 0.001},
	}
	for _, tt := range tests {
		got := ease(tt.x)
		if math.Abs(float64(got-tt.expected)) > tt.tolerance {
			t.Errorf("CSS ease(%v) = %v, want ~%v (tolerance %v)",
				tt.x, got, tt.expected, tt.tolerance)
		}
	}
}

func TestCubicBezierMonotonicity(t *testing.T) {
	// Standard easings should produce monotonically non-decreasing output.
	curves := []struct {
		name           string
		x1, y1, x2, y2 float32
	}{
		{"ease", 0.25, 0.1, 0.25, 1.0},
		{"ease-in-out", 0.42, 0, 0.58, 1},
		{"M3Standard", 0.2, 0.0, 0.0, 1.0},
	}

	const steps = 200
	for _, c := range curves {
		t.Run(c.name, func(t *testing.T) {
			ease := CubicBezier(c.x1, c.y1, c.x2, c.y2)
			prev := ease(0)
			for i := 1; i <= steps; i++ {
				x := float32(i) / float32(steps)
				curr := ease(x)
				if curr < prev-0.001 {
					t.Errorf("%s not monotonic at x=%v: %v < %v", c.name, x, curr, prev)
				}
				prev = curr
			}
		})
	}
}

func TestCubicBezierM3Curves(t *testing.T) {
	// Verify M3 curves evaluate without panic and hit boundaries.
	m3Curves := []struct {
		name string
		ease Easing
	}{
		{"M3Standard", M3Standard},
		{"M3StandardAccelerate", M3StandardAccelerate},
		{"M3StandardDecelerate", M3StandardDecelerate},
		{"M3EmphasizedAccelerate", M3EmphasizedAccelerate},
		{"M3EmphasizedDecelerate", M3EmphasizedDecelerate},
	}
	for _, c := range m3Curves {
		t.Run(c.name, func(t *testing.T) {
			start := c.ease(0)
			end := c.ease(1)
			if start != 0 {
				t.Errorf("%s(0) = %v, want 0", c.name, start)
			}
			if end != 1 {
				t.Errorf("%s(1) = %v, want 1", c.name, end)
			}
			// Check midpoint is reasonable.
			mid := c.ease(0.5)
			if mid < 0 || mid > 1.1 {
				t.Errorf("%s(0.5) = %v, out of reasonable range", c.name, mid)
			}
		})
	}
}

func TestThreePointCubicBoundaries(t *testing.T) {
	start := M3Emphasized(0)
	end := M3Emphasized(1)
	if start != 0 {
		t.Errorf("M3Emphasized(0) = %v, want 0", start)
	}
	if end != 1 {
		t.Errorf("M3Emphasized(1) = %v, want 1", end)
	}
}

func TestThreePointCubicMonotonicity(t *testing.T) {
	const steps = 200
	prev := M3Emphasized(0)
	for i := 1; i <= steps; i++ {
		x := float32(i) / float32(steps)
		curr := M3Emphasized(x)
		if curr < prev-0.01 {
			t.Errorf("M3Emphasized not monotonic at x=%v: %v < %v", x, curr, prev)
		}
		prev = curr
	}
}

func TestThreePointCubicMidpoint(t *testing.T) {
	// At x = midpoint.x (0.166666), output should be close to midpoint.y (0.4).
	got := M3Emphasized(0.166666)
	if math.Abs(float64(got-0.4)) > 0.05 {
		t.Errorf("M3Emphasized(0.166666) = %v, want ~0.4", got)
	}
}

func TestThreePointCubicNegativeAndOverOne(t *testing.T) {
	if got := M3Emphasized(-0.5); got != 0 {
		t.Errorf("M3Emphasized(-0.5) = %v, want 0", got)
	}
	if got := M3Emphasized(1.5); got != 1 {
		t.Errorf("M3Emphasized(1.5) = %v, want 1", got)
	}
}

func TestThreePointCubicCustom(t *testing.T) {
	// Create a custom ThreePointCubic and verify it evaluates correctly.
	ease := ThreePointCubic(
		[2]float32{0.1, 0.0},
		[2]float32{0.2, 0.1},
		[2]float32{0.5, 0.5},
		[2]float32{0.7, 0.9},
		[2]float32{0.9, 1.0},
	)
	start := ease(0)
	end := ease(1)
	if start != 0 {
		t.Errorf("custom ThreePointCubic(0) = %v, want 0", start)
	}
	if end != 1 {
		t.Errorf("custom ThreePointCubic(1) = %v, want 1", end)
	}
	// Midpoint test.
	mid := ease(0.5)
	if math.Abs(float64(mid-0.5)) > 0.1 {
		t.Errorf("custom ThreePointCubic(0.5) = %v, want ~0.5", mid)
	}
}

func BenchmarkCubicBezier(b *testing.B) {
	ease := CubicBezier(0.25, 0.1, 0.25, 1.0)
	b.ResetTimer()
	for b.Loop() {
		ease(0.5)
	}
}

func BenchmarkM3Emphasized(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		M3Emphasized(0.5)
	}
}
