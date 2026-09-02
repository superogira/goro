// Package transition provides animated wrapper widgets that apply motion
// effects to child widgets.
//
// There are two levels of API: high-level convenience widgets ([Slide],
// [Fade]) and the lower-level [Transition] wrapper with composable
// [Effect] values.
//
// # High-Level Widgets
//
// [Slide] and [Fade] are standalone widgets that wrap a child and animate
// a single property. They implement [widget.Lifecycle] for auto-start on
// mount and follow the same time-based animation pattern as the progress
// spinner (elapsed time from [widget.Context.Now], easing functions).
//
//	slide := transition.NewSlide(myWidget,
//	    transition.SlideFrom(transition.FromTop),
//	    transition.SlideDuration(300 * time.Millisecond),
//	    transition.SlideEasing(animation.EaseOutCubic),
//	)
//
//	fade := transition.NewFade(myWidget,
//	    transition.FadeDuration(200 * time.Millisecond),
//	    transition.FadeEasing(animation.EaseInOutCubic),
//	)
//
// # Low-Level Transition Wrapper
//
// [Transition] created via [Wrap] supports composable effects (fade, slide,
// scale) for enter/exit animations. Use this when you need combined effects
// or explicit Show/Hide lifecycle control.
//
//	wrapped := transition.Wrap(myWidget,
//	    transition.EnterEffect(transition.FadeIn()),
//	    transition.ExitEffect(transition.SlideOut(transition.ToBottom)),
//	    transition.Duration(300 * time.Millisecond),
//	)
//
//	wrapped.Show()  // plays enter animation
//	wrapped.Hide()  // plays exit animation, then hides widget
//
// # Canvas Requirements
//
// Fade effects work best when the canvas implements [OpacityPusher] for
// true per-pixel opacity. Without it, [Fade] falls back to a background-
// color overlay approach (drawing a semi-transparent rect over the child).
// Slide effects use [widget.Canvas.PushTransform].
//
// # Retained Mode
//
// During animation, transition widgets call [widget.WidgetBase.SetNeedsRedraw]
// and [widget.Context.InvalidateRect] to request continuous repainting
// until the animation completes.
package transition
