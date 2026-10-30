package game

import (
	"time"

	"github.com/kivutar/goro/client"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/pet"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	worldstate "github.com/kivutar/goro/world"
)

func (m *WorldMode) handlePetCaptureClick(ctx client.Context, projection scene.Projection, now time.Time) bool {
	if !m.petSystem.PendingCapture.Active || ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonLeft) {
		return false
	}
	skill := pet.CaptureTargetSkill()
	actor, ok := clickedSkillTarget(ctx, projection, skill, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorSystem.ActorDeaths)
	if !ok {
		if x, y, groundOK := maps.ClickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY); groundOK {
			glog.Debugf("pet capture canceled by ground click mouse=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, x, y)
			m.pets().CancelCapture("ground-click")
			return true
		}
		glog.Debugf("pet capture target miss mouse=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY)
		return true
	}
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Capture failed: not connected.")
		return true
	}
	m.petSystem.PendingCapture = pet.CaptureState{}
	m.pets().OpenSlotMachine(ctx, actor.ID)
	glog.Debugf("pet capture target selected target=%d name=%q job=%d object_type=%d", actor.ID, actor.Name, actor.Job, actor.ObjectType)
	return true
}

func (m *WorldMode) openPetContextFromInput(ctx client.Context, now time.Time) bool {
	if ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonRight) || m.mapPointerBlocked(ctx) {
		return false
	}
	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	actor, ok := clickedPetTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorSystem.ActorDeaths, m.petSystem.ID)
	if !ok {
		return false
	}
	glog.Debugf("pet context target id=%d name=%q job=%d", actor.ID, actor.Name, actor.Job)
	m.ui.pet.Context.Open(ctx, ctx.Input.MouseX, ctx.Input.MouseY)
	return true
}

func clickedPetTarget(ctx client.Context, projection scene.Projection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time, petID uint32) (worldstate.Actor, bool) {
	if petID == 0 {
		return worldstate.Actor{}, false
	}
	actor, ok := hoveredCursorActor(ctx, projection, mouseX, mouseY, now, deadActors)
	if !ok || actor.ID != petID {
		return worldstate.Actor{}, false
	}
	return actor, true
}

func (m *WorldMode) pets() *pet.System {
	m.petSystem.Bind(pet.Bindings{
		Windows:        &m.ui.pet,
		Console:        &m.ui.console,
		Actors:         m.actors(),
		Skills:         &m.skillSystem,
		StartAnimation: m.startActorAnimation,
		Emotion:        m.applyEmotionNotify,
	})
	return &m.petSystem
}
