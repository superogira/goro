package game

import (
	"testing"

	"github.com/gogpu/gpucontext"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
)

func openCompanionInputTestWindow(ctx client.Context, m *WorldMode, name string) *gameui.Window {
	switch name {
	case "pet":
		m.ui.petInfoWindow.OpenInfo(ctx, network.PetProperty{Name: "Poring"})
		return &m.ui.petInfoWindow.Window
	case "homunculus":
		m.ui.homunculusInfo.OpenInfo(ctx, ctx.Session.Homunculus)
		return &m.ui.homunculusInfo.Window
	case "homunculus skills":
		m.ui.homunculusSkill.Open(ctx, m)
		return &m.ui.homunculusSkill.Window
	case "mercenary":
		m.ui.mercenaryInfo.OpenInfo(ctx, ctx.Session.Mercenary)
		return &m.ui.mercenaryInfo.Window
	default:
		m.ui.mercenarySkill.Open(ctx, m)
		return &m.ui.mercenarySkill.Window
	}
}

func TestCompanionWindowsAllowGameplayInput(t *testing.T) {
	for _, name := range []string{"pet", "homunculus", "homunculus skills", "mercenary", "mercenary skills"} {
		for _, control := range []string{"camera", "skill", "wasd"} {
			t.Run(name+"/"+control, func(t *testing.T) {
				ctx := wasdControlsTestContext(t)
				conn, server := newBotTestConnection(t, 20080910)
				ctx.Network = conn
				ctx.UIManager = gameui.NewManager()
				ctx.Session.Homunculus.Active = true
				ctx.Session.Mercenary.Active = true
				mode := NewWorldMode()
				w := openCompanionInputTestWindow(ctx, mode, name)
				w.SetAutoPosition(100, 100)
				w.Publish(ctx)
				ctx.Input.SetMousePosition(700, 400)
				ctx.Input.EndFrame()
				switch control {
				case "camera":
					ctx.Input.SetMouseButton(input.MouseButtonRight, true)
					ctx.Input.SetMousePosition(720, 400)
				case "skill":
					ctx.Input.SetKey(input.KeyF1, true)
				case "wasd":
					loadKeyboardTestBot(t, ctx, mode)
					if !prepareKeyboardTestText(mode, ctx, gpucontext.KeyW, "w") {
						t.Fatal("WASD did not claim the movement key")
					}
				}
				if _, err := mode.Update(ctx); err != nil {
					t.Fatal(err)
				}
				if !w.IsOpen() {
					t.Fatal("gameplay input closed the companion window")
				}
				switch control {
				case "camera":
					if mode.camera.yawOffset == 0 {
						t.Fatal("open companion window blocked camera rotation outside the UI")
					}
					yaw := mode.camera.yawOffset
					ctx.Input.EndFrame()
					ctx.Input.SetMousePosition(120, 150)
					if !uiPointerBlocked(ctx) {
						t.Fatal("test pointer is not over the companion window")
					}
					if _, err := mode.Update(ctx); err != nil {
						t.Fatal(err)
					}
					if mode.camera.yawOffset != yaw {
						t.Fatal("camera rotated through the companion window")
					}
				case "skill":
					if mode.pendingSkill.skill.ID != ctx.Session.Skills.List[0].ID {
						t.Fatal("open companion window blocked the F1 skill shortcut")
					}
				case "wasd":
					walk, _ := network.BuildWalkToXYPacketForClientDate(10, 28, 20080910)
					readBotTestPackets(t, server, walk)
				}
			})
		}
	}
}

func TestCompanionRenameKeepsKeyboardInput(t *testing.T) {
	for _, name := range []string{"pet", "homunculus"} {
		t.Run(name, func(t *testing.T) {
			ctx := wasdControlsTestContext(t)
			ctx.Network = network.NewClient(20080910, false)
			t.Cleanup(ctx.Network.Close)
			ctx.Session.Homunculus.Active = true
			mode := NewWorldMode()
			loadKeyboardTestBot(t, ctx, mode)
			w := openCompanionInputTestWindow(ctx, mode, name)
			var field *textfield.Widget
			var find func(widget.Widget)
			find = func(node widget.Widget) {
				if f, ok := node.(*textfield.Widget); ok {
					field = f
				}
				for _, child := range node.Children() {
					find(child)
				}
			}
			find(w.Widget())
			if field == nil {
				t.Fatal("companion has no rename field")
			}
			field.SetFocused(true)
			if name == "pet" {
				field.SetText("Fluffy")
				mode.applyPetProperty(ctx, network.PetProperty{Name: "Poring", Fullness: 80})
				mode.applyPetStateChange(ctx, network.PetStateChange{Type: petStateHunger, Data: 79})
			}
			if prepareKeyboardTestText(mode, ctx, gpucontext.KeyW, "w") || mode.ui.console.Active() {
				t.Fatal("gameplay or chat stole text intended for the rename field")
			}
			ctx.Input.SetKey(input.KeyF1, true)
			if _, err := mode.Update(ctx); err != nil {
				t.Fatal(err)
			}
			if mode.pendingSkill.skill.ID != 0 || mode.bot.keyboardAvailable {
				t.Fatal("gameplay keyboard input remained active during rename")
			}
			if mode.ui.nonConsoleKeyboardInputBlockedExcept(ctx, w) {
				t.Fatal("rename blocked the window's own toggle shortcut")
			}
			field.SetFocused(false)
			if mode.ui.KeyboardShortcutsBlocked(ctx) {
				t.Fatal("leaving the rename field did not restore shortcuts")
			}
			field.SetFocused(true)
			w.Close()
			if mode.ui.KeyboardShortcutsBlocked(ctx) {
				t.Fatal("closing the window did not restore shortcuts")
			}
		})
	}
}
