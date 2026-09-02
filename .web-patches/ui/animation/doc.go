// Package animation provides an enterprise-grade animation engine for gogpu/ui.
//
// The engine drives [state.Signal] values through time-based tweens and
// physics-based springs, integrating with the reactive signal system for
// automatic widget invalidation and event-driven rendering.
//
// # Architecture
//
// The animation engine follows the Flutter pattern:
//   - Engine ONLY animates float32 signals
//   - [Tween] maps float32 progress to any type T (Color, Point, Size, etc.)
//   - [Controller] manages active animations and provides the Tick entry point
//
// # Tween Animation
//
//	ctrl := animation.NewController()
//	opacity := state.NewSignal[float32](0)
//
//	animation.To(opacity, 1.0).
//	    Duration(300 * time.Millisecond).
//	    Ease(animation.M3Standard).
//	    Start(ctrl)
//
// # Spring Animation
//
//	position := state.NewSignal[float32](0)
//
//	animation.SpringTo(position, 200.0).
//	    Stiffness(animation.StiffnessMedium).
//	    DampingRatio(animation.DampingNoBouncy).
//	    Start(ctrl)
//
// # Type Interpolation with Tween[T]
//
//	progress := state.NewSignal[float32](0)
//	animation.To(progress, 1.0).Duration(300*time.Millisecond).Start(ctrl)
//
//	colorTween := animation.NewColorTween(startColor, endColor)
//	currentColor := colorTween.At(progress.Get()) // in Draw
//
// # Composition
//
//	animation.NewSequence(anim1, anim2).Start(ctrl)
//	animation.NewParallel(anim1, anim2).Start(ctrl)
//
// # Frame Integration
//
//	// In OnDraw callback:
//	active := ctrl.Tick(dt) // updates signals
//	scheduler.Flush()        // picks up changes
//	window.DrawTo(canvas)    // renders
//	if active { RequestRedraw() }
package animation
