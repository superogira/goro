package animation

// threePointCubic evaluates a two-segment cubic bezier curve that passes
// through three points: (0,0), a midpoint, and (1,1).
//
// This is used for Material Design 3's Emphasized easing curve, which
// cannot be represented by a single cubic bezier. Flutter calls this
// ThreePointCubic.
//
// Each segment is a standard cubic bezier evaluated independently:
//   - Segment 1: (0,0) -> a1 -> b1 -> midpoint   (for x < midpoint.x)
//   - Segment 2: midpoint -> a2 -> b2 -> (1,1)    (for x >= midpoint.x)
type threePointCubic struct {
	a1, b1   [2]float32 // Control points for first segment
	midpoint [2]float32 // Shared midpoint between segments
	a2, b2   [2]float32 // Control points for second segment

	seg1 *cubicBezierCurve // Precomputed first segment
	seg2 *cubicBezierCurve // Precomputed second segment
}

// newThreePointCubic creates a new two-segment cubic curve.
//
// Parameters specify the control points and midpoint:
//   - a1, b1: control points for the first segment (from origin to midpoint)
//   - mid: the shared midpoint where the two segments meet
//   - a2, b2: control points for the second segment (from midpoint to (1,1))
func newThreePointCubic(a1, b1, mid, a2, b2 [2]float32) *threePointCubic {
	tc := &threePointCubic{
		a1:       a1,
		b1:       b1,
		midpoint: mid,
		a2:       a2,
		b2:       b2,
	}

	// Normalize segment 1: map (0,0)->(midX,midY) to (0,0)->(1,1)
	midX := mid[0]
	midY := mid[1]
	if midX > 0 && midY > 0 {
		tc.seg1 = newCubicBezierCurve(
			a1[0]/midX, a1[1]/midY,
			b1[0]/midX, b1[1]/midY,
		)
	} else {
		// Degenerate: use linear for this segment.
		tc.seg1 = newCubicBezierCurve(0, 0, 1, 1)
	}

	// Normalize segment 2: map (midX,midY)->(1,1) to (0,0)->(1,1)
	rangeX := 1 - midX
	rangeY := 1 - midY
	if rangeX > 0 && rangeY > 0 {
		tc.seg2 = newCubicBezierCurve(
			(a2[0]-midX)/rangeX, (a2[1]-midY)/rangeY,
			(b2[0]-midX)/rangeX, (b2[1]-midY)/rangeY,
		)
	} else {
		// Degenerate: use linear for this segment.
		tc.seg2 = newCubicBezierCurve(0, 0, 1, 1)
	}

	return tc
}

// Evaluate returns the easing value for the given normalized time x in [0,1].
func (tc *threePointCubic) Evaluate(x float32) float32 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}

	midX := tc.midpoint[0]
	midY := tc.midpoint[1]

	if x < midX {
		// First segment: normalize x to [0,1] within segment range.
		if midX <= 0 {
			return 0
		}
		localT := x / midX
		return tc.seg1.Evaluate(localT) * midY
	}

	// Second segment: normalize x to [0,1] within segment range.
	rangeX := 1 - midX
	rangeY := 1 - midY
	if rangeX <= 0 {
		return 1
	}
	localT := (x - midX) / rangeX
	return midY + tc.seg2.Evaluate(localT)*rangeY
}

// ThreePointCubic creates an Easing function from a two-segment cubic curve.
//
// This is used for Material Design 3's Emphasized easing, which requires
// two joined cubic bezier segments sharing a midpoint.
//
// Parameters:
//   - a1, b1: control points for first segment (origin to midpoint)
//   - mid: shared midpoint
//   - a2, b2: control points for second segment (midpoint to end)
//
// Example (M3 Emphasized):
//
//	emphasized := animation.ThreePointCubic(
//	    [2]float32{0.05, 0},
//	    [2]float32{0.133333, 0.06},
//	    [2]float32{0.166666, 0.4},
//	    [2]float32{0.208333, 0.82},
//	    [2]float32{0.25, 1.0},
//	)
func ThreePointCubic(a1, b1, mid, a2, b2 [2]float32) Easing {
	tc := newThreePointCubic(a1, b1, mid, a2, b2)
	return tc.Evaluate
}
