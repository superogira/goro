package animation

import (
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// LerpFunc defines a function that linearly interpolates between two values
// of type T. t=0 returns begin, t=1 returns end.
type LerpFunc[T any] func(begin, end T, t float32) T

// Tween maps a float32 progress value [0,1] to an interpolated value of type T.
//
// Tween is a pure evaluator with no lifecycle or state. It follows the Flutter
// pattern where AnimationController produces 0..1 and Tween maps that to any type.
//
// Example:
//
//	colorTween := animation.NewColorTween(red, blue)
//	mid := colorTween.At(0.5) // 50% between red and blue
type Tween[T any] struct {
	begin T
	end   T
	lerp  LerpFunc[T]
}

// NewTween creates a Tween with a custom interpolation function.
//
// The lerpFn is called with t in [0,1] to interpolate between begin and end.
func NewTween[T any](begin, end T, lerpFn LerpFunc[T]) *Tween[T] {
	return &Tween[T]{begin: begin, end: end, lerp: lerpFn}
}

// At evaluates the tween at the given progress t.
//
// t=0 returns begin, t=1 returns end. Values outside [0,1] may extrapolate
// depending on the lerp function.
func (tw *Tween[T]) At(t float32) T {
	return tw.lerp(tw.begin, tw.end, t)
}

// Begin returns the start value.
func (tw *Tween[T]) Begin() T { return tw.begin }

// End returns the end value.
func (tw *Tween[T]) End() T { return tw.end }

// LerpFloat32 linearly interpolates between two float32 values.
func LerpFloat32(begin, end, t float32) float32 {
	return begin + (end-begin)*t
}

// LerpColor linearly interpolates between two widget.Color values.
func LerpColor(begin, end widget.Color, t float32) widget.Color {
	return begin.Lerp(end, t)
}

// LerpPoint linearly interpolates between two geometry.Point values.
func LerpPoint(begin, end geometry.Point, t float32) geometry.Point {
	return begin.Lerp(end, t)
}

// LerpSize linearly interpolates between two geometry.Size values.
func LerpSize(begin, end geometry.Size, t float32) geometry.Size {
	return geometry.Size{
		Width:  begin.Width + (end.Width-begin.Width)*t,
		Height: begin.Height + (end.Height-begin.Height)*t,
	}
}

// NewFloat32Tween creates a Tween that interpolates between two float32 values.
func NewFloat32Tween(begin, end float32) *Tween[float32] {
	return NewTween(begin, end, LerpFloat32)
}

// NewColorTween creates a Tween that interpolates between two colors.
func NewColorTween(begin, end widget.Color) *Tween[widget.Color] {
	return NewTween(begin, end, LerpColor)
}

// NewPointTween creates a Tween that interpolates between two points.
func NewPointTween(begin, end geometry.Point) *Tween[geometry.Point] {
	return NewTween(begin, end, LerpPoint)
}

// NewSizeTween creates a Tween that interpolates between two sizes.
func NewSizeTween(begin, end geometry.Size) *Tween[geometry.Size] {
	return NewTween(begin, end, LerpSize)
}
