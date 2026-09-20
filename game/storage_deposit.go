package game

import (
	"fmt"
	"image/color"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
)

// The storage deposit dialog: A on an inventory item while the storage
// window is open deposits that stack instead of using it. Stacks larger
// than one open this picker — up/down step by 1, left/right by 10, A
// confirms, B cancels. Direct-drawn like the handheld MENU overlay.

type storageDepositDialog struct {
	open      bool
	withdraw  bool // true = withdraw from storage, false = deposit
	itemIndex uint16
	itemID    uint16
	itemName  string
	amount    int
	max       int
}

var storageDepositColors = struct {
	panel   color.RGBA
	border  color.RGBA
	gold    color.RGBA
	text    color.RGBA
	muted   color.RGBA
	outline color.RGBA
}{
	panel:   color.RGBA{R: 24, G: 20, B: 34, A: 235},
	border:  color.RGBA{R: 60, G: 48, B: 84, A: 245},
	gold:    color.RGBA{R: 214, G: 178, B: 92, A: 255},
	text:    color.RGBA{R: 240, G: 240, B: 245, A: 255},
	muted:   color.RGBA{R: 170, G: 178, B: 190, A: 220},
	outline: color.RGBA{R: 10, G: 12, B: 16, A: 210},
}

// begin starts a deposit for the given stack.
func (d *storageDepositDialog) begin(itemIndex, itemID uint16, name string, max int) {
	if max < 1 {
		max = 1
	}
	d.open = true
	d.itemIndex = itemIndex
	d.itemID = itemID
	d.itemName = name
	d.amount = 1
	d.max = max
}

func (d *storageDepositDialog) adjust(delta int) {
	if !d.open {
		return
	}
	d.amount += delta
	if d.amount < 1 {
		d.amount = 1
	}
	if d.amount > d.max {
		d.amount = d.max
	}
}

// updateGamepadStorageDeposit drives the open dialog. It reports whether
// the dialog consumed the press (callers skip everything else while open).
func (m *WorldMode) updateGamepadStorageDeposit(ctx client.Context, now time.Time) bool {
	dialog := &m.storageDeposit
	if !dialog.open {
		return false
	}
	switch {
	case ctx.Input.KeyCodeJustPressed(gpucontext.KeyF18):
		dialog.open = false
	case ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13):
		if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
			m.gamepadActionAt = now
			dialog.open = false
			if ctx.Network != nil {
				var err error
				var verb string
				if dialog.withdraw {
					err = ctx.Network.SendMoveFromStorage(dialog.itemIndex, uint32(dialog.amount))
					verb = "Withdrew"
				} else {
					err = ctx.Network.SendMoveToStorage(dialog.itemIndex, uint32(dialog.amount))
					verb = "Deposited"
				}
				if err != nil {
					glog.Warnf("storage %s failed: %v", verb, err)
					m.ui.console.AddErrorMessage("%s failed.", verb)
				} else {
					render.ShowScreenNotice(fmt.Sprintf("%s %d x %s", verb, dialog.amount, dialog.itemName))
				}
			}
		}
		return true
	case ctx.Input.JustPressed(input.KeyArrowUp):
		if now.Sub(m.invSelMovedAt) >= gamepadNavFloor {
			m.invSelMovedAt = now
			dialog.adjust(1)
		}
	case ctx.Input.JustPressed(input.KeyArrowDown):
		if now.Sub(m.invSelMovedAt) >= gamepadNavFloor {
			m.invSelMovedAt = now
			dialog.adjust(-1)
		}
	case ctx.Input.JustPressed(input.KeyArrowRight):
		if now.Sub(m.invSelMovedAt) >= gamepadNavFloor {
			m.invSelMovedAt = now
			dialog.adjust(10)
		}
	case ctx.Input.JustPressed(input.KeyArrowLeft):
		if now.Sub(m.invSelMovedAt) >= gamepadNavFloor {
			m.invSelMovedAt = now
			dialog.adjust(-10)
		}
	}
	return true
}

// drawStorageDeposit renders the dialog centered on screen.
func (m *WorldMode) drawStorageDeposit(screen *render.Frame) {
	dialog := &m.storageDeposit
	if screen == nil || !dialog.open {
		return
	}
	bounds := screen.Bounds()
	const w = 260.0
	const h = 116.0
	x := (float64(bounds.Dx()) - w) / 2
	y := (float64(bounds.Dy()) - h) / 2
	c := storageDepositColors
	render.DrawRect(screen, x, y, w, h, c.panel)
	render.DrawRect(screen, x, y, w, 22, c.border)
	render.DrawRect(screen, x, y+20, w, 2, c.gold)
	title := "Deposit to Storage"
	if dialog.withdraw {
		title = "Withdraw from Storage"
	}
	render.DrawUIOutlinedTextAt(screen, title, x+10, y+4, c.gold, c.outline)

	render.DrawUIOutlinedTextAt(screen, trimRunesGame(dialog.itemName, 24), x+10, y+32, c.text, c.outline)
	render.DrawUIOutlinedTextAt(screen, fmt.Sprintf("%d / %d", dialog.amount, dialog.max), x+10, y+52, c.text, c.outline)

	// Stepper rail: minus 1 / minus 10 | plus 10 / plus 1, mirroring the
	// d-pad layout so the dialog doubles as a visual legend.
	railY := y + 74
	render.DrawUIOutlinedTextAt(screen, "-1", x+w-116, railY, c.muted, c.outline)
	render.DrawUIOutlinedTextAt(screen, "-10", x+w-156, railY, c.muted, c.outline)
	render.DrawUIOutlinedTextAt(screen, "+1", x+64, railY, c.muted, c.outline)
	render.DrawUIOutlinedTextAt(screen, "+10", x+14, railY, c.muted, c.outline)
	render.DrawUIOutlinedTextAt(screen, "A: OK  B: Cancel", x+10, y+94, c.muted, c.outline)
}

// trimRunesGame clamps a string for the overlay labels.
func trimRunesGame(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes])
}

// updateGamepadStorageWithdraw drives the withdraw amount picker using the
// same dialog mechanics as deposits.
func (m *WorldMode) updateGamepadStorageWithdraw(ctx client.Context, now time.Time) bool {
	// Swap the withdraw dialog into the deposit slot, run the deposit
	// handler (same controls), then swap back.
	m.storageDeposit, m.storageWithdraw = m.storageWithdraw, m.storageDeposit
	result := m.updateGamepadStorageDeposit(ctx, now)
	m.storageDeposit, m.storageWithdraw = m.storageWithdraw, m.storageDeposit
	return result
}
