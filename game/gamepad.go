package game

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	gameui "github.com/kivutar/goro/ui"
	"github.com/kivutar/goro/world"
)

// Handheld controls (rg35xx and friends): the d-pad walks the character
// directly and the A button performs a context action — attack the nearest
// attackable actor, else pick up the nearest dropped item. The fbdev
// platform tags A presses with a KeyF13 edge next to the normal mouse
// click, so mouse-driven platforms are untouched.

const (
	// gamepadActionRange bounds the A-button attack/pickup search in tiles;
	// NPC talk is capped tighter (walking up to an NPC 9 tiles away just to
	// have the request drop server-side feels broken).
	gamepadActionRange  = 9.0
	gamepadNPCTalkRange = 5.0
	// gamepadWalkScreenLead is how far ahead of the player's screen
	// position a d-pad walk aims, in pixels. The world target comes from
	// projecting that point back through the camera, so the walk direction
	// is exactly the direction shown on screen regardless of the isometric
	// camera rotation or zoom.
	gamepadWalkScreenLead = 80.0
	// gamepadActionFloor paces A actions (attack/pickup/talk/confirm). The
	// platform already collapses the firmware's press/release chatter into a
	// single press per tap, so this floor no longer has to fight repeat — it
	// only keeps one press from firing two actions and stops the fastest
	// mashing from double-hitting the same target. The old 600ms made pickup
	// feel sticky; at 150ms the pickup pace follows the player's tapping.
	gamepadActionFloor = 150 * time.Millisecond
	// gamepadNavFloor paces d-pad menu navigation everywhere: one physical
	// tap posts exactly one edge now, so this is a comfort floor, not a
	// chatter filter (220ms felt sluggish after the platform fix).
	gamepadNavFloor = 160 * time.Millisecond
	// gamepadWalkRepeatInterval re-aims the held d-pad walk. The shared
	// held-click interval (500ms) is tuned for a mouse held on a far-away
	// spot; with the short 80px gamepad lead it ran the character in visible
	// bursts — walk, stop, walk. Re-aiming faster keeps the pace even.
	gamepadWalkRepeatInterval = 250 * time.Millisecond
)

// ensureHeldMenu lazily builds the handheld menu item actions.
func (m *WorldMode) heldMenuActivate(ctx client.Context) {
	entry := heldMenuEntries[m.heldMenuSel]
	glog.Infof("handheld menu activate item=%d %q", m.heldMenuSel, entry.label)
	entry.action(m, ctx)
}

// heldMenuEntries pairs each menu label with its action; the index order is
// the draw order. heldMenuItems below derives its labels.
var heldMenuEntries = []struct {
	label  string
	action func(*WorldMode, client.Context)
}{
	{"Sit/Stand", func(m *WorldMode, ctx client.Context) {
		// Toggles the local player posture; the server echoes the action
		// back and battle.go flips ctx.World.Player.Sitting.
		if ctx.Network != nil && ctx.Session != nil {
			action := network.ActionSitDown
			if ctx.World.Player.Sitting {
				action = network.ActionStandUp
			}
			if err := ctx.Network.SendActionRequest(ctx.Session.AccountID, action); err != nil {
				m.ui.console.AddErrorMessage("sit/stand failed")
			}
		}
	}},
	{"Items", func(m *WorldMode, ctx client.Context) { m.ui.inventoryBag.Toggle(ctx) }},
	{"Equipment", func(m *WorldMode, ctx client.Context) { m.ui.equipmentWindow.Toggle(ctx) }},
	{"Skills", func(m *WorldMode, ctx client.Context) { m.ui.skillWindow.Toggle(ctx) }},
	{"Stats", func(m *WorldMode, ctx client.Context) { m.ui.statsWindow.Toggle(ctx) }},
	{"Quests", func(m *WorldMode, ctx client.Context) { m.ui.questWindow.Toggle(ctx) }},
	{"World Map", func(m *WorldMode, ctx client.Context) {
		if err := m.ui.worldMap.Toggle(ctx); err != nil {
			m.ui.console.AddErrorMessage("%s", err.Error())
		}
	}},
	{"Chat", func(m *WorldMode, ctx client.Context) { m.ui.console.OpenForTyping(ctx) }},
	{"Hotbar", func(m *WorldMode, ctx client.Context) { m.ui.hotbar.shown = !m.ui.hotbar.shown }},
	{"Reset Camera", func(m *WorldMode, ctx client.Context) { m.camera.ResetView() }},
	{"Settings", func(m *WorldMode, ctx client.Context) { m.ui.settingsWindow.OpenWindow(ctx) }},
	{"Screenshot", func(m *WorldMode, ctx client.Context) {
		if path, err := ctx.RequestScreenshot(); err == nil {
			m.ui.console.AddSystemMessage("Screenshot: %s", path)
		} else {
			m.ui.console.AddErrorMessage("screenshot failed: %s", err.Error())
		}
	}},
	{"Exit Game", func(m *WorldMode, ctx client.Context) {
		if ctx.RequestQuit != nil {
			ctx.RequestQuit()
		}
	}},
}

var heldMenuItems = func() []string {
	labels := make([]string, len(heldMenuEntries))
	for i, entry := range heldMenuEntries {
		labels[i] = entry.label
	}
	return labels
}()

// closeActiveHandheldWindow closes whatever the handheld layer opened last
// (B button). Stacked windows close top-first: item descriptions and card
// artwork sit on top of the inventory, so walking the stack down means B
// never closes the inventory out from under a detail window — and never
// closes the whole stack in one press. It reports whether anything closed;
// on the plain screen the caller lets B use the hotbar instead.
func (m *WorldMode) closeActiveHandheldWindow(ctx client.Context) bool {
	if m.ui.itemWindows.CloseTopDescription(ctx) {
		return true
	}
	if m.ui.itemWindows.CloseTopIllustration(ctx) {
		return true
	}
	if m.ui.skillWindow.CloseTopDetail(ctx) {
		return true
	}
	closed := false
	switch {
	case m.ui.settingsWindow.IsOpen():
		m.ui.settingsWindow.Toggle(ctx)
		closed = true
	case m.ui.statsWindow.IsOpen():
		m.ui.statsWindow.Toggle(ctx)
		closed = true
	case m.ui.inventoryBag.IsOpen():
		m.ui.inventoryBag.Toggle(ctx)
		closed = true
	case m.ui.storageWindow.IsOpen():
		m.ui.storageWindow.SetOpen(false)
		closed = true
	case m.ui.equipmentWindow.IsOpen():
		m.ui.equipmentWindow.Toggle(ctx)
		closed = true
	case m.ui.skillWindow.IsOpen():
		m.ui.skillWindow.Toggle(ctx)
		closed = true
	case m.ui.questWindow.IsOpen():
		m.ui.questWindow.Toggle(ctx)
		closed = true
	case m.ui.worldMap.IsOpen():
		if err := m.ui.worldMap.Toggle(ctx); err != nil {
			m.ui.console.AddErrorMessage("%s", err.Error())
		}
		closed = true
	}
	return closed
}

// updateHeldMenuInput drives the direct-drawn MENU overlay: d-pad moves the
// selection (debounced — the polled d-pad bounces), A or START activates,
// SELECT closes. MENU itself toggles at the world level and must NOT be
// re-checked here — the same JustPressed edge that opened the menu would
// close it within the same frame (observed as open=true logs with nothing
// rendered and walking continuing underneath).
func (m *WorldMode) updateHeldMenuInput(ctx client.Context, now time.Time) {
	if ctx.Input.JustPressed(input.KeyEscape) || ctx.Input.KeyCodeJustPressed(gpucontext.KeyPrintScreen) {
		m.setHeldMenuOpen(ctx, false)
		return
	}
	if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13) || ctx.Input.JustPressed(input.KeyEnter) {
		m.setHeldMenuOpen(ctx, false)
		m.heldMenuActivatedAt = now
		m.heldMenuActivate(ctx)
		return
	}
	dx, dy := 0, 0
	if ctx.Input.JustPressed(input.KeyArrowUp) {
		dy--
	}
	if ctx.Input.JustPressed(input.KeyArrowDown) {
		dy++
	}
	if ctx.Input.JustPressed(input.KeyArrowLeft) {
		dx--
	}
	if ctx.Input.JustPressed(input.KeyArrowRight) {
		dx++
	}
	if dx == 0 && dy == 0 {
		return
	}
	if now.Sub(m.heldMenuMovedAt) < gamepadNavFloor {
		return
	}
	m.heldMenuMovedAt = now
	step := dy
	if step == 0 {
		step = dx
	}
	// Wrap-around: past either end the selection loops to the other side.
	count := len(heldMenuItems)
	m.heldMenuSel = ((m.heldMenuSel+step)%count + count) % count
}

// drawHeldMenu renders the overlay with immediate primitives only —
// DrawRect + DrawOutlinedTextAt draw straight into the frame (the volume
// HUD path). The earlier UI-label variants appended to compositor lists
// that never flushed for game-drawn content, so nothing appeared.
func (m *WorldMode) drawHeldMenu(screen *render.Frame) {
	if !m.heldMenuOpen || screen == nil {
		return
	}
	const itemH = 18
	const pad = 10
	const menuW = 190
	titleH := 22
	menuH := titleH + pad + len(heldMenuItems)*itemH + pad
	bounds := screen.Bounds()
	x := (float64(bounds.Dx()) - menuW) / 2
	// Below the MENU companion panels: the character info window occupies
	// the top-left ~324x134, and a vertically centered menu covered it on
	// the small screen.
	y := (float64(bounds.Dy()) - float64(menuH)) / 2
	if top := 12 + 134 + 10; y < float64(top) {
		y = float64(top)
	}
	if bottom := float64(bounds.Dy()) - float64(menuH) - 8; y > bottom {
		y = math.Max(8, bottom)
	}

	// Panel and title bar.
	render.DrawRect(screen, x, y, menuW, float64(menuH), color.RGBA{R: 24, G: 20, B: 34, A: 235})
	render.DrawRect(screen, x, y, menuW, float64(titleH), color.RGBA{R: 60, G: 48, B: 84, A: 245})
	render.DrawRect(screen, x, y+float64(titleH-2), menuW, 2, color.RGBA{R: 214, G: 178, B: 92, A: 255})
	render.DrawOutlinedTextAt(screen, "MENU", int(x)+pad, int(y)+4, color.RGBA{R: 250, G: 240, B: 210, A: 255}, color.RGBA{A: 200})

	// Items; the selection inverts its row.
	for i, label := range heldMenuItems {
		rowY := y + float64(titleH+pad+i*itemH)
		if i == m.heldMenuSel {
			render.DrawRect(screen, x+4, rowY-1, menuW-8, float64(itemH), color.RGBA{R: 214, G: 178, B: 92, A: 235})
			render.DrawOutlinedTextAt(screen, label, int(x)+pad, int(rowY)+2, color.RGBA{R: 30, G: 22, B: 12, A: 255}, color.RGBA{A: 0})
			continue
		}
		render.DrawOutlinedTextAt(screen, label, int(x)+pad, int(rowY)+2, color.RGBA{R: 235, G: 232, B: 240, A: 255}, color.RGBA{A: 200})
	}
}

// updateGamepadControls runs the handheld input layer each world frame. It
// returns true when the A press was consumed as a gamepad action, so the
// same-frame mouse click (the platform emits both) must not also walk.
func (m *WorldMode) updateGamepadControls(ctx client.Context, pointerBlocked bool, now time.Time) bool {
	if ctx.Input == nil || ctx.World == nil {
		return false
	}
	// While an NPC dialog is open, A confirms it — checked before the
	// keyboard-blocked gate below, which reports the open dialog itself
	// and would swallow the press. Floored: the firmware's hold-repeat
	// pairs otherwise advanced several dialog pages per held press.
	if m.ui.npcDialog.IsOpen() && ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13) {
		if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
			m.gamepadActionAt = now
			m.ui.npcDialog.Confirm(ctx)
		}
		return true
	}
	// The direct-drawn MENU overlay owns every button while open; B closes.
	if m.heldMenuOpen {
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF18) {
			m.setHeldMenuOpen(ctx, false)
			return true
		}
		m.updateHeldMenuInput(ctx, now)
		return true
	}
	// L2/R2 step the hotbar's active slot. They work everywhere — windows
	// open or not, bar shown or hidden.
	if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF23) {
		m.ui.hotbar.cycle(-1)
		return true
	}
	if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF24) {
		m.ui.hotbar.cycle(1)
		return true
	}
	// The large map overlay owns every button while shown; X or B dismisses
	// it. The small corner thumbnail does not: play continues around it.
	if m.mapOverlay == 2 {
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF22) || ctx.Input.KeyCodeJustPressed(gpucontext.KeyF18) {
			m.mapOverlay = 0
		}
		return true
	}
	// The storage deposit picker owns every button while open (A confirms,
	// B cancels, d-pad steps the amount).
	if m.updateGamepadStorageDeposit(ctx, now) {
		return true
	}
	// The on-screen keyboard owns every button while open (d-pad navigates,
	// A types, B backspaces, START submits, SELECT toggles symbols).
	if updateGamepadOSK(ctx, now) {
		return true
	}
	// START opens the on-screen keyboard when a text field has focus (the
	// chat console, or any modal input).
	if ctx.Input.JustPressed(input.KeyEnter) && (m.ui.console.Active() || m.ui.keyboardInputBlocked(ctx)) {
		tryOpenOSK(true)
		return true
	}
	// B closes the active (topmost relevant) window. Checked before the
	// walk layer so B never walks or attacks while dismissing a window. On
	// the plain screen, with nothing to close, B uses the hotbar's active
	// slot instead.
	if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF18) {
		if m.closeActiveHandheldWindow(ctx) {
			return true
		}
		m.useHotbarActive(ctx)
		return true
	}
	// The stats window owns the d-pad while open: up/down (or left/right)
	// move the stat selection, A raises it, walking is suspended.
	if m.ui.statsWindow.IsOpen() {
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13) {
			if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
				m.gamepadActionAt = now
				m.ui.statsWindow.GamepadConfirm(ctx)
			}
			return true
		}
		delta := 0
		if ctx.Input.JustPressed(input.KeyArrowUp) || ctx.Input.JustPressed(input.KeyArrowLeft) {
			delta--
		}
		if ctx.Input.JustPressed(input.KeyArrowDown) || ctx.Input.JustPressed(input.KeyArrowRight) {
			delta++
		}
		if delta != 0 && now.Sub(m.statsSelMovedAt) >= gamepadNavFloor {
			m.statsSelMovedAt = now
			m.ui.statsWindow.GamepadNavigate(ctx, delta)
		}
		return true
	}
	// The inventory owns every handheld button while open: d-pad moves the
	// cell selection, A activates the item (use/equip, the double-click
	// path), Y opens its description, L1/R1 cycle the Item/Equip/Etc
	// tabs. While a description window sits on top, the d-pad drives its
	// card slots (when it has any) and X opens the selected card; B is
	// handled by the branch above and closes the stack top-down.
	if m.ui.inventoryBag.IsOpen() {
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF20) {
			m.ui.inventoryBag.GamepadTab(ctx, -1)
			return true
		}
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF21) {
			m.ui.inventoryBag.GamepadTab(ctx, 1)
			return true
		}
		// Y always opens the selected inventory item's description —
		// pressing it again on another item stacks another window, the
		// way the upstream client's right-click does.
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF19) {
			if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
				m.gamepadActionAt = now
				m.ui.inventoryBag.GamepadInfo(ctx)
			}
			return true
		}
		// X inspects the card in the selected slot of the top-most
		// description window; with no description open, X on the Item/Equip
		// tabs fills the next hotbar slot with the selected item (the Etc
		// tab keeps X inert).
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF22) {
			consumed := false
			if m.ui.itemWindows.HasOpenDescriptions() {
				if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
					m.gamepadActionAt = now
					m.ui.itemWindows.GamepadOpenSelectedCard(ctx)
				}
				consumed = true
			} else if m.ui.inventoryBag.GamepadTabAllowsHotbar() {
				if item, ok := m.ui.inventoryBag.GamepadSelectedItem(ctx); ok {
					m.ui.hotbar.addItem(item)
					slot := ((m.ui.hotbar.fill + hotbarSlotCount - 1) % hotbarSlotCount) + 1
					render.ShowScreenNotice(fmt.Sprintf("Hotbar %d: %s", slot, gameui.ItemDisplayName(ctx.Resources, item)))
				}
				consumed = true
			}
			if consumed {
				return true
			}
		}
		// SELECT (its own PrintScreen key) is the dedicated hotbar-fill
		// button: it adds the selected item on the Item/Equip tabs even
		// while a description window is open.
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyPrintScreen) && m.ui.inventoryBag.GamepadTabAllowsHotbar() {
			if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
				m.gamepadActionAt = now
				if item, ok := m.ui.inventoryBag.GamepadSelectedItem(ctx); ok {
					m.ui.hotbar.addItem(item)
					slot := ((m.ui.hotbar.fill + hotbarSlotCount - 1) % hotbarSlotCount) + 1
					render.ShowScreenNotice(fmt.Sprintf("Hotbar %d: %s", slot, gameui.ItemDisplayName(ctx.Resources, item)))
				}
			}
			return true
		}
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13) {
			if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
				m.gamepadActionAt = now
				if m.ui.storageWindow.IsOpen() {
					// With storage open, A deposits the selected stack
					// instead of using the item; stacks larger than one go
					// through the amount picker.
					if item, ok := m.ui.inventoryBag.GamepadSelectedItem(ctx); ok {
						if item.Amount > 1 {
							m.storageDeposit.begin(item.Index, item.ItemID, gameui.ItemDisplayName(ctx.Resources, item), int(item.Amount))
						} else if ctx.Network != nil {
							if err := ctx.Network.SendMoveToStorage(item.Index, 1); err != nil {
								m.ui.console.AddErrorMessage("Deposit failed.")
							} else {
								render.ShowScreenNotice(fmt.Sprintf("Deposited 1 x %s", gameui.ItemDisplayName(ctx.Resources, item)))
							}
						}
					}
				} else {
					m.ui.inventoryBag.GamepadActivate(ctx)
				}
			}
			return true
		}
		dx := 0
		dy := 0
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
			if now.Sub(m.invSelMovedAt) >= gamepadNavFloor {
				m.invSelMovedAt = now
				// A description with card slots on top takes the d-pad
				// for its slot row (left/right, up/down both work);
				// anything else routes to the inventory grid beneath.
				if m.ui.itemWindows.HasOpenDescriptions() &&
					m.ui.itemWindows.GamepadCardNavigate(ctx, dx+dy) {
					m.invSelMovedAt = now
				} else {
					m.ui.inventoryBag.GamepadNavigate(ctx, dx, dy)
				}
			}
			return true
		}
		return true
	}
	// The equipment window mirrors the inventory scheme: the d-pad walks
	// the equip slots, A takes the item off, Y opens its description.
	// While such a description with card slots sits on top, the d-pad
	// drives its slot row and X opens the selected card.
	if m.ui.equipmentWindow.IsOpen() {
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF19) {
			if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
				m.gamepadActionAt = now
				m.ui.equipmentWindow.GamepadInfo(ctx)
			}
			return true
		}
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF22) && m.ui.itemWindows.HasOpenDescriptions() {
			if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
				m.gamepadActionAt = now
				m.ui.itemWindows.GamepadOpenSelectedCard(ctx)
			}
			return true
		}
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13) {
			if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
				m.gamepadActionAt = now
				m.ui.equipmentWindow.GamepadActivate(ctx)
			}
			return true
		}
		dx := 0
		dy := 0
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
			if now.Sub(m.invSelMovedAt) >= gamepadNavFloor {
				m.invSelMovedAt = now
				if m.ui.itemWindows.HasOpenDescriptions() &&
					m.ui.itemWindows.GamepadCardNavigate(ctx, dx+dy) {
					m.invSelMovedAt = now
				} else {
					m.ui.equipmentWindow.GamepadNavigate(ctx, dx, dy)
				}
			}
			return true
		}
		return true
	}
	// The skill tree follows the same handheld scheme: d-pad walks the skill
	// cells (table rows in list mode) and then the footer's Reset/Confirm
	// buttons, A stages a level-up (or presses the footer button), Y opens
	// the skill detail window, L1/R1 cycle the class tabs, X flips between
	// the table and tree views.
	if m.ui.skillWindow.IsOpen() {
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF20) {
			m.ui.skillWindow.GamepadTab(ctx, -1)
			return true
		}
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF21) {
			m.ui.skillWindow.GamepadTab(ctx, 1)
			return true
		}
		// SELECT (its own PrintScreen key) fills the next hotbar slot with
		// the selected skill — the dedicated hotbar-fill button, matching
		// the inventory. START and X both flip the table/tree view.
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyPrintScreen) {
			if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
				m.gamepadActionAt = now
				if skill, ok := m.ui.skillWindow.GamepadSelectedSkill(ctx); ok {
					m.ui.hotbar.addSkill(skill.ID)
					slot := ((m.ui.hotbar.fill + hotbarSlotCount - 1) % hotbarSlotCount) + 1
					render.ShowScreenNotice(fmt.Sprintf("Hotbar %d: %s", slot, skillLabel(skill)))
				}
			}
			return true
		}
		if ctx.Input.JustPressed(input.KeyEnter) || ctx.Input.KeyCodeJustPressed(gpucontext.KeyF22) {
			if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
				m.gamepadActionAt = now
				m.ui.skillWindow.GamepadToggleMode(ctx)
			}
			return true
		}
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF19) {
			if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
				m.gamepadActionAt = now
				m.ui.skillWindow.GamepadInfo(ctx)
			}
			return true
		}
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13) {
			if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
				m.gamepadActionAt = now
				m.ui.skillWindow.GamepadActivate(ctx)
			}
			return true
		}
		dx := 0
		dy := 0
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
			if now.Sub(m.invSelMovedAt) >= gamepadNavFloor {
				m.invSelMovedAt = now
				// An open detail window with overflowing text takes the
				// d-pad for scrolling; at the scroll edges (and whenever the
				// text fits) the press falls through to the skill navigation.
				if dy != 0 && m.ui.skillWindow.GamepadScrollDetail(dy) {
					m.invSelMovedAt = now
				} else {
					m.ui.skillWindow.GamepadNavigate(ctx, dx, dy)
				}
			}
			return true
		}
		return true
	}
	// The settings window rows: d-pad up/down walks the entries, left/right
	// adjusts the selected value, A toggles booleans (or steps the value up).
	if m.ui.settingsWindow.IsOpen() {
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13) {
			if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
				m.gamepadActionAt = now
				m.ui.settingsWindow.GamepadActivate(ctx)
			}
			return true
		}
		dx := 0
		dy := 0
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
			if now.Sub(m.invSelMovedAt) >= gamepadNavFloor {
				m.invSelMovedAt = now
				if dy != 0 {
					m.ui.settingsWindow.GamepadNavigate(ctx, dy)
				}
				if dx != 0 {
					m.ui.settingsWindow.GamepadAdjust(ctx, dx)
				}
			}
			return true
		}
		return true
	}
	if m.ui.console.Active() || m.ui.keyboardInputBlocked(ctx) {
		return false
	}
	if m.pendingSkill.skill.ID != 0 || m.pendingPetCapture.active {
		return false
	}
	// Plain-screen shortcuts: Y opens the inventory (the window branches
	// above have already consumed Y while any window was open), X cycles
	// the frameless map overlay: corner thumbnail, large map, hidden.
	if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF19) {
		if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
			m.gamepadActionAt = now
			m.ui.inventoryBag.Toggle(ctx)
		}
		return true
	}
	if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF22) {
		if now.Sub(m.gamepadActionAt) >= gamepadActionFloor {
			m.gamepadActionAt = now
			m.mapOverlay = (m.mapOverlay + 1) % 3
		}
		return true
	}
	if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13) && !pointerBlocked {
		// The same A press that activated a handheld-menu item must not
		// also fall into the combat chain a frame later (observed:
		// screenshot then attack from one press).
		if now.Sub(m.heldMenuActivatedAt) < 300*time.Millisecond {
			return true
		}
		// Global A action floor: holding/tapping A rapidly fired
		// attack/pickup back-to-back (log: four pickups in one second).
		if !m.gamepadActionAt.IsZero() && now.Sub(m.gamepadActionAt) < gamepadActionFloor {
			return true
		}
		m.gamepadActionAt = now
		if m.gamepadPrimaryAction(ctx, now) {
			return true
		}
		// Nothing in range: consume the press anyway. Falling through would
		// click wherever the stale pointer sits (top-left corner), making A
		// walk the player northwest for no visible reason.
		return true
	}
	m.updateGamepadWalk(ctx, pointerBlocked, now)
	return false
}

// gamepadPrimaryAction attacks the nearest attackable actor in range, else
// requests pickup of the nearest floor item in range.
func (m *WorldMode) gamepadPrimaryAction(ctx client.Context, now time.Time) bool {
	playerX, playerY := currentPlayerCell(ctx, now)

	bestDistance := math.Inf(1)
	var bestActor world.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := m.actorDeaths[actor.ID]; dead {
			continue
		}
		if !actorCanBeAttackClicked(ctx, actor) {
			continue
		}
		actorX, actorY := actorRenderPosition(actor, now)
		distance := math.Hypot(actorX-float64(playerX), actorY-float64(playerY))
		if distance < bestDistance {
			bestDistance = distance
			bestActor = actor
		}
	}
	if bestDistance <= gamepadActionRange {
		glog.Infof("gamepad a attack target id=%d name=%q distance=%.1f player=%d,%d", bestActor.ID, bestActor.Name, bestDistance, playerX, playerY)
		m.requestAttack(ctx, bestActor, "gamepad a")
		return true
	}

	bestItemDistance := math.Inf(1)
	var bestItem world.FloorItem
	for _, item := range ctx.World.Items {
		distance := math.Hypot(float64(item.X)-float64(playerX), float64(item.Y)-float64(playerY))
		if distance < bestItemDistance {
			bestItemDistance = distance
			bestItem = item
		}
	}
	if bestItemDistance <= gamepadActionRange {
		glog.Infof("gamepad a pickup target id=%d item_id=%d distance=%.1f player=%d,%d", bestItem.ID, bestItem.ItemID, bestItemDistance, playerX, playerY)
		m.clearLockedAttack()
		m.clearAttackFocus()
		m.requestPickup(ctx, bestItem, "gamepad a")
		return true
	}

	// No combat and no loot: talk to the nearest NPC instead.
	bestTalkDistance := math.Inf(1)
	var bestTalkActor world.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := m.actorDeaths[actor.ID]; dead {
			continue
		}
		if !cursorActorCanTalk(actor) {
			continue
		}
		actorX, actorY := actorRenderPosition(actor, now)
		distance := math.Hypot(actorX-float64(playerX), actorY-float64(playerY))
		if distance < bestTalkDistance {
			bestTalkDistance = distance
			bestTalkActor = actor
		}
	}
	if bestTalkDistance <= gamepadNPCTalkRange {
		glog.Infof("gamepad a npc talk target id=%d name=%q distance=%.1f player=%d,%d", bestTalkActor.ID, bestTalkActor.Name, bestTalkDistance, playerX, playerY)
		m.requestNPCTalk(ctx, bestTalkActor, "gamepad a")
		return true
	}
	return false
}

// updateGamepadWalk issues walk requests toward the held d-pad direction.
// The direction is interpreted in screen space and projected through the
// camera, so "up" always walks toward the top of the screen. Two held axes
// combine into a diagonal. While MENU is held (KeyF17), the d-pad steers
// the camera instead — up/down zoom, left/right rotate — and walking stops.
func (m *WorldMode) updateGamepadWalk(ctx client.Context, pointerBlocked bool, now time.Time) {
	if pointerBlocked {
		return
	}
	dx, dy := 0, 0
	if ctx.Input.Pressed(input.KeyArrowLeft) {
		dx--
	}
	if ctx.Input.Pressed(input.KeyArrowRight) {
		dx++
	}
	if ctx.Input.Pressed(input.KeyArrowUp) {
		dy--
	}
	if ctx.Input.Pressed(input.KeyArrowDown) {
		dy++
	}
	if dx == 0 && dy == 0 {
		m.gamepadDirLogged = false
		return
	}
	if ctx.Input.KeyCodeDown(gpucontext.KeyF17) {
		// Rate-limited fine camera steps: the held d-pad repeats through
		// the game loop, so gate to a gentle per-tenth-second nudge.
		if now.Sub(m.gamepadCameraAt) < 100*time.Millisecond {
			return
		}
		m.gamepadCameraAt = now
		const zoomFactor = 1.06
		const rotateAngle = 8.0
		if !cameraZoomLockedForMap(ctx) {
			if dy < 0 {
				m.camera.ZoomBy(zoomFactor)
			} else if dy > 0 {
				m.camera.ZoomBy(1 / zoomFactor)
			}
		}
		if !cameraRotationLockedForMap(ctx) {
			if dx < 0 {
				m.camera.Rotate(-rotateAngle)
			} else if dx > 0 {
				m.camera.Rotate(rotateAngle)
			}
		}
		m.gamepadDirLogged = false
		return
	}
	// One info line per held direction: proves on the device that the d-pad
	// reaches the world layer (or not) without debug-level logging.
	if !m.gamepadDirLogged {
		m.gamepadDirLogged = true
		glog.Infof("gamepad walk direction active dir=%d,%d", dx, dy)
	}
	if !m.walkReady(now) || (!m.nextHeldWalkAt.IsZero() && now.Before(m.nextHeldWalkAt)) {
		return
	}
	m.nextHeldWalkAt = now.Add(gamepadWalkRepeatInterval)

	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	playerX, playerY := currentPlayerCell(ctx, now)
	terrainZ := terrainHeightAt(ctx.World, float64(playerX), float64(playerY))
	point := projection.Project(cellCenter(float64(playerX)), cellCenter(float64(playerY)), terrainZ)
	screenX := math.Min(math.Max(float64(point.x)+float64(dx)*gamepadWalkScreenLead, 0), float64(screenW-1))
	screenY := math.Min(math.Max(float64(point.y)+float64(dy)*gamepadWalkScreenLead, 0), float64(screenH-1))
	targetX, targetY, ok := clickedWalkTarget(ctx, projection, int(screenX), int(screenY))
	if !ok || playerAtWalkTarget(ctx.World.Player, targetX, targetY, now) {
		return
	}
	// Walking elsewhere is the cancel gesture for in-flight actions, same
	// as a ground click: stop chasing a pickup target and drop the attack
	// intent so combat auto-repeat ends.
	m.pendingPickup = pickupIntent{}
	m.cancelAttackIntent()
	if shouldUseTurnOnlyGroundClick(ctx) {
		m.requestChangeDirection(ctx, targetX, targetY, "gamepad walk")
		return
	}
	m.requestWalk(ctx, targetX, targetY, "gamepad walk")
}
