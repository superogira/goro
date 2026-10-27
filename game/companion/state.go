package companion

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

const (
	MotionStand  = 0
	MotionMove   = 1
	MotionAttack = 2
	MotionDead   = 3

	HungerMax = 100
)

func ApplyActorEntry(ctx client.Context, entry network.ActorEntry) {
	if ctx.Session == nil || entry.ID == 0 || !entry.HasObjectType {
		return
	}
	switch entry.ObjectType {
	case scene.ActorObjectTypeHomunculus:
		ctx.Session.Homunculus.ID = entry.ID
		ctx.Session.Homunculus.Active = true
		ctx.Session.Homunculus.Job = entry.Job
	case scene.ActorObjectTypeMercenary:
		ctx.Session.Mercenary.ID = entry.ID
		ctx.Session.Mercenary.Active = true
		ctx.Session.Mercenary.Job = entry.Job
	}
}

func ActorEntryName(ctx client.Context, entry network.ActorEntry) string {
	if ctx.Session == nil || !entry.HasObjectType {
		return ""
	}
	switch entry.ObjectType {
	case scene.ActorObjectTypeHomunculus:
		if ctx.Session.Homunculus.ID != 0 && ctx.Session.Homunculus.ID != entry.ID {
			return ""
		}
		return scene.SanitizeActorName(ctx.Session.Homunculus.Name)
	case scene.ActorObjectTypeMercenary:
		if ctx.Session.Mercenary.ID != 0 && ctx.Session.Mercenary.ID != entry.ID {
			return ""
		}
		return scene.SanitizeActorName(ctx.Session.Mercenary.Name)
	default:
		return ""
	}
}

func (m *System) ApplyHomunculusProperty(ctx client.Context, property network.HomunculusProperty) {
	if ctx.Session == nil {
		return
	}
	hom := &ctx.Session.Homunculus
	hom.Active = true
	hom.Name = property.Name
	hom.Flags = property.Flags
	hom.Level = property.Level
	hom.Hunger = property.Hunger
	hom.Intimacy = property.Intimacy
	hom.ItemID = property.ItemID
	hom.Attack = property.Attack
	hom.MagicAttack = property.MagicAttack
	hom.Hit = property.Hit
	hom.Critical = property.Critical
	hom.Defense = property.Defense
	hom.MagicDefense = property.MagicDefense
	hom.Flee = property.Flee
	hom.ASPD = property.ASPD
	hom.HP = property.HP
	hom.MaxHP = property.MaxHP
	hom.SP = property.SP
	hom.MaxSP = property.MaxSP
	hom.Exp = property.Exp
	hom.MaxExp = property.MaxExp
	hom.Skills.Points = property.SkillPoints
	hom.AttackRange = property.AttackRange
	if hom.ID == 0 {
		hom.ID = FindActorID(ctx, scene.ActorObjectTypeHomunculus)
	}
	applyCompanionActorName(ctx, hom.ID, hom.Name)
	m.bindings.SetLife(ctx, hom.ID, property.HP, property.MaxHP, property.SP, property.MaxSP, property.AttackRange)
	m.bindings.SetHunger(ctx, hom.ID, property.Hunger, HungerMax)
	glog.Debugf("homunculus property id=%d name=%q level=%d hp=%d/%d sp=%d/%d hunger=%d intimacy=%d range=%d skills=%d", hom.ID, hom.Name, hom.Level, hom.HP, hom.MaxHP, hom.SP, hom.MaxSP, hom.Hunger, hom.Intimacy, hom.AttackRange, hom.Skills.Points)
}

func (m *System) ApplyHomunculusStateChange(ctx client.Context, change network.HomunculusStateChange) {
	if ctx.Session == nil || change.GID == 0 {
		return
	}
	hom := &ctx.Session.Homunculus
	hom.ID = change.GID
	hom.Active = true
	applyCompanionActorName(ctx, hom.ID, hom.Name)
	switch change.State {
	case 0:
		// Pre-init: robrowser uses this packet to learn Session.homunId.
	case 1:
		hom.Intimacy = int(change.Data)
	case 2:
		hom.Hunger = int(change.Data)
		m.bindings.SetHunger(ctx, hom.ID, hom.Hunger, HungerMax)
	}
	glog.Debugf("homunculus state id=%d type=%d state=%d data=%d", change.GID, change.Type, change.State, change.Data)
}

func (m *System) ApplyHomunculusParamChange(ctx client.Context, change network.HomunculusParamChange) {
	if ctx.Session == nil || !ctx.Session.Homunculus.Active {
		return
	}
	hom := &ctx.Session.Homunculus
	previousLevel := hom.Level
	value := int(change.Value)
	switch change.Param {
	case network.StatusSpeed:
		applyCompanionSpeed(ctx, hom.ID, value)
	case network.StatusBaseExp:
		hom.Exp = nonNegativeUint64(change.Value)
	case network.StatusHP:
		hom.HP = value
	case network.StatusMaxHP:
		hom.MaxHP = value
	case network.StatusSP:
		hom.SP = value
	case network.StatusMaxSP:
		hom.MaxSP = value
	case network.StatusBaseLevel:
		if previousLevel > 0 && value > previousLevel {
			if hom.ID == 0 {
				hom.ID = FindActorID(ctx, scene.ActorObjectTypeHomunculus)
			}
			m.bindings.LevelUp(ctx, hom.ID)
		}
		hom.Level = value
	case network.StatusSkillPoint:
		hom.Skills.Points = value
	case network.StatusNextBaseExp:
		hom.MaxExp = nonNegativeUint64(change.Value)
	default:
		glog.Debugf("homunculus param ignored param=%d value=%d", change.Param, change.Value)
		return
	}
	m.bindings.SetLife(ctx, hom.ID, hom.HP, hom.MaxHP, hom.SP, hom.MaxSP, hom.AttackRange)
	glog.Debugf("homunculus param id=%d param=%d value=%d hp=%d/%d sp=%d/%d exp=%d/%d skills=%d", hom.ID, change.Param, change.Value, hom.HP, hom.MaxHP, hom.SP, hom.MaxSP, hom.Exp, hom.MaxExp, hom.Skills.Points)
}

func ApplyHomunculusSkillInfoList(ctx client.Context, list network.HomunculusSkillInfoList) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Homunculus.Skills.List = ctx.Session.Homunculus.Skills.List[:0]
	for _, skill := range list.Skills {
		ctx.Session.Homunculus.Skills.List = append(ctx.Session.Homunculus.Skills.List, scene.SessionSkillFromNetworkWithResources(ctx.Resources, skill))
	}
	glog.Debugf("homunculus skill list received count=%d points=%d", len(ctx.Session.Homunculus.Skills.List), ctx.Session.Homunculus.Skills.Points)
}

func ApplyHomunculusSkillInfoUpdate(ctx client.Context, update network.HomunculusSkillInfoUpdate) {
	if ctx.Session == nil {
		return
	}
	upsertCompanionSkill(&ctx.Session.Homunculus, scene.SessionSkillFromNetworkWithResources(ctx.Resources, update.Skill))
	glog.Debugf("homunculus skill update id=%d level=%d sp=%d range=%d upgradable=%t", update.Skill.ID, update.Skill.Level, update.Skill.SPCost, update.Skill.Range, update.Skill.Upgradable)
}

func (m *System) ApplyMercenaryProperty(ctx client.Context, property network.MercenaryProperty) {
	if ctx.Session == nil {
		return
	}
	merc := &ctx.Session.Mercenary
	merc.Active = true
	if property.ID != 0 {
		merc.ID = property.ID
	}
	if merc.ID == 0 {
		merc.ID = FindActorID(ctx, scene.ActorObjectTypeMercenary)
	}
	if property.Name != "" {
		merc.Name = property.Name
	}
	merc.Level = property.Level
	merc.Faith = property.Faith
	merc.SummonCount = property.SummonCount
	merc.Calls = property.Calls
	merc.Kills = property.Kills
	merc.ExpireTick = property.ExpireTick
	merc.Attack = property.Attack
	merc.MagicAttack = property.MagicAttack
	merc.Hit = property.Hit
	merc.Critical = property.Critical
	merc.Defense = property.Defense
	merc.MagicDefense = property.MagicDefense
	merc.Flee = property.Flee
	merc.ASPD = property.ASPD
	merc.HP = property.HP
	merc.MaxHP = property.MaxHP
	merc.SP = property.SP
	merc.MaxSP = property.MaxSP
	merc.Exp = uint64(property.Exp)
	if property.AttackRange > 0 {
		merc.AttackRange = property.AttackRange
	}
	applyCompanionActorName(ctx, merc.ID, merc.Name)
	m.bindings.SetLife(ctx, merc.ID, merc.HP, merc.MaxHP, merc.SP, merc.MaxSP, merc.AttackRange)
	glog.Debugf("mercenary property id=%d name=%q level=%d hp=%d/%d sp=%d/%d faith=%d kills=%d range=%d", merc.ID, merc.Name, merc.Level, merc.HP, merc.MaxHP, merc.SP, merc.MaxSP, merc.Faith, merc.Kills, merc.AttackRange)
}

func (m *System) ApplyMercenaryParamChange(ctx client.Context, change network.MercenaryParamChange) {
	if ctx.Session == nil || !ctx.Session.Mercenary.Active {
		return
	}
	merc := &ctx.Session.Mercenary
	value := int(change.Value)
	switch change.Param {
	case network.StatusHP:
		merc.HP = value
	case network.StatusMaxHP:
		merc.MaxHP = value
	case network.StatusSP:
		merc.SP = value
	case network.StatusMaxSP:
		merc.MaxSP = value
	case network.StatusMercFlee:
		merc.Flee = value
	case network.StatusMercKills:
		merc.Kills = uint32(scene.MaxInt(0, value))
	case network.StatusMercFaith:
		merc.Faith = value
	default:
		glog.Debugf("mercenary param ignored param=%d value=%d", change.Param, value)
		return
	}
	m.bindings.SetLife(ctx, merc.ID, merc.HP, merc.MaxHP, merc.SP, merc.MaxSP, merc.AttackRange)
	glog.Debugf("mercenary param id=%d param=%d value=%d hp=%d/%d sp=%d/%d", merc.ID, change.Param, value, merc.HP, merc.MaxHP, merc.SP, merc.MaxSP)
}

func ApplyMercenarySkillInfoList(ctx client.Context, list network.MercenarySkillInfoList) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Mercenary.Skills.List = ctx.Session.Mercenary.Skills.List[:0]
	for _, skill := range list.Skills {
		ctx.Session.Mercenary.Skills.List = append(ctx.Session.Mercenary.Skills.List, scene.SessionSkillFromNetworkWithResources(ctx.Resources, skill))
	}
	glog.Debugf("mercenary skill list received count=%d", len(ctx.Session.Mercenary.Skills.List))
}

func ApplyMercenarySkillInfoUpdate(ctx client.Context, update network.MercenarySkillInfoUpdate) {
	if ctx.Session == nil {
		return
	}
	upsertCompanionSkill(&ctx.Session.Mercenary, scene.SessionSkillFromNetworkWithResources(ctx.Resources, update.Skill))
	glog.Debugf("mercenary skill update id=%d level=%d sp=%d range=%d upgradable=%t", update.Skill.ID, update.Skill.Level, update.Skill.SPCost, update.Skill.Range, update.Skill.Upgradable)
}

func upsertCompanionSkill(companion *session.Companion, skill session.Skill) {
	if companion == nil {
		return
	}
	for i := range companion.Skills.List {
		if companion.Skills.List[i].ID != skill.ID {
			continue
		}
		companion.Skills.List[i] = skill
		return
	}
	companion.Skills.List = append(companion.Skills.List, skill)
}

func FindActorID(ctx client.Context, objectType uint8) uint32 {
	if ctx.World == nil {
		return 0
	}
	for id, actor := range ctx.World.Actors {
		if actor.HasObjectType && actor.ObjectType == objectType {
			return id
		}
	}
	return 0
}

func applyCompanionActorName(ctx client.Context, id uint32, name string) {
	if id == 0 || ctx.World == nil {
		return
	}
	name = scene.SanitizeActorName(name)
	if name == "" {
		return
	}
	actor, ok := ctx.World.Actors[id]
	if !ok {
		return
	}
	actor.Name = name
	ctx.World.Actors[id] = actor
}

func applyCompanionSpeed(ctx client.Context, id uint32, speed int) {
	if id == 0 || ctx.World == nil {
		return
	}
	if actor, ok := ctx.World.Actors[id]; ok {
		actor.Speed = speed
		ctx.World.Actors[id] = actor
	}
}

func nonNegativeUint64(value int64) uint64 {
	if value <= 0 {
		return 0
	}
	return uint64(value)
}

func SetActorMotion(ctx client.Context, id uint32, motion int, targetID uint32, expires time.Time) {
	if id == 0 || ctx.World == nil {
		return
	}
	if scene.IsLocalActor(ctx, id) {
		ctx.World.Player.AIMotion = motion
		ctx.World.Player.HasAIMotion = true
		ctx.World.Player.AIMotionExpires = expires
		ctx.World.Player.AITargetID = targetID
		return
	}
	actor, ok := ctx.World.Actors[id]
	if !ok {
		return
	}
	actor.AIMotion = motion
	actor.HasAIMotion = true
	actor.AIMotionExpires = expires
	actor.AITargetID = targetID
	ctx.World.Actors[id] = actor
}

func Life(ctx client.Context, id uint32) (hp, maxHP, sp, maxSP int, ok bool) {
	if ctx.Session != nil {
		if id == ctx.Session.Homunculus.ID && ctx.Session.Homunculus.Active {
			hom := ctx.Session.Homunculus
			return hom.HP, hom.MaxHP, hom.SP, hom.MaxSP, true
		}
		if id == ctx.Session.Mercenary.ID && ctx.Session.Mercenary.Active {
			merc := ctx.Session.Mercenary
			return merc.HP, merc.MaxHP, merc.SP, merc.MaxSP, true
		}
	}
	return 0, 0, 0, 0, false
}
