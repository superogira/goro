package testutil

import (
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
)

func DrawnWidgetText(root widget.Widget) string {
	if root == nil {
		return ""
	}
	// Window overlays expose their updated children after drawing the damaged
	// frame. Assert the text the player would see, after that render step.
	ctx := widget.NewContext()
	root.Layout(ctx, geometry.Loose(geometry.Sz(1024, 768)))
	root.Draw(ctx, &uitest.MockCanvas{})
	return WidgetText(root)
}

func WidgetText(root widget.Widget) string {
	text := ""
	if label, ok := root.(interface{ Content() string }); ok {
		text = label.Content() + "\n"
	}
	for _, child := range root.Children() {
		text += WidgetText(child)
	}
	return text
}
