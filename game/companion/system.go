// Package companion manages homunculus and mercenary state, commands, and Lua AI.
package companion

import (
	"time"

	"github.com/kivutar/goro/client"
	gameui "github.com/kivutar/goro/ui"
)

// Bindings connect companion state to the world's rendering and UI. They are
// called only on the game thread; Lua workers receive copies of the world state.
type Bindings struct {
	Windows      *Windows
	SetLife      func(client.Context, uint32, int, int, int, int, int)
	SetHunger    func(client.Context, uint32, int, int)
	LevelUp      func(client.Context, uint32)
	SnapshotLife func() map[uint32]Vitals
	Error        func(string, ...any)
	Actions      gameui.GameActions
}

type Vitals struct{ HP, MaxHP, SP, MaxSP int }

type Windows struct {
	HomunculusInfo    gameui.HomunculusInfoWindow
	HomunculusSkill   gameui.HomunculusSkillWindow
	HomunculusContext gameui.HomunculusContextMenu
	HomunculusConfirm gameui.ConfirmModal
	MercenaryInfo     gameui.MercenaryInfoWindow
	MercenarySkill    gameui.MercenarySkillWindow
	MercenaryContext  gameui.MercenaryContextMenu
	MercenaryConfirm  gameui.ConfirmModal
}

type System struct {
	HomunculusDeleteID uint32
	MercenaryDeleteID  uint32

	bindings    Bindings
	companionAI *companionAISystem
}

func (s *System) Bind(bindings Bindings) { s.bindings = bindings }

func (s *System) aiSystem() *companionAISystem {
	if s.companionAI == nil {
		s.companionAI = &companionAISystem{started: time.Now()}
	}
	return s.companionAI
}

func (s *System) Close() {
	s.companionAI.close()
	s.companionAI = nil
}

func (s *System) SetMessage(id uint32, message string) {
	s.aiSystem().setCompanionAIMessage(id, message)
}

func (s *System) snapshot(ctx client.Context, deaths map[uint32]time.Time) companionAISnapshot {
	snapshot := newCompanionAISnapshot(ctx, deaths)
	if s.bindings.SnapshotLife != nil {
		snapshot.life = s.bindings.SnapshotLife()
	}
	return snapshot
}
