package game

import (
	"encoding/binary"
	"slices"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/battle"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestMonsterInfoPacketOpensInformationWindow(t *testing.T) {
	now := time.Now()
	uiManager := &worldModeTestUIManager{}
	world := &worldstate.World{Actors: map[uint32]worldstate.Actor{
		300: {ID: 300, Job: 1002, Name: "Poring", HasObjectType: true, ObjectType: scene.ActorObjectTypeMob},
	}}
	ctx := client.Context{
		ScreenW:   800,
		ScreenH:   600,
		World:     world,
		UIManager: uiManager,
	}
	mode := &WorldMode{battleSystem: battle.System{ActorLife: map[uint32]battle.Life{300: {HP: 50, MaxHP: 100}}}}
	mode.battle().RememberSenseTarget(db.SkillWZEstimation, 300, now)
	data := make([]byte, 29)
	binary.LittleEndian.PutUint16(data[0:2], network.PacketZCMonsterInfo)
	binary.LittleEndian.PutUint16(data[2:4], 1002)
	binary.LittleEndian.PutUint16(data[4:6], 4)
	binary.LittleEndian.PutUint32(data[8:12], 45)
	for i := 20; i < len(data); i++ {
		data[i] = 100
	}

	if next, stop := mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCMonsterInfo, Data: data}, now); next != nil || stop {
		t.Fatalf("monster info changed mode: next=%T stop=%t", next, stop)
	}
	if !mode.ui.battle.MonsterInfo.IsOpen() {
		t.Fatal("monster information window did not open")
	}
	if mode.ui.interactionModalOpen() {
		t.Fatal("monster information window should not block world interaction")
	}
	if life := mode.battleSystem.ActorLife[300]; life.HP != 45 || life.MaxHP != 100 {
		t.Fatalf("cached monster life = %+v, want 45/100", life)
	}
	if len(uiManager.overlays) != 1 {
		t.Fatalf("monster info overlays = %d, want 1", len(uiManager.overlays))
	}
}

func TestStatusIconsRemainRemovableAfterMapTransitions(t *testing.T) {
	for _, tt := range []struct {
		name             string
		updateBeforeStop bool
	}{
		{"status_ends_after_first_update", true},
		{"status_ends_before_first_update", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			manager := &worldModeTestUIManager{}
			ctx := client.Context{
				Session: &session.Session{AccountID: 2000000, CharID: 150000},
				World:   worldstate.New(), UIManager: manager, ScreenW: 800, ScreenH: 600,
			}
			now := time.Now()
			mode := NewWorldMode()
			mode.handleNetworkPacket(ctx, testStatusEffectChangePacket(db.StatusIncAgi, ctx.Session.AccountID, true), now)
			mode.ui.battle.StatusIcons.Update(ctx, now)
			if len(manager.overlays) != 1 {
				t.Fatalf("status overlays = %d, want 1", len(manager.overlays))
			}
			original := manager.overlays[0]
			for range 3 {
				mode = mode.nextWorldMode()
				if tt.updateBeforeStop {
					mode.ui.battle.StatusIcons.Update(ctx, now)
					if len(manager.overlays) != 1 || manager.overlays[0] != original {
						t.Errorf("map transition left %d overlays, want the original status overlay only", len(manager.overlays))
					}
				}
			}

			mode.handleNetworkPacket(ctx, testStatusEffectChangePacket(db.StatusIncAgi, ctx.Session.AccountID, false), now)
			if _, active := ctx.Session.Statuses.Active[db.StatusIncAgi]; active {
				t.Fatal("server stop packet did not remove Increase Agility")
			}
			mode.ui.battle.StatusIcons.Update(ctx, now)
			if len(manager.overlays) != 0 {
				t.Fatalf("status ended but %d overlays remain", len(manager.overlays))
			}
		})
	}
}

func TestStatusIconsRefreshWhileWindowConsumesInput(t *testing.T) {
	for _, modal := range []bool{false, true} {
		name := "hovered_window"
		if modal {
			name = "escape_menu"
		}
		t.Run(name, func(t *testing.T) {
			manager := &worldModeTestUIManager{}
			netClient := network.NewClient(20080910, false)
			defer netClient.Close()
			ctx := client.Context{
				Session: &session.Session{AccountID: 2000000, CharID: 150000},
				World:   worldstate.New(), Input: input.NewState(), Network: netClient,
				UIManager: manager, ScreenW: 800, ScreenH: 600,
			}
			mode := NewWorldMode()
			mode.handleNetworkPacket(ctx, testStatusEffectChangePacket(db.StatusIncAgi, ctx.Session.AccountID, true), time.Now())
			mode.ui.battle.StatusIcons.Update(ctx, time.Now())
			if len(manager.overlays) != 1 {
				t.Fatalf("status overlays = %d, want 1", len(manager.overlays))
			}
			iconOverlay := manager.overlays[0]
			if modal {
				mode.ui.escapeMenu.Toggle(ctx)
			} else {
				mode.ui.statsWindow.OpenWindow(ctx)
				mode.ui.statsWindow.SetAutoPosition(100, 200)
				ctx.Input.SetMousePosition(110, 210)
			}

			mode.handleNetworkPacket(ctx, testStatusEffectChangePacket(db.StatusIncAgi, ctx.Session.AccountID, false), time.Now())
			if _, err := mode.Update(ctx); err != nil {
				t.Fatal(err)
			}
			if slices.Contains(manager.overlays, iconOverlay) {
				t.Fatal("status overlay remained visible while the window consumed input")
			}
		})
	}
}
