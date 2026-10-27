package companion

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func (m *System) HandleHomunculusContextAction(ctx client.Context, action gameui.HomunculusContextAction) {
	switch action.Kind {
	case gameui.HomunculusContextActionInfo:
		m.OpenHomunculusInfo(ctx)
	case gameui.HomunculusContextActionFeed:
		m.openHomunculusFeedConfirm(ctx)
	case gameui.HomunculusContextActionToggleAssist:
		m.toggleHomunculusAssist(ctx)
	}
}

func (m *System) HandleHomunculusInfoAction(ctx client.Context, action gameui.HomunculusInfoAction) {
	switch action.Kind {
	case gameui.HomunculusInfoActionSkill:
		m.bindings.Windows.HomunculusSkill.Toggle(ctx, m.bindings.Actions)
	case gameui.HomunculusInfoActionFeed:
		m.openHomunculusFeedConfirm(ctx)
	case gameui.HomunculusInfoActionDelete:
		m.openHomunculusDeleteConfirm(ctx)
	case gameui.HomunculusInfoActionRename:
		m.sendHomunculusRename(ctx, action.Name)
	}
}

func (m *System) OpenHomunculusInfo(ctx client.Context) {
	if ctx.Session == nil || (!ctx.Session.Homunculus.Active && ctx.Session.Homunculus.ID == 0) {
		return
	}
	if ctx.Network != nil {
		if err := ctx.Network.SendHomunculusCommand(network.HomunculusCommandInfo); err != nil {
			m.bindings.Error("Homunculus info request failed.")
			glog.Warnf("homunculus info request failed: %v", err)
		}
	}
	m.bindings.Windows.HomunculusInfo.OpenInfo(ctx, ctx.Session.Homunculus)
}

func (m *System) openHomunculusFeedConfirm(ctx client.Context) {
	m.bindings.Windows.HomunculusConfirm.Open(ctx, "Feed Homunculus", "Are you sure you want to feed your homunculus?", func() {
		m.sendHomunculusFeed(ctx)
	}, nil)
}

func (m *System) openHomunculusDeleteConfirm(ctx client.Context) {
	m.bindings.Windows.HomunculusConfirm.Open(ctx, "Delete Homunculus", "Are you sure that you want to delete?", func() {
		m.sendHomunculusDelete(ctx)
	}, nil)
}

func (m *System) sendHomunculusFeed(ctx client.Context) {
	if ctx.Network == nil {
		m.bindings.Error("Homunculus feed failed: not connected.")
		return
	}
	if err := ctx.Network.SendHomunculusFeed(); err != nil {
		m.bindings.Error("Homunculus feed failed.")
		glog.Warnf("homunculus feed failed: %v", err)
	}
}

func (m *System) sendHomunculusDelete(ctx client.Context) {
	if ctx.Network == nil {
		m.bindings.Error("Homunculus delete failed: not connected.")
		return
	}
	if err := ctx.Network.SendHomunculusDelete(); err != nil {
		m.bindings.Error("Homunculus delete failed.")
		glog.Warnf("homunculus delete failed: %v", err)
		return
	}
	m.HomunculusDeleteID = currentHomunculusID(ctx)
}

func (m *System) sendHomunculusRename(ctx client.Context, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if ctx.Network == nil {
		m.bindings.Error("Homunculus rename failed: not connected.")
		return
	}
	if err := ctx.Network.SendHomunculusRename(name); err != nil {
		m.bindings.Error("Homunculus rename failed.")
		glog.Warnf("homunculus rename failed name=%q: %v", name, err)
	}
}

func (m *System) ApplyHomunculusFeedResultMessage(ctx client.Context, result network.HomunculusFeedResult) {
	if result.Result {
		return
	}
	name := fmt.Sprintf("item %d", result.ItemID)
	if ctx.Resources != nil {
		if resolved, ok := ctx.Resources.ItemDisplayName(int(result.ItemID), true); ok && strings.TrimSpace(resolved) != "" {
			name = resolved
		}
	}
	m.bindings.Error("Failed to feed homunculus with %s.", name)
}

func (m *System) toggleHomunculusAssist(ctx client.Context) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.HomunculusAggressive = !ctx.Session.HomunculusAggressive
	glog.Debugf("homunculus assist aggressive=%t", ctx.Session.HomunculusAggressive)
}

func homunculusAggressive(ctx client.Context) bool {
	return ctx.Session != nil && ctx.Session.HomunculusAggressive
}

func currentHomunculusID(ctx client.Context) uint32 {
	if ctx.Session != nil && ctx.Session.Homunculus.ID != 0 {
		return ctx.Session.Homunculus.ID
	}
	return FindActorID(ctx, scene.ActorObjectTypeHomunculus)
}

func (m *System) ClearDeletedHomunculus(ctx client.Context) {
	id := m.HomunculusDeleteID
	if id == 0 && ctx.Session != nil {
		id = ctx.Session.Homunculus.ID
	}
	m.HomunculusDeleteID = 0
	if ctx.Session != nil {
		ctx.Session.Homunculus = session.Companion{}
	}
	if id != 0 {
		m.aiSystem().clearMessages(id)
	}
	m.bindings.Windows.HomunculusInfo.Close()
	m.bindings.Windows.HomunculusSkill.Close()
	m.bindings.Windows.HomunculusContext.Close()
}

func (m *System) OpenHomunculusContext(ctx client.Context, actor worldstate.Actor) {
	if ctx.Session != nil {
		ctx.Session.Homunculus.ID = actor.ID
		ctx.Session.Homunculus.Active = true
		ctx.Session.Homunculus.Job = actor.Job
	}
	glog.Debugf("homunculus context target id=%d name=%q job=%d", actor.ID, actor.Name, actor.Job)
	m.bindings.Windows.HomunculusContext.Open(ctx, ctx.Input.MouseX, ctx.Input.MouseY, homunculusAggressive(ctx))
}
