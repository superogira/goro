package game

import (
	"image/color"
	"strings"
	"time"

	"github.com/gogpu/gogpu/hooks"
	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
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
	actionAt   time.Time
	movedAt    time.Time
	aLatch     bool
	bLatch     bool
	enterLatch bool
	selLatch   bool
}{}

// OSK pacing. Movement is edge-only with a debounce floor: one press
// edge = one cell, always. The held-state (KeyCodeDown) reading proved
// unreliable on this hardware — after release it lingered "down" long
// enough that a hold-repeat built on it multi-stepped single taps and
// merged quick taps into phantom holds. If the firmware emits repeat
// press pairs while the d-pad is held, those arrive as edges too and
// stride at the floor's pace naturally.
const (
	oskActionFloor = 100 * time.Millisecond
	oskNavFloor    = 120 * time.Millisecond
)

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

// oskCallback delivers a typed character or action to the host mode.
// The host sets this before showing the keyboard; nil callbacks are
// ignored.
var oskCallback func(ch string, action string)

// oskL1Handler runs when L1 is pressed while the OSK hook is active.
var oskL1Handler func() // action: "type", "bksp", "submit"

// inject sends the key's effect to the host via the callback.
func (osk *onScreenKeyboard) inject(_ client.Context) {
	key := osk.currentKey()
	glog.Infof("osk: inject key=%q special=%q row=%d col=%d callback=%v", key.label, key.special, osk.row, osk.col, oskCallback != nil)
	if key.special == "shift" {
		osk.shift = !osk.shift
		return
	}
	if key.special == "space" {
		if oskCallback != nil {
			oskCallback(" ", "type")
		}
		return
	}
	if key.special == "bksp" {
		if oskCallback != nil {
			oskCallback("", "bksp")
		}
		return
	}
	if key.special == "enter" {
		oskSubmittedFlag = true
		osk.open = false
		render.SetOSKActive(false)
		if oskCallback != nil {
			oskCallback("", "submit")
		}
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
	if osk.shift {
		osk.shift = false
	}
	if oskCallback != nil {
		oskCallback(ch, "type")
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
	glog.Infof("osk: update called, F13=%v F18=%v Enter=%v input=%v",
		ctx.Input != nil && ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13),
		ctx.Input != nil && ctx.Input.KeyCodeJustPressed(gpucontext.KeyF18),
		ctx.Input != nil && ctx.Input.KeyCodeJustPressed(gpucontext.KeyEnter),
		ctx.Input != nil)
	// The A button's companion mouse click must not reach the form under
	// the keyboard — clear both buttons every frame while the OSK is open.
	if ctx.Input != nil {
		ctx.Input.SetMouseButton(input.MouseButtonLeft, false)
		ctx.Input.SetMouseButton(input.MouseButtonRight, false)
	}
	// Use KeyCodeDown (held state) with latches: the platform's key press
	// edges (JustPressed) are cleared by EndFrame between the event dispatch
	// and game.Update on the fbdev backend, so edges never reach the game
	// layer. The held state persists across frames (A is held >= 80ms).
	// Independent if-blocks, NOT a switch: a switch's first-match-wins made
	// the "!A down" latch-reset case match every frame and the B/START/
	// SELECT cases below it unreachable dead code.
	floored := now.Sub(oskShared.actionAt) >= oskActionFloor
	if ctx.Input.KeyCodeDown(gpucontext.KeyF13) {
		if !oskShared.aLatch && floored {
			oskShared.aLatch = true
			oskShared.actionAt = now
			glog.Infof("osk: A down, injecting key=%q", osk.currentKey().label)
			osk.inject(ctx)
			return true
		}
	} else {
		oskShared.aLatch = false
	}
	if ctx.Input.KeyCodeDown(gpucontext.KeyF18) || ctx.Input.KeyCodeDown(gpucontext.KeyF22) {
		if !oskShared.bLatch {
			oskShared.bLatch = true
			// B/X: backspace.
			osk.backspace()
			return true
		}
	} else {
		oskShared.bLatch = false
	}
	if ctx.Input.KeyCodeDown(gpucontext.KeyEnter) {
		if !oskShared.enterLatch {
			oskShared.enterLatch = true
			// START: submit.
			osk.row, osk.col = len(osk.rows())-1, len(osk.rows()[len(osk.rows())-1])-1
			osk.inject(ctx)
			return true
		}
	} else {
		oskShared.enterLatch = false
	}
	if ctx.Input.KeyCodeDown(gpucontext.KeyPrintScreen) {
		if !oskShared.selLatch {
			oskShared.selLatch = true
			osk.symbols = !osk.symbols
			osk.row, osk.col = 0, 0
			return true
		}
	} else {
		oskShared.selLatch = false
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
		if now.Sub(oskShared.movedAt) >= oskNavFloor {
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
	h := float64(gridH + 24)
	x := (float64(bounds.Dx()) - w) / 2
	y := 8.0 // Top of the screen: clear of the login form below

	render.DrawRect(screen, x, y, w, h, c.panel)
	render.DrawRect(screen, x, y, w, 3, c.border)
	render.DrawRect(screen, x, y+3, w, 2, c.gold)

	// Grid. (No typed-text preview line: it echoed the password being
	// typed, in the clear, right above the login form.)
	gy := y + 12
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
		// Seed every key latch: the key that opened the keyboard (START)
		// is still held, and the latch handling would otherwise read that
		// as an immediate OK press in the same frame — the keyboard
		// flashed on and submitted the instant it opened. Each latch
		// clears as soon as its own key is released.
		oskShared.aLatch = true
		oskShared.bLatch = true
		oskShared.enterLatch = true
		oskShared.selLatch = true
		render.SetOSKActive(true)
		setupOSKHook()
		glog.Infof("osk: opened, hook installed")
	}
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

// setupOSKHook wires the platform's button dispatch directly to the OSK
// (the fbdev event pipeline's key state never reaches game.Update). The
// hook self-uninstalls once the keyboard closes — the world layer has no
// teardown call, and an installed-but-closed hook must never swallow
// game buttons. Returning false lets the press fall through to its
// normal key dispatch.
func setupOSKHook() {
	hooks.OSKButtonHook = func(button int) bool {
		osk := oskState()
		if !osk.open {
			hooks.OSKButtonHook = nil
			return false
		}
		switch button {
		case 0: // A: type the selected key
			osk.inject(client.Context{})
		case 1: // B: close
			osk.open = false
			render.SetOSKActive(false)
		case 2: // START: submit
			osk.row, osk.col = len(osk.rows())-1, len(osk.rows()[len(osk.rows())-1])-1
			osk.inject(client.Context{})
		case 3: // SELECT: toggle symbols
			osk.symbols = !osk.symbols
			osk.row, osk.col = 0, 0
		case 4: // X: backspace
			osk.backspace()
		}
		return true
	}
}

// backspace injects the ⌫ cell when the current page carries one (the
// symbols page does not — X is inert there until the ABC page returns).
func (osk *onScreenKeyboard) backspace() {
	last := osk.rows()[len(osk.rows())-1]
	for ci, key := range last {
		if key.special == "bksp" {
			osk.row, osk.col = len(osk.rows())-1, ci
			osk.inject(client.Context{})
			return
		}
	}
}

// teardownOSKHook removes the platform hook.
func teardownOSKHook() {
	hooks.OSKButtonHook = nil
}
