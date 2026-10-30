package battle

import (
	"fmt"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

const senseResponseTimeout = 5 * time.Second

type senseRequest struct {
	targetID    uint32
	requestedAt time.Time
}

func isSenseSkill(skillID uint16) bool {
	return skillID == db.SkillWZEstimation || skillID == db.SkillMerEstimation
}

func (m *System) RememberSenseTarget(skillID uint16, targetID uint32, now time.Time) {
	if m == nil || !isSenseSkill(skillID) || targetID == 0 {
		return
	}
	m.senseRequest = senseRequest{targetID: targetID, requestedAt: now}
}

func (m *System) ClearSenseRequest(skillID uint16) {
	if m != nil && isSenseSkill(skillID) {
		m.senseRequest = senseRequest{}
	}
}

func (m *System) ApplyMonsterInfo(ctx client.Context, info network.MonsterInfo, now time.Time) {
	actor, matched := m.senseActor(ctx, info.Class, now)
	maximumHP := uint32(0)
	if matched {
		if life, ok := m.monsterLifeForSense(actor.ID); ok && life.MaxHP >= int(info.HP) {
			maximumHP = uint32(life.MaxHP)
			life.HP = int(info.HP)
			m.ActorLife[actor.ID] = life
		}
	}
	view := gameui.MonsterInfoView{
		Info:    info,
		Name:    monsterInfoDisplayName(ctx, actor, matched, info.Class),
		MaxHP:   maximumHP,
		Preview: m.bindings.MonsterPreview(ctx, info.Class, monsterInfoPreviewWidth, monsterInfoPreviewHeight),
	}
	m.bindings.Windows.MonsterInfo.OpenInfo(ctx, view)
	glog.Debugf("monster info class=%d level=%d hp=%d max_hp=%d actor=%d", info.Class, info.Level, info.HP, maximumHP, actor.ID)
}

// senseActor associates the response with the request that caused it. The
// original packet has no actor ID, so party-shared Sense results must not be
// guessed from nearby same-class monsters.
func (m *System) senseActor(ctx client.Context, class uint16, now time.Time) (worldstate.Actor, bool) {
	if m == nil || ctx.World == nil {
		return worldstate.Actor{}, false
	}
	request := m.senseRequest
	if request.targetID != 0 && now.Sub(request.requestedAt) <= senseResponseTimeout {
		if actor, ok := ctx.World.Actors[request.targetID]; ok && uint16(actor.Job) == class && scene.IsMonsterLikeHoverActor(actor) {
			m.senseRequest = senseRequest{}
			if actor.ID == 0 {
				actor.ID = request.targetID
			}
			return actor, true
		}
	} else if request.targetID != 0 {
		m.senseRequest = senseRequest{}
	}
	return worldstate.Actor{}, false
}

func monsterInfoDisplayName(ctx client.Context, actor worldstate.Actor, matched bool, class uint16) string {
	if matched {
		if name := scene.SanitizeActorName(actor.Name); name != "" {
			return name
		}
	}
	if name := strings.TrimSpace(db.MonsterDisplayName[int(class)]); name != "" {
		return name
	}
	if ctx.Resources != nil {
		if resourceName, ok := ctx.Resources.NonPCResourceName(int(class)); ok {
			if name := actors.DisplayNameFromResource(resourceName); name != "" {
				return name
			}
		}
	}
	return fmt.Sprintf("Monster %d", class)
}

const (
	monsterInfoPreviewWidth  = 110
	monsterInfoPreviewHeight = 160
)
