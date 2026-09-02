package animation

import "time"

// Material Design 3 easing curves.
//
// These match the exact values from M3 specification, Flutter's Easing class,
// and Android's MotionTokens.kt.
var (
	// M3Standard is the standard M3 easing for utility animations that
	// begin and end on screen.
	M3Standard = CubicBezier(0.2, 0.0, 0.0, 1.0)

	// M3StandardAccelerate is the M3 easing for utility animations
	// exiting the screen.
	M3StandardAccelerate = CubicBezier(0.3, 0.0, 1.0, 1.0)

	// M3StandardDecelerate is the M3 easing for utility animations
	// entering the screen.
	M3StandardDecelerate = CubicBezier(0.0, 0.0, 0.0, 1.0)

	// M3Emphasized is the primary M3 easing for common animations on screen.
	// This is a two-segment curve (ThreePointCubic) matching Flutter's exact
	// easeInOutCubicEmphasized definition.
	M3Emphasized = ThreePointCubic(
		[2]float32{0.05, 0},
		[2]float32{0.133333, 0.06},
		[2]float32{0.166666, 0.4},
		[2]float32{0.208333, 0.82},
		[2]float32{0.25, 1.0},
	)

	// M3EmphasizedAccelerate is the M3 easing for styled animations
	// exiting the screen.
	M3EmphasizedAccelerate = CubicBezier(0.3, 0.0, 0.8, 0.15)

	// M3EmphasizedDecelerate is the M3 easing for styled animations
	// entering the screen.
	M3EmphasizedDecelerate = CubicBezier(0.05, 0.7, 0.1, 1.0)

	// M3Linear is the M3 linear easing for simple, non-styled motion.
	M3Linear = Linear
)

// Material Design 3 duration tokens.
//
// Duration increases with animation area and traversal distance.
// Values match M3 specification exactly.
const (
	// DurationShort1 is for micro-interactions (icon state changes).
	DurationShort1 = 50 * time.Millisecond

	// DurationShort2 is for small element fades.
	DurationShort2 = 100 * time.Millisecond

	// DurationShort3 is for button press feedback.
	DurationShort3 = 150 * time.Millisecond

	// DurationShort4 is for checkbox and switch toggles.
	DurationShort4 = 200 * time.Millisecond

	// DurationMedium1 is for card expand and small transitions.
	DurationMedium1 = 250 * time.Millisecond

	// DurationMedium2 is for dialog appear and menu open.
	DurationMedium2 = 300 * time.Millisecond

	// DurationMedium3 is for medium area transitions.
	DurationMedium3 = 350 * time.Millisecond

	// DurationMedium4 is for navigation transitions.
	DurationMedium4 = 400 * time.Millisecond

	// DurationLong1 is for large area transitions.
	DurationLong1 = 450 * time.Millisecond

	// DurationLong2 is for full-screen transitions.
	DurationLong2 = 500 * time.Millisecond

	// DurationLong3 is for complex multi-element transitions.
	DurationLong3 = 550 * time.Millisecond

	// DurationLong4 is for large complex transitions.
	DurationLong4 = 600 * time.Millisecond

	// DurationExtraLong1 is for extended transitions.
	DurationExtraLong1 = 700 * time.Millisecond

	// DurationExtraLong2 is for extended complex transitions.
	DurationExtraLong2 = 800 * time.Millisecond

	// DurationExtraLong3 is for very large transitions.
	DurationExtraLong3 = 900 * time.Millisecond

	// DurationExtraLong4 is the maximum standard duration.
	DurationExtraLong4 = 1000 * time.Millisecond
)

// Spring damping ratio presets (from Jetpack Compose).
const (
	// DampingHighBouncy produces very bouncy springs with many oscillations.
	DampingHighBouncy float32 = 0.2

	// DampingMediumBouncy produces medium bounce springs.
	DampingMediumBouncy float32 = 0.5

	// DampingLowBouncy produces springs with slight bounce.
	DampingLowBouncy float32 = 0.75

	// DampingNoBouncy produces critically damped springs with no bounce.
	DampingNoBouncy float32 = 1.0
)

// Spring stiffness presets (from Jetpack Compose).
const (
	// StiffnessHigh produces very fast springs.
	StiffnessHigh float32 = 10000

	// StiffnessMedium is the default balanced stiffness.
	StiffnessMedium float32 = 1500

	// StiffnessLow produces slow, gentle springs.
	StiffnessLow float32 = 200

	// StiffnessVeryLow produces very slow, dramatic springs.
	StiffnessVeryLow float32 = 50
)
