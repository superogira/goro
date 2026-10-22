package bot

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
	lua "github.com/yuin/gopher-lua"
)

func autoRevive(ctx client.Context) bool {
	if err := client.RequestAutoRevive(ctx); err != nil {
		glog.Debugf("script auto-revive failed: %v", err)
		return false
	}
	return true
}

func sendMessage(ctx client.Context, message string) bool {
	if err := client.SendChat(ctx, message); err != nil {
		glog.Debugf("script message failed: %v", err)
		return false
	}
	return true
}

func skillByName(ctx client.Context, name string) (session.Skill, bool) {
	if ctx.Session == nil {
		return session.Skill{}, false
	}
	needle := normalizeLuaSkillName(name)
	if needle == "" {
		return session.Skill{}, false
	}
	for _, skill := range ctx.Session.Skills.List {
		if normalizeLuaSkillName(skill.Name) == needle || normalizeLuaSkillName(db.SkillResourceName[skill.ID]) == needle {
			return skill, true
		}
	}
	return session.Skill{}, false
}

func normalizeLuaSkillName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, "_", " ")
	return strings.Join(strings.Fields(name), " ")
}

func actorTable(L *lua.LState, actor worldstate.Actor, playerX, playerY int) *lua.LTable {
	row := L.NewTable()
	row.RawSetString("id", lua.LNumber(actor.ID))
	row.RawSetString("name", lua.LString(actor.Name))
	row.RawSetString("x", lua.LNumber(actor.X))
	row.RawSetString("y", lua.LNumber(actor.Y))
	row.RawSetString("job", lua.LNumber(actor.Job))
	row.RawSetString("distance", lua.LNumber(cellDistance(playerX, playerY, actor.X, actor.Y)))
	return row
}

func companionOwnedByPlayer(s *session.Session, actor worldstate.Actor) bool {
	if s == nil {
		return false
	}
	if actor.ObjectType == scene.ActorObjectTypeHomunculus {
		return actor.ID == s.Homunculus.ID
	}
	return actor.ObjectType == scene.ActorObjectTypeMercenary && actor.ID == s.Mercenary.ID
}

func itemList(L *lua.LState, ctx client.Context) *lua.LTable {
	result := L.NewTable()
	if ctx.World == nil {
		return result
	}
	now := time.Now()
	playerX, playerY := scene.CurrentPlayerCell(ctx, now)
	ids := make([]int, 0, len(ctx.World.Items))
	for id := range ctx.World.Items {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	for _, id := range ids {
		item := ctx.World.Items[uint32(id)]
		row := L.NewTable()
		row.RawSetString("id", lua.LNumber(item.ID))
		row.RawSetString("item_id", lua.LNumber(item.ItemID))
		row.RawSetString("amount", lua.LNumber(item.Amount))
		row.RawSetString("x", lua.LNumber(item.X))
		row.RawSetString("y", lua.LNumber(item.Y))
		row.RawSetString("identified", lua.LBool(item.Identified))
		row.RawSetString("distance", lua.LNumber(cellDistance(playerX, playerY, item.X, item.Y)))
		result.Append(row)
	}
	return result
}

func inventoryList(L *lua.LState, ctx client.Context) *lua.LTable {
	result := L.NewTable()
	if ctx.Session == nil {
		return result
	}
	items := append([]session.InventoryItem(nil), ctx.Session.Inventory.Items...)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Index < items[j].Index
	})
	for _, item := range items {
		row := L.NewTable()
		row.RawSetString("index", lua.LNumber(item.Index))
		row.RawSetString("item_id", lua.LNumber(item.ItemID))
		row.RawSetString("amount", lua.LNumber(item.Amount))
		row.RawSetString("identified", lua.LBool(item.Identified))
		row.RawSetString("usable", lua.LBool(db.ItemTypeIsUsable(item.Type)))
		result.Append(row)
	}
	return result
}

func playerTable(L *lua.LState, ctx client.Context) *lua.LTable {
	result := L.NewTable()
	now := time.Now()
	x, y := scene.CurrentPlayerCell(ctx, now)
	hp, maxHP := hp(ctx)
	sp, maxSP := sp(ctx)
	if ctx.Session != nil {
		result.RawSetString("id", lua.LNumber(scene.LocalSkillTarget(ctx)))
	}
	result.RawSetString("x", lua.LNumber(x))
	result.RawSetString("y", lua.LNumber(y))
	result.RawSetString("hp", lua.LNumber(hp))
	result.RawSetString("max_hp", lua.LNumber(maxHP))
	result.RawSetString("sp", lua.LNumber(sp))
	result.RawSetString("max_sp", lua.LNumber(maxSP))
	result.RawSetString("dead", lua.LBool(ctx.Session != nil && ctx.Session.Dead))
	result.RawSetString("moving", lua.LBool(ctx.World != nil && scene.ActorIsMovingAt(ctx.World.Player, now)))
	return result
}

func hp(ctx client.Context) (int, int) {
	if ctx.Session == nil {
		return 0, 0
	}
	return ctx.Session.Vitals.HP, ctx.Session.Vitals.MaxHP
}

func sp(ctx client.Context) (int, int) {
	if ctx.Session == nil {
		return 0, 0
	}
	return ctx.Session.Vitals.SP, ctx.Session.Vitals.MaxSP
}

func cellDistance(ax, ay, bx, by int) float64 {
	return math.Hypot(float64(bx-ax), float64(by-ay))
}

func optionalActorID(L *lua.LState, index int) uint32 {
	if L.Get(index) == lua.LNil {
		return 0
	}
	value := float64(L.CheckNumber(index))
	if value < 0 || value > float64(^uint32(0)) || value != math.Trunc(value) {
		L.ArgError(index, "actor id must be a non-negative integer")
		return 0
	}
	return uint32(value)
}

func (b *Script) registerAPI(api *lua.LTable) {
	world := b.bindings.World
	b.state.SetFuncs(api, map[string]lua.LGFunction{
		"enemies": func(L *lua.LState) int {
			ctx := b.worldContext()
			L.Push(b.enemyList(L, ctx))
			return 1
		},
		"players": func(L *lua.LState) int {
			ctx := b.worldContext()
			L.Push(b.playerList(L, ctx))
			return 1
		},
		"companions": func(L *lua.LState) int {
			ctx := b.worldContext()
			L.Push(b.companionList(L, ctx))
			return 1
		},
		"items": func(L *lua.LState) int {
			ctx := b.worldContext()
			L.Push(itemList(L, ctx))
			return 1
		},
		"inventory": func(L *lua.LState) int {
			ctx := b.worldContext()
			L.Push(inventoryList(L, ctx))
			return 1
		},
		"use_item": func(L *lua.LState) int {
			ctx := b.worldContext()
			L.Push(lua.LBool(world != nil && world.UseItem(ctx, L.CheckInt(1))))
			return 1
		},
		"revive": func(L *lua.LState) int {
			ctx := b.worldContext()
			L.Push(lua.LBool(world != nil && autoRevive(ctx)))
			return 1
		},
		"message": func(L *lua.LState) int {
			ctx := b.worldContext()
			L.Push(lua.LBool(world != nil && sendMessage(ctx, L.CheckString(1))))
			return 1
		},
		"walk": func(L *lua.LState) int {
			ctx := b.worldContext()
			L.Push(lua.LBool(world != nil && world.Walk(ctx, L.CheckInt(1), L.CheckInt(2))))
			return 1
		},
		"stop": func(L *lua.LState) int {
			ctx := b.worldContext()
			L.Push(lua.LBool(world != nil && world.Stop(ctx)))
			return 1
		},
		"attack": func(L *lua.LState) int {
			ctx := b.worldContext()
			id := uint32(L.CheckInt(1))
			L.Push(lua.LBool(world != nil && world.Attack(ctx, id)))
			return 1
		},
		"target": func(L *lua.LState) int {
			ctx := b.worldContext()
			id := uint32(L.CheckInt(1))
			L.Push(lua.LBool(world != nil && world.Attack(ctx, id)))
			return 1
		},
		"skill": func(L *lua.LState) int {
			ctx := b.worldContext()
			id := uint32(L.CheckInt(1))
			skillArg := L.Get(2)
			if skillArg == lua.LNil {
				L.ArgError(2, "skill id or name expected")
				return 0
			}
			level := -1
			if L.GetTop() >= 3 && L.Get(3) != lua.LNil {
				level = L.CheckInt(3)
			}
			L.Push(lua.LBool(world != nil && b.useSkill(ctx, id, skillArg, level)))
			return 1
		},
		"pending_skill": func(L *lua.LState) int {
			ctx := b.worldContext()
			L.Push(b.pendingSkill(L, ctx))
			return 1
		},
		"skill_targets": func(L *lua.LState) int {
			ctx := b.worldContext()
			L.Push(b.skillTargets(L, ctx, L.OptBool(1, false)))
			return 1
		},
		"use_pending_skill": func(L *lua.LState) int {
			ctx := b.worldContext()
			id := optionalActorID(L, 1)
			L.Push(lua.LBool(world != nil && world.UsePendingSkill(ctx, id)))
			return 1
		},
		"highlight_actor": func(L *lua.LState) int {
			ctx := b.worldContext()
			id := optionalActorID(L, 1)
			L.Push(lua.LBool(world != nil && world.HighlightActor(ctx, id)))
			return 1
		},
		"loot": func(L *lua.LState) int {
			ctx := b.worldContext()
			id := uint32(L.CheckInt(1))
			L.Push(lua.LBool(world != nil && world.Loot(ctx, id)))
			return 1
		},
		"hp": func(L *lua.LState) int {
			ctx := b.worldContext()
			hp, maxHP := hp(ctx)
			L.Push(lua.LNumber(hp))
			L.Push(lua.LNumber(maxHP))
			return 2
		},
		"sp": func(L *lua.LState) int {
			ctx := b.worldContext()
			sp, maxSP := sp(ctx)
			L.Push(lua.LNumber(sp))
			L.Push(lua.LNumber(maxSP))
			return 2
		},
		"player": func(L *lua.LState) int {
			ctx := b.worldContext()
			player := playerTable(L, ctx)
			if world != nil {
				player.RawSetString("walk_sequence", lua.LNumber(world.WalkSequence()))
			}
			L.Push(player)
			return 1
		},
	})
	b.registerControlsAPI(api)
}

func (b *Script) enemyList(L *lua.LState, ctx client.Context) *lua.LTable {
	result := L.NewTable()
	if ctx.World == nil {
		return result
	}
	now := time.Now()
	playerX, playerY := scene.CurrentPlayerCell(ctx, now)
	ids := make([]int, 0, len(ctx.World.Actors))
	for id, actor := range ctx.World.Actors {
		if b.bindings.World.ActorDead(id) {
			continue
		}
		if b.bindings.World.CanAttack(ctx, actor) {
			ids = append(ids, int(id))
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		actor := ctx.World.Actors[uint32(id)]
		row := actorTable(L, actor, playerX, playerY)
		row.RawSetString("object_type", lua.LNumber(actor.ObjectType))
		result.Append(row)
	}
	return result
}

func (b *Script) playerList(L *lua.LState, ctx client.Context) *lua.LTable {
	result := L.NewTable()
	if ctx.World == nil {
		return result
	}
	playerX, playerY := scene.CurrentPlayerCell(ctx, time.Now())
	ids := make([]int, 0, len(ctx.World.Actors))
	for id, actor := range ctx.World.Actors {
		if scene.ActorRepresentsPlayer(actor) && !scene.IsLocalActor(ctx, actor.ID) {
			ids = append(ids, int(id))
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		actor := ctx.World.Actors[uint32(id)]
		row := actorTable(L, actor, playerX, playerY)
		member := luaPartyMember(ctx.Session, actor.ID)
		row.RawSetString("party_member", lua.LBool(member != nil))
		dead := b.bindings.World.ActorDead(actor.ID)
		if member != nil {
			row.RawSetString("hp", lua.LNumber(member.HP))
			row.RawSetString("max_hp", lua.LNumber(member.MaxHP))
			dead = dead || member.Dead
		} else {
			row.RawSetString("hp", lua.LNumber(0))
			row.RawSetString("max_hp", lua.LNumber(0))
		}
		row.RawSetString("dead", lua.LBool(dead))
		result.Append(row)
	}
	return result
}

func (b *Script) companionList(L *lua.LState, ctx client.Context) *lua.LTable {
	result := L.NewTable()
	if ctx.World == nil {
		return result
	}
	playerX, playerY := scene.CurrentPlayerCell(ctx, time.Now())
	ids := make([]int, 0, 2)
	for id, actor := range ctx.World.Actors {
		if actor.HasObjectType && (actor.ObjectType == scene.ActorObjectTypeHomunculus || actor.ObjectType == scene.ActorObjectTypeMercenary) {
			ids = append(ids, int(id))
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		actor := ctx.World.Actors[uint32(id)]
		row := actorTable(L, actor, playerX, playerY)
		kind := "homunculus"
		if actor.ObjectType == scene.ActorObjectTypeMercenary {
			kind = "mercenary"
		}
		row.RawSetString("kind", lua.LString(kind))
		row.RawSetString("own", lua.LBool(companionOwnedByPlayer(ctx.Session, actor)))
		hp, maxHP, sp, maxSP, _ := b.bindings.World.CompanionLife(ctx, actor.ID)
		row.RawSetString("hp", lua.LNumber(hp))
		row.RawSetString("max_hp", lua.LNumber(maxHP))
		row.RawSetString("sp", lua.LNumber(sp))
		row.RawSetString("max_sp", lua.LNumber(maxSP))
		dead := b.bindings.World.ActorDead(actor.ID)
		row.RawSetString("dead", lua.LBool(dead))
		result.Append(row)
	}
	return result
}

func luaPartyMember(s *session.Session, actorID uint32) *session.PartyMember {
	if s == nil || actorID == 0 {
		return nil
	}
	for i := range s.Party.Members {
		if s.Party.Members[i].AccountID == actorID {
			return &s.Party.Members[i]
		}
	}
	return nil
}

func resolveSkill(ctx client.Context, skillArg lua.LValue) (session.Skill, bool) {
	switch value := skillArg.(type) {
	case lua.LNumber:
		if value <= 0 || value > 65535 {
			return session.Skill{}, false
		}
		id := uint16(value)
		if id == 0 {
			return session.Skill{}, false
		}
		if ctx.Session != nil {
			for _, skill := range ctx.Session.Skills.List {
				if skill.ID == id {
					return skill, true
				}
			}
		}
		return session.Skill{}, false
	case lua.LString:
		return skillByName(ctx, string(value))
	default:
		return session.Skill{}, false
	}
}

func (b *Script) useSkill(ctx client.Context, id uint32, arg lua.LValue, level int) bool {
	if ctx.World == nil {
		return false
	}
	skill, ok := resolveSkill(ctx, arg)
	if !ok {
		glog.Debugf("script skill unavailable target=%d skill=%s", id, arg.String())
		return false
	}
	return b.bindings.World.Skill(ctx, id, skill, level)
}

func (b *Script) skillTargets(L *lua.LState, ctx client.Context, ignoreShift bool) *lua.LTable {
	result := L.NewTable()
	if b.bindings.World == nil {
		return result
	}
	now := time.Now()
	x, y := scene.CurrentPlayerCell(ctx, now)
	for _, actor := range b.bindings.World.SkillTargets(ctx, ignoreShift, now) {
		result.Append(actorTable(L, actor, x, y))
	}
	return result
}

func (b *Script) pendingSkill(L *lua.LState, ctx client.Context) lua.LValue {
	if b.bindings.World == nil {
		return lua.LNil
	}
	pending, ok := b.bindings.World.PendingSkill(ctx)
	if !ok {
		return lua.LNil
	}
	result := L.NewTable()
	result.RawSetString("id", lua.LNumber(pending.Skill.ID))
	result.RawSetString("name", lua.LString(pending.Skill.Name))
	result.RawSetString("level", lua.LNumber(pending.Skill.Level))
	result.RawSetString("max_level", lua.LNumber(pending.MaxLevel))
	result.RawSetString("type", lua.LNumber(pending.Skill.Type))
	result.RawSetString("range", lua.LNumber(pending.Skill.Range))
	result.RawSetString("target", lua.LString(pending.Target))
	if pending.CasterKind != "" {
		result.RawSetString("caster_id", lua.LNumber(pending.CasterID))
		result.RawSetString("caster_kind", lua.LString(pending.CasterKind))
		result.RawSetString("caster_x", lua.LNumber(pending.CasterX))
		result.RawSetString("caster_y", lua.LNumber(pending.CasterY))
	}
	return result
}
