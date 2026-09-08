package ui

import (
	"image/color"
	"strings"



	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
)

// The basic menu (Status/Option/Items/Equip/Skill/Map/Comm/Friend) is drawn
// straight into the game frame from cached surfaces and GPU-cached text,
// matching the character HUD: no retained widget tree, no re-raster when
// anything around it changes. Buttons are hit-tested by hand.

const (
	basicMenuX         = windowScreenMargin
	basicMenuY         = characterWindowY + characterWindowHeight + basicMenuFollowGap
	basicMenuFollowGap = 6
	basicMenuCols      = 4
	basicMenuRows      = 2
	basicMenuButtonW   = 72
	basicMenuButtonH   = 24
	basicMenuGapX      = 6
	basicMenuGapY      = 5
	basicMenuPad       = 8

	basicMenuCloseSize = 17
)

var (
	basicMenuBackground = color.RGBA{R: 14, G: 18, B: 24, A: 189}
	basicMenuBorder     = color.RGBA{R: 180, G: 198, B: 218, A: 94}
	basicMenuButtonBack = color.RGBA{R: 255, G: 255, B: 255, A: 26}
	basicMenuText       = color.RGBA{R: 235, G: 242, B: 250, A: 255}
)

type BasicMenu struct {
	callbacks  BasicMenuCallbacks
	open       bool
	dismissed  bool
	x          int
	y          int
	width      int
	height     int
	pressedKey    string
	webSyncedOpen bool
}

type BasicMenuCallbacks struct {
	OnStatus func()
	OnOption func()
	OnItems  func()
	OnEquip  func()
	OnSkill  func()
	OnMap    func()
	OnComm   func()
	OnFriend func()
}

type basicMenuButton struct {
	key   string
	label string
}

var basicMenuButtons = []basicMenuButton{
	{key: "status", label: "Status"},
	{key: "option", label: "Option"},
	{key: "items", label: "Items"},
	{key: "equip", label: "Equip"},
	{key: "skill", label: "Skill"},
	{key: "map", label: "Map"},
	{key: "comm", label: "Comm"},
	{key: "friend", label: "Friend"},
}

func (m *BasicMenu) IsOpen() bool {
	return m != nil && m.open
}

// Close hides the menu until its edge tab is tapped again.
func (m *BasicMenu) Close() {
	if m == nil {
		return
	}
	m.open = false
	m.dismissed = true
}

// DrainWebActions services DOM menu taps unconditionally: the canvas menu
// is gated on keyboardInputBlocked (modal safety), but a tap on a real DOM
// button is always an intentional act and must not be swallowed because a
// modal happened to be open.
func (m *BasicMenu) DrainWebActions(ctx client.Context, callbacks BasicMenuCallbacks) {
	if m == nil || !hudWebEnabled() {
		return
	}
	m.callbacks = callbacks
	for _, action := range hudWebDrainActions() {
		if !strings.HasPrefix(action, "menu:") {
			continue
		}
		switch key := strings.TrimPrefix(action, "menu:"); key {
		case "tab":
			m.open = true
			m.dismissed = false
			menuWebSync(true)
		case "close":
			m.Close()
			menuWebSync(false)
		default:
			m.invoke(key)
		}
	}
}

func (m *BasicMenu) Rebind(_ client.Context, callbacks BasicMenuCallbacks) {
	m.callbacks = callbacks
}

func basicMenuBounds() (int, int, int, int) {
	w, h := basicMenuSize()
	return basicMenuX, basicMenuY, w, h
}

func basicMenuSize() (int, int) {
	w := basicMenuPad*2 + basicMenuCols*basicMenuButtonW + (basicMenuCols-1)*basicMenuGapX
	h := basicMenuPad*2 + basicMenuRows*basicMenuButtonH + (basicMenuRows-1)*basicMenuGapY
	return w, h
}

// basicMenuEdgeTabRect is the flush-left tab shown while the menu is
// dismissed; tapping it reopens the window.
func basicMenuEdgeTabRect() (int, int, int, int) {
	return 0, 8 + hudEdgeTabH + 6, hudEdgeTabW, hudEdgeTabH
}

func basicMenuEdgeTabHit(mouseX, mouseY int) bool {
	tx, ty, tw, th := basicMenuEdgeTabRect()
	return pointInRect(mouseX, mouseY, tx, ty, tw, th)
}

// buttonRect returns the on-screen rect of the button at index i.
func (m *BasicMenu) buttonRect(i int) (int, int, int, int) {
	row, col := i/basicMenuCols, i%basicMenuCols
	return m.x + basicMenuPad + col*(basicMenuButtonW+basicMenuGapX),
		m.y + basicMenuPad + row*(basicMenuButtonH+basicMenuGapY),
		basicMenuButtonW, basicMenuButtonH
}

func (m *BasicMenu) closeRect() (int, int, int, int) {
	return m.x + m.width - basicMenuCloseSize - 5, m.y + 5, basicMenuCloseSize, basicMenuCloseSize
}

func (m *BasicMenu) Update(ctx client.Context, callbacks BasicMenuCallbacks) bool {
	if m == nil {
		return false
	}
	m.callbacks = callbacks
	if hudWebEnabled() {
		hudWebInstallHooks()
		if !m.open && !m.dismissed {
			m.open = true
		}
		if m.open != m.webSyncedOpen {
			m.webSyncedOpen = m.open
			menuWebSync(m.open)
		}
		return false
	}
	if ctx.Input == nil {
		return false
	}
	if m.width == 0 {
		m.width, m.height = basicMenuSize()
		m.x, m.y = basicMenuX, basicMenuY
	}
	if !m.open {
		if !m.dismissed {
			m.open = true
		} else {
			if ctx.Input.MouseJustPressed(input.MouseButtonLeft) && basicMenuEdgeTabHit(ctx.Input.MouseX, ctx.Input.MouseY) {
				m.open = true
				m.dismissed = false
				return true
			}
			return false
		}
	}
	mouseX, mouseY := ctx.Input.MouseX, ctx.Input.MouseY
	if !ctx.Input.MouseJustPressed(input.MouseButtonLeft) {
		return false
	}
	if !pointInRect(mouseX, mouseY, m.x, m.y, m.width, m.height) {
		return false
	}
	if cx, cy, cw, ch := m.closeRect(); pointInRect(mouseX, mouseY, cx, cy, cw, ch) {
		m.Close()
		return true
	}
	for i, button := range basicMenuButtons {
		if bx, by, bw, bh := m.buttonRect(i); pointInRect(mouseX, mouseY, bx, by, bw, bh) {
			m.invoke(button.key)
			return true
		}
	}
	return true
}

// FollowCharacterWindow keeps the menu attached below the character HUD.
func (m *BasicMenu) FollowCharacterWindow(_ client.Context, character *CharacterWindow) {
	if m == nil || character == nil || !character.IsOpen() {
		return
	}
	if m.width == 0 {
		m.width, m.height = basicMenuSize()
	}
	m.x = character.x
	m.y = character.y + character.height + basicMenuFollowGap
}

// Draw renders the menu panel, its buttons, and the close glyph. Called
// every frame from the world's UI overlay pass.
func (m *BasicMenu) Draw(screen *render.Frame) {
	if m == nil || screen == nil {
		return
	}
	if hudWebEnabled() {
		return
	}
	if m.width == 0 {
		m.width, m.height = basicMenuSize()
	}
	if m.dismissed || !m.open {
		tx, ty, tw, th := basicMenuEdgeTabRect()
		DrawRoundedSurface(screen, tx, ty, tw, th, basicMenuBackground, basicMenuBorder, characterHUDRadius)
		// Hamburger glyph: three stacked lines.
		for i := 0; i < 3; i++ {
			render.DrawRect(screen, float64(tx+6), float64(ty+14+i*8), 14, 3, basicMenuText)
		}
		return
	}
	DrawRoundedSurface(screen, m.x, m.y, m.width, m.height, basicMenuBackground, basicMenuBorder, characterHUDRadius)
	for i, button := range basicMenuButtons {
		bx, by, bw, bh := m.buttonRect(i)
		DrawRoundedSurface(screen, bx, by, bw, bh, basicMenuButtonBack, basicMenuBorder, 6)
		if labelW := render.MeasureUIText(button.label, characterHUDTextSize); labelW > 0 {
			render.DrawUITextAtSize(screen, button.label, float64(bx+(bw-int(labelW))/2), float64(by+(bh-16)/2), basicMenuText, characterHUDTextSize)
		}
	}
	cx, cy, cw, ch := m.closeRect()
	DrawCloseButton(screen, cx, cy, cw, ch, basicMenuButtonBack, basicMenuText)
}

func (m *BasicMenu) invoke(key string) {
	var callback func()
	switch key {
	case "status":
		callback = m.callbacks.OnStatus
	case "option":
		callback = m.callbacks.OnOption
	case "items":
		callback = m.callbacks.OnItems
	case "equip":
		callback = m.callbacks.OnEquip
	case "skill":
		callback = m.callbacks.OnSkill
	case "map":
		callback = m.callbacks.OnMap
	case "comm":
		callback = m.callbacks.OnComm
	case "friend":
		callback = m.callbacks.OnFriend
	}
	if callback != nil {
		callback()
	}
}
