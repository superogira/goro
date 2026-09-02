package animation

// Easing maps normalized time [0,1] to animation progress [0,1].
//
// The input t is clamped to [0,1] by the animation engine before calling.
// The output may exceed [0,1] for overshoot effects (e.g., back easing).
type Easing func(t float32) float32

// Standard polynomial easing functions.
var (
	// Linear returns t unchanged (constant speed).
	Linear Easing = func(t float32) float32 { return t }

	// EaseInQuad accelerates from zero velocity (quadratic).
	EaseInQuad Easing = func(t float32) float32 { return t * t }

	// EaseOutQuad decelerates to zero velocity (quadratic).
	EaseOutQuad Easing = func(t float32) float32 {
		inv := 1 - t
		return 1 - inv*inv
	}

	// EaseInOutQuad accelerates then decelerates (quadratic).
	EaseInOutQuad Easing = func(t float32) float32 {
		if t < 0.5 {
			return 2 * t * t
		}
		inv := -2*t + 2
		return 1 - inv*inv/2
	}

	// EaseInCubic accelerates from zero velocity (cubic).
	EaseInCubic Easing = func(t float32) float32 { return t * t * t }

	// EaseOutCubic decelerates to zero velocity (cubic).
	EaseOutCubic Easing = func(t float32) float32 {
		inv := 1 - t
		return 1 - inv*inv*inv
	}

	// EaseInOutCubic accelerates then decelerates (cubic).
	EaseInOutCubic Easing = func(t float32) float32 {
		if t < 0.5 {
			return 4 * t * t * t
		}
		inv := -2*t + 2
		return 1 - inv*inv*inv/2
	}
)
