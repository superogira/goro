package animation

// sampleTableSize is the number of precomputed X(t) samples for CubicBezier lookup.
const sampleTableSize = 11

// sampleStepSize is the parametric distance between each sample.
const sampleStepSize = 1.0 / float32(sampleTableSize-1)

// newtonIterations is the number of Newton-Raphson refinement steps.
const newtonIterations = 4

// newtonMinSlope is the minimum derivative threshold below which Newton-Raphson
// falls back to bisection.
const newtonMinSlope = 1e-7

// bisectionEpsilon is the convergence threshold for bisection fallback.
const bisectionEpsilon = 1e-7

// bisectionMaxIter is the maximum number of bisection iterations.
const bisectionMaxIter = 10

// cubicBezierCurve evaluates a cubic bezier easing curve.
//
// Given control points (x1,y1) and (x2,y2), with implicit endpoints (0,0) and (1,1),
// it solves X(t) = x for parameter t using a precomputed sample table with
// Newton-Raphson refinement and bisection fallback, then evaluates Y(t).
//
// This is the industry-standard algorithm used in Chrome, Firefox, and the
// bezier-easing npm package. Performance is approximately 10ns per evaluation.
type cubicBezierCurve struct {
	x1, y1, x2, y2 float32
	samples        [sampleTableSize]float32
}

// newCubicBezierCurve creates a new cubic bezier curve with precomputed sample table.
func newCubicBezierCurve(x1, y1, x2, y2 float32) *cubicBezierCurve {
	c := &cubicBezierCurve{x1: x1, y1: y1, x2: x2, y2: y2}
	// Precompute X(t) samples at evenly spaced t values.
	for i := range sampleTableSize {
		t := float32(i) * sampleStepSize
		c.samples[i] = c.calcX(t)
	}
	return c
}

// Evaluate returns the easing value for the given normalized time x in [0,1].
func (c *cubicBezierCurve) Evaluate(x float32) float32 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	// Special case: linear bezier.
	if c.x1 == c.y1 && c.x2 == c.y2 {
		return x
	}
	t := c.findT(x)
	return c.calcY(t)
}

// findT solves X(t) = x for parameter t using sample table + Newton-Raphson.
func (c *cubicBezierCurve) findT(x float32) float32 {
	// Binary search in sample table for initial interval.
	intervalStart := float32(0)
	currentSample := 1
	lastSample := sampleTableSize - 1

	for ; currentSample != lastSample && c.samples[currentSample] <= x; currentSample++ {
		intervalStart += sampleStepSize
	}
	currentSample--

	// Linear interpolation within the interval for initial guess.
	dist := (x - c.samples[currentSample]) / (c.samples[currentSample+1] - c.samples[currentSample])
	guessT := intervalStart + dist*sampleStepSize

	// Newton-Raphson refinement.
	for range newtonIterations {
		slope := c.calcDX(guessT)
		if slope < newtonMinSlope {
			// Derivative too small, fall back to bisection.
			return c.bisect(x, intervalStart, intervalStart+sampleStepSize)
		}
		currentX := c.calcX(guessT) - x
		guessT -= currentX / slope
	}

	return guessT
}

// bisect uses binary search to find t where X(t) = x.
func (c *cubicBezierCurve) bisect(x, a, b float32) float32 {
	for range bisectionMaxIter {
		mid := a + (b-a)/2
		xMid := c.calcX(mid) - x
		switch {
		case xMid > bisectionEpsilon:
			b = mid
		case xMid < -bisectionEpsilon:
			a = mid
		default:
			return mid
		}
	}
	return a + (b-a)/2
}

// calcX evaluates the X component of the cubic bezier at parameter t.
//
// X(t) = 3*(1-t)^2*t*x1 + 3*(1-t)*t^2*x2 + t^3
//
// In Horner form with coefficients:
//
//	a = 1 - 3*x2 + 3*x1
//	b = 3*x2 - 6*x1
//	c = 3*x1
//	X(t) = ((a*t + b)*t + c)*t
func (c *cubicBezierCurve) calcX(t float32) float32 {
	return ((1-3*c.x2+3*c.x1)*t+3*c.x2-6*c.x1)*t*t + 3*c.x1*t
}

// calcY evaluates the Y component of the cubic bezier at parameter t.
func (c *cubicBezierCurve) calcY(t float32) float32 {
	return ((1-3*c.y2+3*c.y1)*t+3*c.y2-6*c.y1)*t*t + 3*c.y1*t
}

// calcDX evaluates the derivative dX/dt at parameter t.
//
// dX/dt = 3*a*t^2 + 2*b*t + c where a,b,c are from calcX.
func (c *cubicBezierCurve) calcDX(t float32) float32 {
	return 3*(1-3*c.x2+3*c.x1)*t*t + 2*(3*c.x2-6*c.x1)*t + 3*c.x1
}

// CubicBezier creates an Easing function from cubic bezier control points.
//
// The control points define a CSS-style cubic-bezier(x1, y1, x2, y2) curve.
// The curve starts at (0,0) and ends at (1,1). x1 and x2 should be in [0,1].
//
// Example:
//
//	ease := animation.CubicBezier(0.25, 0.1, 0.25, 1.0) // CSS "ease"
func CubicBezier(x1, y1, x2, y2 float32) Easing {
	curve := newCubicBezierCurve(x1, y1, x2, y2)
	return curve.Evaluate
}
