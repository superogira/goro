package game

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
)

// The on-screen keyboard for the handheld: no physical keyboard exists on
// the RG35XX, so text fields (login credentials, character names, chat)
// need a gamepad-driven overlay. START opens it whenever a text field has
// focus; the d-pad walks a QWERTY grid, A appends the character, B
// backspaces, Shift toggles case, and START submits.

// osk is package-level so both LoginMode and WorldMode can drive it.
var osk onScreenKeyboard

// oskShared carries the pacing timestamps without a mode receiver.
var oskShared = struct {
	actionAt time.Time
	movedAt  time.Time
}{}

func oskState() *onScreenKeyboard { return &osk }

const (
	oskCellW = 36
	oskCellH = 30
	oskPad   = 3
)

type oskKey struct {
	label   string // display
	upper   string // shifted display ("" = same)
	inject  string // text to insert ("" = same as label)
	special string // "shift", "space", "bksp", "enter", "sym", "abc"
}

var oskRows = [][]oskKey{
	{{label: "1"}, {label: "2"}, {label: "3"}, {label: "4"}, {label: "5"},
		{label: "6"}, {label: "7"}, {label: "8"}, {label: "9"}, {label: "0"}},
	{{label: "q", upper: "Q"}, {label: "w", upper: "W"}, {label: "e", upper: "E"},
		{label: "r", upper: "R"}, {label: "t", upper: "T"}, {label: "y", upper: "Y"},
		{label: "u", upper: "U"}, {label: "i", upper: "I"}, {label: "o", upper: "O"},
		{label: "p", upper: "P"}},
	{{label: "a", upper: "A"}, {label: "s", upper: "S"}, {label: "d", upper: "D"},
		{label: "f", upper: "F"}, {label: "g", upper: "G"}, {label: "h", upper: "H"},
		{label: "j", upper: "J"}, {label: "k", upper: "K"}, {label: "l", upper: "L"}},
	{{label: "z", upper: "Z"}, {label: "x", upper: "X"}, {label: "c", upper: "C"},
		{label: "v", upper: "V"}, {label: "b", upper: "B"}, {label: "n", upper: "N"},
		{label: "m", upper: "M"}, {label: "_", upper: "-"}},
	{{label: "Shift", special: "shift", inject: ""},
		{label: "Space", special: "space", inject: " "},
		{label: "⌫", special: "bksp"},
		{label: "OK", special: "enter"}},
}

var oskSymRows = [][]oskKey{
	{{label: "!"}, {label: "@"}, {label: "#"}, {label: "$"}, {label: "%"},
		{label: "^"}, {label: "&"}, {label: "*"}, {label: "("}, {label: ")"}},
	{{label: "-"}, {label: "+"}, {label: "="}, {label: "["}, {label: "]"},
		{label: "{"}, {label: "}"}, {label: "\\"}, {label: ":"}, {label: ";"}},
	{{label: "'"}, {label: "\""}, {label: "<"}, {label: ">"}, {label: ","},
		{label: "."}, {label: "?"}, {label: "/"}, {label: "|"}, {label: "`"}},
	{{label: "~"}, {label: "0"}, {label: "1"}, {label: "2"}, {label: "3"},
		{label: "4"}, {label: "5"}, {label: "6"}, {label: "7"}, {label: "8"}},
	{{label: "ABC", special: "abc"}},
}

type onScreenKeyboard struct {
	open    bool
	row     int
	col     int
	shift   bool
	symbols bool // false = letters page, true = symbols page
	// lastTyped tracks the last injection for the preview line.
	preview string
}

func (osk *onScreenKeyboard) rows() [][]oskKey {
	if osk.symbols {
		return oskSymRows
	}
	return oskRows
}

func (osk *onScreenKeyboard) currentKey() oskKey {
	rows := osk.rows()
	if osk.row < 0 || osk.row >= len(rows) {
		return oskKey{}
	}
	row := rows[osk.row]
	if osk.col < 0 || osk.col >= len(row) {
		return oskKey{}
	}
	return row[osk.col]
}

func (osk *onScreenKeyboard) clamp() {
	rows := osk.rows()
	if osk.row >= len(rows) {
		osk.row = len(rows) - 1
	}
	if osk.row < 0 {
		osk.row = 0
	}
	row := rows[osk.row]
	if osk.col >= len(row) {
		osk.col = len(row) - 1
	}
	if osk.col < 0 {
		osk.col = 0
	}
}

func (osk *onScreenKeyboard) move(dx, dy int) {
	rows := osk.rows()
	osk.row += dy
	osk.col += dx
	if osk.row < 0 {
		osk.row = len(rows) - 1
	}
	if osk.row >= len(rows) {
		osk.row = 0
	}
	row := rows[osk.row]
	if osk.col < 0 {
		osk.col = len(row) - 1
	}
	if osk.col >= len(row) {
		osk.col = 0
	}
}

// inject sends the key's effect into the input state.
func (osk *onScreenKeyboard) inject(ctx client.Context) {
	key := osk.currentKey()
	if key.special == "shift" {
		osk.shift = !osk.shift
		return
	}
	if key.special == "space" {
		ctx.Input.AddTextInput(" ")
		osk.preview += " "
		return
	}
	if key.special == "bksp" {
		// Feed a backspace press+release into the input state so the
		// focused widget deletes a character.
		ctx.Input.SetKeyCode(gpucontext.KeyBackspace, true)
		ctx.Input.SetKeyCode(gpucontext.KeyBackspace, false)
		if len(osk.preview) > 0 {
			runes := []rune(osk.preview)
			osk.preview = string(runes[:len(runes)-1])
		}
		return
	}
	if key.special == "enter" {
		oskSubmittedFlag = true
		ctx.Input.SetKeyCode(gpucontext.KeyEnter, true)
		osk.open = false
		return
	}
	if key.special == "abc" {
		osk.symbols = false
		osk.row, osk.col = 0, 0
		return
	}
	// Regular character: apply shift.
	ch := key.label
	if osk.shift && key.upper != "" {
		ch = key.upper
	}
	ctx.Input.AddTextInput(ch)
	osk.preview += ch
	if osk.shift {
		osk.shift = false
	}
}

var oskColors = struct {
	panel   color.RGBA
	border  color.RGBA
	gold    color.RGBA
	text    color.RGBA
	muted   color.RGBA
	active  color.RGBA
	outline color.RGBA
}{
	panel:   color.RGBA{R: 24, G: 20, B: 34, A: 240},
	border:  color.RGBA{R: 60, G: 48, B: 84, A: 250},
	gold:    color.RGBA{R: 214, G: 178, B: 92, A: 255},
	text:    color.RGBA{R: 244, G: 248, B: 252, A: 255},
	muted:   color.RGBA{R: 170, G: 178, B: 190, A: 220},
	active:  color.RGBA{R: 214, G: 178, B: 92, A: 220},
	outline: color.RGBA{R: 10, G: 12, B: 16, A: 210},
}

// updateGamepadOSK drives the keyboard overlay. Returns true while open.
func updateGamepadOSK(ctx client.Context, now time.Time) bool {
	osk := oskState()
	if !osk.open {
		return false
	}
	// The A button's companion mouse click must not reach the form under
	// the keyboard — clear both buttons every frame while the OSK is open.
	if ctx.Input != nil {
		ctx.Input.SetMouseButton(input.MouseButtonLeft, false)
		ctx.Input.SetMouseButton(input.MouseButtonRight, false)
	}
	switch {
	case ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13):
		// A: type the key.
		if now.Sub(oskShared.actionAt) >= gamepadActionFloor {
			oskShared.actionAt = now
			osk.inject(ctx)
		}
		return true
	case ctx.Input.KeyCodeJustPressed(gpucontext.KeyF18):
		// B: backspace (hold to close if the field is already empty).
		osk.row, osk.col = len(osk.rows())-1, 2
		osk.inject(ctx)
		return true
	case ctx.Input.KeyCodeJustPressed(gpucontext.KeyF18):
		// B (already handled above for backspace, but also close).
		osk.open = false
		return true
	case ctx.Input.KeyCodeJustPressed(gpucontext.KeyEnter):
		// START tap: submit the focused field.
		osk.row, osk.col = len(osk.rows())-1, len(osk.rows()[len(osk.rows())-1])-1
		osk.inject(ctx)
		return true
	case ctx.Input.KeyCodeJustPressed(gpucontext.KeyPrintScreen):
		// SELECT: toggle the symbols page.
		osk.symbols = !osk.symbols
		osk.row, osk.col = 0, 0
		return true
	}
	dx, dy := 0, 0
	if ctx.Input.JustPressed(input.KeyArrowLeft) {
		dx--
	}
	if ctx.Input.JustPressed(input.KeyArrowRight) {
		dx++
	}
	if ctx.Input.JustPressed(input.KeyArrowUp) {
		dy--
	}
	if ctx.Input.JustPressed(input.KeyArrowDown) {
		dy++
	}
	if dx != 0 || dy != 0 {
		if now.Sub(oskShared.movedAt) >= gamepadNavFloor {
			oskShared.movedAt = now
			osk.move(dx, dy)
		}
		return true
	}
	return true
}

// drawOSK renders the keyboard overlay at the bottom half of the screen.
func drawOSK(screen *render.Frame) {
	osk := oskState()
	if screen == nil || !osk.open {
		return
	}
	rows := osk.rows()
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	c := oskColors
	bounds := screen.Bounds()
	gridW := maxCols*oskCellW + (maxCols-1)*oskPad
	gridH := len(rows)*oskCellH + (len(rows)-1)*oskPad
	w := float64(gridW + 24)
	h := float64(gridH + 24 + 28) // grid + preview line + padding
	x := (float64(bounds.Dx()) - w) / 2
	y := float64(bounds.Dy()) - h - 8

	render.DrawRect(screen, x, y, w, h, c.panel)
	render.DrawRect(screen, x, y, w, 3, c.border)
	render.DrawRect(screen, x, y+3, w, 2, c.gold)

	// Preview line.
	label := "Letters"
	if osk.symbols {
		label = "Symbols"
	}
	shiftLabel := ""
	if osk.shift {
		shiftLabel = " (Shift)"
	}
	render.DrawOutlinedTextAt(screen, fmt.Sprintf("%s%s: %s", label, shiftLabel, osk.preview), int(x+12), int(y+10), c.muted, c.outline)

	// Grid.
	gy := y + 34
	for ri, row := range rows {
		rowW := len(row)*oskCellW + (len(row)-1)*oskPad
		rx := x + (w-float64(rowW))/2
		for ci, key := range row {
			kx := rx + float64(ci*(oskCellW+oskPad))
			ky := gy + float64(ri*(oskCellH+oskPad))
			active := ri == osk.row && ci == osk.col
			fill := color.RGBA{R: 40, G: 46, B: 58, A: 200}
			if active {
				fill = c.active
			}
			if key.special != "" {
				fill.A = 200
				if active {
					fill = c.active
				}
			}
			render.DrawRect(screen, kx, ky, oskCellW, oskCellH, fill)
			if active {
				render.DrawRect(screen, kx, ky, oskCellW, 2, c.gold)
				render.DrawRect(screen, kx, ky+float64(oskCellH)-2, oskCellW, 2, c.gold)
			}
			text := key.label
			if osk.shift && key.upper != "" {
				text = key.upper
			}
			if key.special == "shift" && osk.shift {
				text = "SHIFT"
			}
			// Center the label.
			tw := int(render.MeasureUIText(text, 12))
			tx := kx + (float64(oskCellW)-float64(tw))/2
			ty := ky + (float64(oskCellH)-14)/2
			labelColor := c.text
			if key.special != "" {
				labelColor = c.muted
			}
			render.DrawOutlinedTextAt(screen, text, int(tx), int(ty), labelColor, c.outline)
		}
	}
}

// tryOpenOSK opens the on-screen keyboard when a text field has focus.
func tryOpenOSK(textFocused bool) {
	if textFocused {
		osk.open = true
		osk.row, osk.col = 0, 0
		osk.preview = ""
	}
}

// trimOSKRunes clamps the preview line for drawing.
func trimOSKRunes(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes])
}

var _ = strings.TrimSpace

// oskJustSubmitted reports whether the OSK just fired Enter (the submit
// action), so the caller can let it through to the form.
var oskSubmittedFlag bool

func oskJustSubmitted() bool {
	v := oskSubmittedFlag
	oskSubmittedFlag = false
	return v
}
