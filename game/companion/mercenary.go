package companion

import (
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func (m *System) HandleMercenaryContextAction(ctx client.Context, action gameui.MercenaryContextAction) {
	switch action.Kind {
	case gameui.MercenaryContextActionInfo:
		m.OpenMercenaryInfo(ctx)
	case gameui.MercenaryContextActionToggleAssist:
		m.toggleMercenaryAssist(ctx)
	}
}

func (m *System) HandleMercenaryInfoAction(ctx client.Context, action gameui.MercenaryInfoAction) {
	switch action.Kind {
	case gameui.MercenaryInfoActionSkill:
		m.bindings.Windows.MercenarySkill.Toggle(ctx, m.bindings.Actions)
	case gameui.MercenaryInfoActionDelete:
		m.openMercenaryDeleteConfirm(ctx)
	}
}

func (m *System) OpenMercenaryInfo(ctx client.Context) {
	if ctx.Session == nil || (!ctx.Session.Mercenary.Active && ctx.Session.Mercenary.ID == 0) {
		return
	}
	m.bindings.Windows.MercenaryInfo.OpenInfo(ctx, ctx.Session.Mercenary)
}

func (m *System) openMercenaryDeleteConfirm(ctx client.Context) {
	m.bindings.Windows.MercenaryConfirm.Open(ctx, "Dismiss Mercenary", "Are you sure that you want to delete?", func() {
		m.sendMercenaryDelete(ctx)
	}, nil)
}

func (m *System) sendMercenaryDelete(ctx client.Context) {
	if ctx.Network == nil {
		m.bindings.Error("Mercenary dismiss failed: not connected.")
		return
	}
	if err := ctx.Network.SendMercenaryCommand(network.MercenaryCommandDelete); err != nil {
		m.bindings.Error("Mercenary dismiss failed.")
		glog.Warnf("mercenary dismiss failed: %v", err)
		return
	}
	m.MercenaryDeleteID = currentMercenaryID(ctx)
}

func (m *System) toggleMercenaryAssist(ctx client.Context) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.MercenaryAggressive = !ctx.Session.MercenaryAggressive
	glog.Debugf("mercenary assist aggressive=%t", ctx.Session.MercenaryAggressive)
}

func mercenaryAggressive(ctx client.Context) bool {
	return ctx.Session != nil && ctx.Session.MercenaryAggressive
}

func currentMercenaryID(ctx client.Context) uint32 {
	if ctx.Session != nil && ctx.Session.Mercenary.ID != 0 {
		return ctx.Session.Mercenary.ID
	}
	return FindActorID(ctx, scene.ActorObjectTypeMercenary)
}

func IsMercenary(actor worldstate.Actor) bool {
	return actor.HasObjectType && actor.ObjectType == scene.ActorObjectTypeMercenary
}

func MercenaryWeaponForAppearance(job int, weapon int) int {
	if weapon > 0 {
		return weapon
	}
	return mercenaryDefaultWeapon(job)
}

func MercenaryWeaponBaseJob(job int) int {
	switch {
	case job >= 6017 && job <= 6026:
		return db.JobArcher
	case job >= 6027 && job <= 6046:
		return db.JobSwordman
	default:
		return db.JobNovice
	}
}

func mercenaryDefaultWeapon(job int) int {
	switch {
	case job >= 6017 && job <= 6026:
		return db.WeaponBow
	case job >= 6027 && job <= 6036:
		return db.WeaponSpear
	case job >= 6037 && job <= 6046:
		return db.WeaponSword
	default:
		return 0
	}
}

func MercenarySexForJob(job int, fallback byte) byte {
	if resource, ok := db.MonsterResourceName[job]; ok {
		normalized := strings.ReplaceAll(resource, "/", "\\")
		switch {
		case strings.HasPrefix(normalized, "남\\"):
			return 1
		case strings.HasPrefix(normalized, "여\\"):
			return 0
		}
	}
	switch {
	case job >= 6017 && job <= 6026:
		return 0
	case job >= 6027 && job <= 6046:
		return 1
	default:
		return fallback
	}
}

func MercenaryUsesArcherActions(job int16) bool {
	return job >= 6017 && job <= 6026
}

func (m *System) ClearDeletedMercenary(ctx client.Context) {
	id := m.MercenaryDeleteID
	if id == 0 && ctx.Session != nil {
		id = ctx.Session.Mercenary.ID
	}
	m.MercenaryDeleteID = 0
	if ctx.Session != nil {
		ctx.Session.Mercenary = session.Companion{}
	}
	if id != 0 {
		m.aiSystem().clearMessages(id)
	}
	m.bindings.Windows.MercenaryInfo.Close()
	m.bindings.Windows.MercenarySkill.Close()
	m.bindings.Windows.MercenaryContext.Close()
}

func (m *System) OpenMercenaryContext(ctx client.Context, actor worldstate.Actor) {
	if ctx.Session != nil {
		ctx.Session.Mercenary.ID = actor.ID
		ctx.Session.Mercenary.Active = true
		ctx.Session.Mercenary.Job = actor.Job
		if ctx.Session.Mercenary.Name == "" {
			ctx.Session.Mercenary.Name = actor.Name
		}
	}
	glog.Debugf("mercenary context target id=%d name=%q job=%d", actor.ID, actor.Name, actor.Job)
	m.bindings.Windows.MercenaryContext.Open(ctx, ctx.Input.MouseX, ctx.Input.MouseY, mercenaryAggressive(ctx))
}
