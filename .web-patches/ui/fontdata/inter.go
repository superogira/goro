// Package fontdata exposes copies of the fonts embedded by UI for callers
// that need to render text outside the standard UI canvas.
package fontdata

import "github.com/gogpu/ui/internal/render/fonts"

// InterRegular contains the embedded Inter Regular font data.
// Callers must not modify its contents.
var InterRegular = fonts.InterRegular

// InterBold contains the embedded Inter Bold font data.
// Callers must not modify its contents.
var InterBold = fonts.InterBold
