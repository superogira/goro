package game

import (
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/input"
	gameui "github.com/kivutar/goro/ui"
)

func TestUISettingsRestoreAndRelog(t *testing.T) {
	t.Skip("fork: the character HUD and basic menu are custom layouts without upstream compact/collapse state")
}
func TestUIPositionsSaveAfterDrag(t *testing.T) {
	for _, target := range []string{"console", "shortcuts", "stats", "inventory"} {
		t.Run(target, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			ctx.UIManager = gameui.NewManager()
			initial := config.UISettings{ShortcutRows: 1,
				CameraZoom:        scene.CameraZoom(),
				ConsolePosition:   config.WindowPosition{X: 40, Y: 320, Set: true},
				ShortcutPosition:  config.WindowPosition{X: 200, Y: 160, Set: true},
				StatsPosition:     config.WindowPosition{X: 300, Y: 80, Set: true},
				StatsOpen:         target == "stats",
				InventoryPosition: config.WindowPosition{X: 300, Y: 80, Set: true},
				InventoryOpen:     target == "inventory",
			}
			if _, err := ctx.Config.SaveUISettings(initial); err != nil {
				t.Fatal(err)
			}
			m := NewWorldMode()
			m.restoreUISettings(ctx)
			x, y := 140, 380 // console message area
			update := func() { m.ui.console.Update(ctx) }
			if target == "shortcuts" {
				x, y = 205, 180 // left grip
				update = func() { m.ui.shortcutBar.Update(ctx, nil) }
			} else if target == "stats" {
				x, y = 320, 85 // title bar
				update = func() { m.ui.statsWindow.Update(ctx) }
			} else if target == "inventory" {
				x, y = 320, 85
				update = func() { m.ui.inventoryBag.Update(ctx, nil, nil, nil, nil, nil, nil) }
			}
			ctx.Input.SetMousePosition(x, y)
			ctx.Input.SetMouseButton(input.MouseButtonLeft, true)
			update()
			ctx.Input.EndFrame()
			ctx.Input.SetMousePosition(x+30, y-40)
			update()
			m.SaveUISettings(ctx)
			if got, err := ctx.Config.LoadUISettings(); err != nil || got != initial {
				t.Fatalf("wrote settings during drag: %+v, %v", got, err)
			}
			ctx.Input.EndFrame()
			ctx.Input.SetMouseButton(input.MouseButtonLeft, false)
			update()
			m.SaveUISettings(ctx)
			want := initial
			position := &want.ConsolePosition
			if target == "shortcuts" {
				position = &want.ShortcutPosition
			} else if target == "stats" {
				position = &want.StatsPosition
			} else if target == "inventory" {
				position = &want.InventoryPosition
			}
			position.X += 30
			position.Y -= 40
			if got, err := ctx.Config.LoadUISettings(); err != nil || got != want {
				t.Fatalf("released position = %+v, %v; want %+v", got, err, want)
			}
		})
	}
}

func TestUISettingsWaitForCompanionsAndCart(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Config.UI = config.UISettings{PetOpen: true, HomunculusOpen: true, HomunculusSkillsOpen: true,
		MercenaryOpen: true, MercenarySkillsOpen: true, CartOpen: true, ShortcutRows: 1,
		PetPosition:              config.WindowPosition{X: 40, Y: 60, Set: true},
		HomunculusPosition:       config.WindowPosition{X: 50, Y: 60, Set: true},
		HomunculusSkillsPosition: config.WindowPosition{X: 60, Y: 60, Set: true},
		MercenaryPosition:        config.WindowPosition{X: 70, Y: 60, Set: true},
		MercenarySkillsPosition:  config.WindowPosition{X: 80, Y: 60, Set: true},
		CartPosition:             config.WindowPosition{X: 90, Y: 60, Set: true},
	}
	m := NewWorldMode()
	m.restoreUISettings(ctx)
	if m.ui.pet.Info.IsOpen() || m.ui.companions.HomunculusInfo.IsOpen() || m.ui.cartWindow.IsOpen() {
		t.Fatal("opened windows before their data was available")
	}
	m.ui.inventoryBag.Toggle(ctx)
	m.SaveUISettings(ctx)
	saved, err := ctx.Config.LoadUISettings()
	if err != nil || !saved.PetOpen || !saved.HomunculusOpen || !saved.CartOpen {
		t.Fatalf("late data erased saved choices: %+v, %v", saved, err)
	}
	m.petSystem.HasProperty = true
	ctx.Session.Homunculus.Active = true
	ctx.Session.Mercenary.Active = true
	ctx.Session.Cart.MaxAmount = 100
	m.restoreUISettings(ctx)
	if !m.ui.pet.Info.IsOpen() || !m.ui.companions.HomunculusInfo.IsOpen() ||
		!m.ui.companions.HomunculusSkill.IsOpen() || !m.ui.companions.MercenaryInfo.IsOpen() ||
		!m.ui.companions.MercenarySkill.IsOpen() || !m.ui.cartWindow.IsOpen() {
		t.Fatal("did not restore windows after their data arrived")
	}
	if got := m.uiSettings(); got.PetPosition != ctx.Config.UI.PetPosition ||
		got.HomunculusPosition != ctx.Config.UI.HomunculusPosition || got.HomunculusSkillsPosition != ctx.Config.UI.HomunculusSkillsPosition ||
		got.MercenaryPosition != ctx.Config.UI.MercenaryPosition || got.MercenarySkillsPosition != ctx.Config.UI.MercenarySkillsPosition || got.CartPosition != ctx.Config.UI.CartPosition {
		t.Fatal("late data lost restored positions")
	}
	m.ui.companions.HomunculusInfo.Close()
	m.ui.cartWindow.Close()
	m.restoreUISettings(ctx)
	m.SaveUISettings(ctx)
	saved, err = ctx.Config.LoadUISettings()
	if err != nil || saved.HomunculusOpen || saved.CartOpen {
		t.Fatalf("reopened manually closed windows: %+v, %v", saved, err)
	}
}

func TestUISettingsRestoreBasicInfoAndMinimapPositions(t *testing.T) {
	t.Skip("fork: HUD/minimap use fixed placement; position persistence is upstream widget-window behavior")
}
func TestUISettingsDoNotSaveBeforeRestoreOrWhenHeadless(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	m := NewWorldMode()
	m.ui.inventoryBag.Toggle(ctx)
	m.SaveUISettings(ctx)
	ctx.Config.Headless = true
	m.ui.preferences.loaded = true
	m.SaveUISettings(ctx)
	if _, err := os.Stat(ctx.Config.ConfigPath); !os.IsNotExist(err) {
		t.Fatalf("saved UI settings without a graphical session: %v", err)
	}
}

func TestUISettingsRememberZoom(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := chatShortcutTestContext(t)
		m := NewWorldMode()
		m.restoreUISettings(ctx)
		m.camera.zoom = m.camera.targetZoom()
		for range 3 {
			m.camera.ZoomByDelta(2.5)
			m.SaveUISettings(ctx)
			time.Sleep(100 * time.Millisecond)
		}
		if _, err := os.Stat(ctx.Config.ConfigPath); !os.IsNotExist(err) {
			t.Fatalf("wrote preferences during continuous zoom input: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
		m.SaveUISettings(ctx)
		want := m.camera.targetZoom()
		if saved, err := ctx.Config.LoadUISettings(); err != nil || saved.CameraZoom != want {
			t.Fatalf("saved zoom = %v, %v; want target %v", saved.CameraZoom, err, want)
		}
		// Animation changes the displayed zoom, not the user's selected value.
		old := time.Unix(1000, 0)
		if err := os.Chtimes(ctx.Config.ConfigPath, old, old); err != nil {
			t.Fatal(err)
		}
		m.camera.zoom += 1
		m.SaveUISettings(ctx)
		if info, err := os.Stat(ctx.Config.ConfigPath); err != nil || !info.ModTime().Equal(old) {
			t.Fatalf("zoom animation rewrote preferences: %v, %v", info, err)
		}
		next := NewWorldMode()
		next.restoreUISettings(ctx)
		if next.camera.currentZoom() != want || next.camera.targetZoom() != want {
			t.Fatal("relogin did not immediately restore the chosen zoom")
		}
		next.camera.ZoomBy(1.05)
		want = next.camera.targetZoom()
		next.FlushUISettings(ctx)
		if saved, err := ctx.Config.LoadUISettings(); err != nil || saved.CameraZoom != want {
			t.Fatalf("exit lost the final zoom change: %v, %v; want %v", saved.CameraZoom, err, want)
		}
	})
}

func TestUISettingsRecoverFromFailedWrite(t *testing.T) {
	for _, recovery := range []string{"shutdown", "leave world", "new choice"} {
		t.Run(recovery, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			parent := filepath.Join(t.TempDir(), "settings")
			ctx.Config.ConfigPath = filepath.Join(parent, "goro.ini")
			m := NewWorldMode()
			m.restoreUISettings(ctx)
			// A file in place of the parent directory temporarily blocks writes.
			if err := os.WriteFile(parent, []byte("blocked"), 0o600); err != nil {
				t.Fatal(err)
			}
			m.ui.inventoryBag.Toggle(ctx)
			m.SaveUISettings(ctx)
			if err := os.Remove(parent); err != nil {
				t.Fatal(err)
			}
			for range 3 {
				m.SaveUISettings(ctx)
			}
			if _, err := os.Stat(ctx.Config.ConfigPath); !os.IsNotExist(err) {
				t.Fatalf("unchanged frames retried the failed write: %v", err)
			}
			manager := &Manager{ctx: ctx, mode: m}
			switch recovery {
			case "shutdown":
				manager.Close()
			case "leave world":
				manager.enter(&mapTextureExitMode{})
			case "new choice":
				m.ui.statsWindow.OpenWindow(ctx)
				m.SaveUISettings(ctx)
			}
			if _, err := os.Stat(ctx.Config.ConfigPath); err != nil {
				t.Fatalf("did not retry the unsaved layout: %v", err)
			}
			saved, err := ctx.Config.LoadUISettings()
			if err != nil || !saved.InventoryOpen || saved.StatsOpen != (recovery == "new choice") {
				t.Fatalf("recovered settings = %+v, %v", saved, err)
			}
			// A successful retry must clear dirty state, including at shutdown.
			old := time.Unix(1000, 0)
			if err := os.Chtimes(ctx.Config.ConfigPath, old, old); err != nil {
				t.Fatal(err)
			}
			manager.Close()
			info, err := os.Stat(ctx.Config.ConfigPath)
			if err != nil || !info.ModTime().Equal(old) {
				t.Fatalf("clean shutdown rewrote settings: %v, %v", info, err)
			}
		})
	}
}
