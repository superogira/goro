package rotheme

import (
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

const (
	ButtonRadius        float32 = 6
	ButtonPaddingX      float32 = 8
	ButtonPaddingY      float32 = 5.5
	LargeButtonPaddingY float32 = 8.5
	buttonReflectInset  float32 = 3
	buttonReflectAlpha  float32 = 0.4
)

func Button(label string, onClick func()) *primitives.BoxWidget {
	return ButtonDisabled(label, false, onClick)
}

func ButtonDisabled(label string, disabled bool, onClick func()) *primitives.BoxWidget {
	return buttonWithPadding(label, disabled, ButtonPaddingY, onClick)
}

func ButtonDisabledFn(label string, disabled func() bool, onClick func()) *primitives.BoxWidget {
	return buttonWithPaddingFn(label, disabled, ButtonPaddingY, onClick)
}

func LargeButton(label string, onClick func()) *primitives.BoxWidget {
	return LargeButtonDisabled(label, false, onClick)
}

func LargeButtonDisabled(label string, disabled bool, onClick func()) *primitives.BoxWidget {
	return buttonWithPadding(label, disabled, LargeButtonPaddingY, onClick)
}

func buttonWithPadding(label string, disabled bool, paddingY float32, onClick func()) *primitives.BoxWidget {
	opts := []button.Option{
		button.TextOpt(label),
		button.SizeOpt(button.Small),
		button.PainterOpt(ButtonPainter{}),
		button.RoundedOpt(ButtonRadius),
		button.Disabled(disabled),
	}
	if onClick != nil {
		opts = append(opts, button.OnClick(onClick))
	}
	return primitives.Box(
		newMouseOnlyButton(button.New(opts...).PaddingXY(ButtonPaddingX, paddingY)),
	).
		CrossAlign(primitives.CrossAxisStretch).
		Height(Default.Typography.TextSize + paddingY*2)
}

func buttonWithPaddingFn(label string, disabled func() bool, paddingY float32, onClick func()) *primitives.BoxWidget {
	opts := []button.Option{
		button.TextOpt(label),
		button.SizeOpt(button.Small),
		button.PainterOpt(ButtonPainter{}),
		button.RoundedOpt(ButtonRadius),
		button.DisabledFn(disabled),
	}
	if onClick != nil {
		opts = append(opts, button.OnClick(onClick))
	}
	return primitives.Box(
		newMouseOnlyButton(button.New(opts...).PaddingXY(ButtonPaddingX, paddingY)),
	).
		CrossAlign(primitives.CrossAxisStretch).
		Height(Default.Typography.TextSize + paddingY*2)
}

type mouseOnlyButtonWidget struct {
	widget.WidgetBase
	button *button.Widget
}

func newMouseOnlyButton(btn *button.Widget) *mouseOnlyButtonWidget {
	w := &mouseOnlyButtonWidget{button: btn}
	w.SetVisible(true)
	w.SetEnabled(true)
	btn.SetParent(w)
	return w
}

func (w *mouseOnlyButtonWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	return w.button.Layout(ctx, constraints)
}

func (w *mouseOnlyButtonWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	widget.StampScreenOrigin(w.button, canvas)
	widget.DrawChild(w.button, ctx, canvas)
}

func (w *mouseOnlyButtonWidget) Event(ctx widget.Context, e event.Event) bool {
	if _, ok := e.(*event.KeyEvent); ok {
		return false
	}
	consumed := w.button.Event(ctx, e)
	if mouse, ok := e.(*event.MouseEvent); ok && mouse.Button == event.ButtonLeft {
		ctx.ReleaseFocus(w.button)
	}
	return consumed
}

func (w *mouseOnlyButtonWidget) Children() []widget.Widget {
	return []widget.Widget{w.button}
}

func (w *mouseOnlyButtonWidget) SetBounds(bounds geometry.Rect) {
	w.WidgetBase.SetBounds(bounds)
	w.button.SetBounds(bounds)
}

type ButtonPainter struct{}

func (ButtonPainter) PaintButton(canvas widget.Canvas, state button.PaintState) {
	if state.Bounds.IsEmpty() {
		return
	}
	top, bottom := buttonTitleBarGradient(2)
	if state.Background != nil {
		top = *state.Background
		bottom = buttonGradientLight(top)
	}
	if state.Hovered {
		top, bottom = buttonTitleBarGradient(4)
	}
	if state.Pressed {
		top = Default.Colors.ButtonDown
		bottom = buttonGradientLight(top)
	}
	if state.Disabled {
		top = Default.Colors.Disabled
		bottom = buttonGradientLight(top)
	}
	border := Default.Colors.ButtonBorder
	if state.Disabled {
		border = Default.Colors.FooterLine
	}
	radius := ButtonRadius
	if state.Radius != nil {
		radius = *state.Radius
	}
	drawButtonShadow(canvas, state.Bounds, radius)
	drawButtonGradientColors(canvas, state.Bounds, top, bottom, radius)
	drawButtonReflect(canvas, state.Bounds, radius)
	canvas.StrokeRoundRect(state.Bounds, border, radius, 1)

	text := Default.Colors.Text
	if state.Disabled {
		text = Default.Colors.MutedText
	}
	DrawText(canvas, state.Text, state.Bounds, Default.Typography.TextSize, text, false, widget.TextAlignCenter)
}

func buttonGradientLight(dark widget.Color) widget.Color {
	return dark.Lerp(widget.RGBA(1, 1, 1, dark.A), 0.42)
}

func lighterTitleBarGradient(factor float32) (top, bottom widget.Color) {
	return lighterGradient(Default.Colors.WindowTitleTop, Default.Colors.WindowTitle, factor)
}

func buttonTitleBarGradient(factor float32) (top, bottom widget.Color) {
	light, dark := lighterTitleBarGradient(factor)
	return dark, light
}

func lighterGradient(top, bottom widget.Color, factor float32) (widget.Color, widget.Color) {
	return lighterColor(top, factor), lighterColor(bottom, factor)
}

func lighterColor(base widget.Color, factor float32) widget.Color {
	if factor <= 1 {
		return base
	}
	return base.Lerp(widget.RGBA(1, 1, 1, base.A), 1-1/factor)
}

func drawButtonGradient(canvas widget.Canvas, bounds geometry.Rect, dark widget.Color, radius float32) {
	drawButtonGradientColors(canvas, bounds, dark, buttonGradientLight(dark), radius)
}

func drawButtonGradientColors(canvas widget.Canvas, bounds geometry.Rect, top, bottom widget.Color, radius float32) {
	if radius > 0 {
		canvas.PushClipRoundRect(bounds, radius)
		defer canvas.PopClip()
	}
	DrawVerticalGradient(canvas, bounds, top, bottom)
}

func drawButtonShadow(canvas widget.Canvas, bounds geometry.Rect, radius float32) {
	canvas.DrawRoundRect(bounds.TranslateXY(0, 2), widget.RGBA(0.4, 0.4, 0.4, 0.15), radius)
	canvas.DrawRoundRect(bounds.TranslateXY(0, 1), widget.RGBA(0.4, 0.4, 0.4, 0.35), radius)
}

func drawButtonReflect(canvas widget.Canvas, bounds geometry.Rect, radius float32) {
	width := bounds.Width() - buttonReflectInset*2
	height := (bounds.Height() - buttonReflectInset*2) / 2
	if width <= 0 || height <= 0 {
		return
	}
	reflectBounds := geometry.NewRect(
		bounds.Min.X+buttonReflectInset,
		bounds.Min.Y+buttonReflectInset,
		width,
		height,
	)
	reflectRadius := min(max(float32(0), radius-buttonReflectInset), height/2)
	reflectColor := widget.RGBA(1, 1, 1, buttonReflectAlpha)
	if reflectRadius > 0 {
		canvas.DrawRoundRect(reflectBounds, reflectColor, reflectRadius)
		return
	}
	canvas.DrawRect(reflectBounds, reflectColor)
}
