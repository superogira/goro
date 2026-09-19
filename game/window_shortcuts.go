package game

import (
	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	gameui "github.com/kivutar/goro/ui"
)

// Like the other shortcuts, these use physical keys, not typed characters.
var windowShortcutKeys = [...]input.KeyCode{
	gpucontext.KeyE, gpucontext.KeyQ, gpucontext.KeyS, gpucontext.KeyA,
	gpucontext.KeyZ, gpucontext.KeyH, gpucontext.KeyW, gpucontext.KeyC,
	gpucontext.KeyV, gpucontext.KeyJ, gpucontext.KeyR, gpucontext.KeyO,
}

// Shared with text filtering so handled shortcuts cannot also type into chat.
func (m *WorldMode) windowShortcutAllowed(ctx client.Context, code input.KeyCode) bool {
	var target *gameui.Window
	if code == gpucontext.KeyR && plainCtrlDown(ctx.Input) {
		target = &m.ui.mercenaryInfo.Window
	} else {
		if !plainAltDown(ctx.Input) {
			return false
		}
		switch code {
		case gpucontext.KeyE, gpucontext.KeyQ, gpucontext.KeyS, gpucontext.KeyA,
			gpucontext.KeyZ, gpucontext.KeyH, gpucontext.KeyW, gpucontext.KeyV:
		case gpucontext.KeyC:
			target = &m.ui.chatRoomCreate.Window
		case gpucontext.KeyJ:
			target = &m.ui.petInfoWindow.Window
		case gpucontext.KeyR:
			target = &m.ui.homunculusInfo.Window
		case gpucontext.KeyO:
			target = &m.ui.settingsWindow.Window
		default:
			return false
		}
	}
	return !m.ui.nonConsoleKeyboardInputBlockedExcept(ctx, target)
}

func (m *WorldMode) toggleWindowFromInput(ctx client.Context) bool {
	if !plainAltDown(ctx.Input) && !plainCtrlDown(ctx.Input) {
		return false
	}
	for _, code := range windowShortcutKeys {
		if !ctx.Input.KeyCodeJustPressed(code) || !m.windowShortcutAllowed(ctx, code) {
			continue
		}
		ctx.Input.ConsumeKeyCodePress(code)
		switch code {
		case gpucontext.KeyE:
			m.ui.inventoryBag.Toggle(ctx)
		case gpucontext.KeyQ:
			m.ui.equipmentWindow.Toggle(ctx)
		case gpucontext.KeyS:
			m.ui.skillWindow.Toggle(ctx)
		case gpucontext.KeyA:
			m.ui.statsWindow.Toggle(ctx)
		case gpucontext.KeyZ:
			m.ui.friendsWindow.ToggleParty(ctx)
		case gpucontext.KeyH:
			m.ui.friendsWindow.ToggleFriends(ctx)
		case gpucontext.KeyW:
			m.ui.cartWindow.Toggle(ctx)
		case gpucontext.KeyV:
			m.ui.characterWindow.ToggleCompact()
			m.ui.basicMenu.FollowCharacterWindow(ctx, &m.ui.characterWindow)
		case gpucontext.KeyC:
			if m.ui.chatRoomCreate.IsOpen() {
				m.ui.chatRoomCreate.Close()
			} else {
				m.ui.chatRoomCreate.Open(ctx)
			}
		case gpucontext.KeyJ:
			if m.ui.petInfoWindow.IsOpen() {
				m.ui.petInfoWindow.Close()
			} else if m.petID != 0 {
				m.openPetInfo(ctx)
			}
		case gpucontext.KeyR:
			if plainCtrlDown(ctx.Input) {
				if m.ui.mercenaryInfo.IsOpen() {
					m.ui.mercenaryInfo.Close()
				} else {
					m.openMercenaryInfo(ctx)
				}
			} else if m.ui.homunculusInfo.IsOpen() {
				m.ui.homunculusInfo.Close()
			} else {
				m.openHomunculusInfo(ctx)
			}
		case gpucontext.KeyO:
			if m.ui.settingsWindow.IsOpen() {
				m.ui.settingsWindow.Close()
			} else {
				m.ui.settingsWindow.OpenWindow(ctx)
			}
		}
		return true
	}
	return false
}
