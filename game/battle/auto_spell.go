package battle

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
)

func (m *System) ApplyAutoSpellList(ctx client.Context, list network.AutoSpellList, assets gameui.AssetProvider) {
	if len(list.SkillIDs) == 0 {
		m.sendAutoSpellSelection(ctx, 0)
		return
	}
	m.bindings.Windows.AutoSpell.OpenList(ctx, list, assets)
	glog.Debugf("auto spell choices skills=%v", list.SkillIDs)
}

func (m *System) UpdateAutoSpellWindow(ctx client.Context) bool {
	consumed := m.bindings.Windows.AutoSpell.Update(ctx)
	if skillID, ok := m.bindings.Windows.AutoSpell.PopSelection(); ok {
		m.sendAutoSpellSelection(ctx, skillID)
		return true
	}
	return consumed
}

func (m *System) sendAutoSpellSelection(ctx client.Context, skillID uint16) {
	if ctx.Network == nil {
		m.bindings.Console.AddErrorMessage("Auto Spell selection failed: not connected.")
		return
	}
	if err := ctx.Network.SendSelectAutoSpell(skillID); err != nil {
		m.bindings.Console.AddErrorMessage("Auto Spell selection failed.")
		return
	}
	glog.Debugf("auto spell selected skill=%d", skillID)
}
