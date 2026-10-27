package bot

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

// World supplies the game operations without exposing WorldMode or Lua state.
// All callbacks are provided together; a nil World is used during login.
type World struct {
	Walk            func(client.Context, int, int) bool
	Stop            func(client.Context) bool
	Attack          func(client.Context, uint32) bool
	Skill           func(client.Context, uint32, session.Skill, int) bool
	Loot            func(client.Context, uint32) bool
	UseItem         func(client.Context, int) bool
	UsePendingSkill func(client.Context, uint32) bool
	HighlightActor  func(client.Context, uint32) bool
	SkillTargets    func(client.Context, bool, time.Time) []worldstate.Actor
	PendingSkill    func(client.Context) (PendingSkill, bool)
	CanAttack       func(client.Context, worldstate.Actor) bool
	ActorDead       func(uint32) bool
	CompanionLife   func(client.Context, uint32) (int, int, int, int, bool)
	WalkSequence    func() uint64
	UseShortcut     func(client.Context, int) (uint16, bool)
	RotateCamera    func(client.Context, float64, float64)
	ZoomCamera      func(client.Context, float64)
	CameraYaw       func(client.Context) float64
	CancelSkill     func()
}

type PendingSkill struct {
	Skill      session.Skill
	MaxLevel   int
	Target     string
	CasterID   uint32
	CasterKind string
	CasterX    int
	CasterY    int
}
