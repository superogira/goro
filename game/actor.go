package game

import (
	"image/color"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/effects"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

// robr stores entity positions as cells and adds +0.5 in SpriteRenderer.vs.
// Goro projects world coordinates directly, so actor-like entities use cell centers.
func cellWorldAnchor(x, y float64) (float64, float64) {
	return scene.CellCenter(x), scene.CellCenter(y)
}

func actorWorldAnchor(_ worldstate.Actor, x, y float64) (float64, float64) {
	return cellWorldAnchor(x, y)
}

func pointInActorPickBounds(mouseX, mouseY, centerX, centerY, scale float64) bool {
	scale = normalizePickScale(scale)
	left := centerX - 44*scale
	right := centerX + 44*scale
	top := centerY - float64(scene.HumanoidBillboardAnchorY)*scale
	bottom := centerY + 20*scale
	return mouseX >= left && mouseX <= right && mouseY >= top && mouseY <= bottom
}

func actorPickBoundsCenter(centerX, centerY, scale float64) (float64, float64) {
	scale = normalizePickScale(scale)
	top := centerY - float64(scene.HumanoidBillboardAnchorY)*scale
	bottom := centerY + 20*scale
	return centerX, (top + bottom) / 2
}

func normalizePickScale(scale float64) float64 {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return 1
	}
	return scale
}

func actorCanBeSkillTargeted(ctx client.Context, skill session.Skill, actor worldstate.Actor) bool {
	return actorCanBeSkillTargetedWithOverride(ctx, skill, actor, skillTargetOverrideActive(ctx))
}

func actorCanBeSkillTargetedWithOverride(ctx client.Context, skill session.Skill, actor worldstate.Actor, override bool) bool {
	if actor.ID == 0 || isWarpActor(actor) {
		return false
	}
	if scene.ActorHasStealth(actor) && !scene.IsLocalActor(ctx, actor.ID) {
		return false
	}
	targetFlags, ok := skillTargetFlagsForActor(ctx, actor)
	if !ok {
		return false
	}
	if skill.Type == skillTargetPet {
		return targetFlags&skillTargetPet != 0
	}
	if skill.Type&targetFlags != 0 {
		if scene.IsLocalActor(ctx, actor.ID) && skill.Type&skillTargetEnemy != 0 {
			return false
		}
		return true
	}
	if override || skillTargetMapStateAllowsMismatch(ctx, actor) {
		if scene.IsLocalActor(ctx, actor.ID) && skill.Type&skillTargetEnemy != 0 {
			return false
		}
		return true
	}
	return false
}

func skillCanTargetDeadActor(skill session.Skill, actor worldstate.Actor) bool {
	return skill.ID == db.SkillALLResurrection && scene.ActorRepresentsPlayer(actor)
}

func skillTargetFlagsForActor(ctx client.Context, actor worldstate.Actor) (uint32, bool) {
	if actor.ID == 0 || isWarpActor(actor) || actor.Job == scene.ActorJobHiddenWarpNPC {
		return 0, false
	}
	if scene.IsLocalActor(ctx, actor.ID) {
		return skillTargetFriend, true
	}
	if actor.HasObjectType {
		switch actor.ObjectType {
		case scene.ActorObjectTypePC, scene.ActorObjectTypeElemental:
			return skillTargetFriend, true
		case scene.ActorObjectTypeHomunculus, scene.ActorObjectTypeMercenary:
			return skillTargetFriend | skillTargetHomun, true
		case scene.ActorObjectTypeMob, scene.ActorObjectTypeUnit, scene.ActorObjectTypeNPCABR, scene.ActorObjectTypeNPCBionic:
			return skillTargetEnemy | skillTargetPet, true
		default:
			return 0, false
		}
	}
	if res.HasPlayerJobToken(int(actor.Job)) {
		return skillTargetFriend, true
	}
	return 0, false
}

func skillTargetMapStateAllowsMismatch(ctx client.Context, actor worldstate.Actor) bool {
	if ctx.World == nil {
		return false
	}
	if scene.ActorRepresentsPlayer(actor) {
		return actorCanBePlayerCombatTarget(ctx, actor)
	}
	return actorCanBeEnemyWoECompanion(ctx, actor)
}

func actorCanOpenPlayerContext(ctx client.Context, actor worldstate.Actor) bool {
	if scene.ActorHasStealth(actor) || scene.IsLocalActor(ctx, actor.ID) || strings.TrimSpace(actor.Name) == "" {
		return false
	}
	return scene.ActorRepresentsPlayer(actor)
}

func actorCanBeAttackClicked(ctx client.Context, actor worldstate.Actor) bool {
	if actor.ID == 0 || actor.Job == scene.ActorJobHiddenWarpNPC || scene.ActorHasStealth(actor) || scene.IsLocalActor(ctx, actor.ID) {
		return false
	}
	if scene.ActorRepresentsPlayer(actor) {
		return actorCanBePlayerCombatTarget(ctx, actor)
	}
	if actorCanBeEnemyWoECompanion(ctx, actor) {
		return true
	}
	return actorHasMobObjectType(actor)
}

func actorCanBePlayerCombatTarget(ctx client.Context, actor worldstate.Actor) bool {
	if ctx.World == nil || !scene.ActorRepresentsPlayer(actor) || scene.IsLocalActor(ctx, actor.ID) {
		return false
	}
	if ctx.World.MapProperty.IsPvP() {
		return true
	}
	if !ctx.World.MapProperty.IsGvG() {
		return false
	}
	return actor.GuildID == 0 || localGuildID(ctx) == 0 || actor.GuildID != localGuildID(ctx)
}

func actorCanBeEnemyWoECompanion(ctx client.Context, actor worldstate.Actor) bool {
	if ctx.World == nil || !ctx.World.MapProperty.IsGvG() || !actor.HasObjectType {
		return false
	}
	if actor.ObjectType != scene.ActorObjectTypeHomunculus && actor.ObjectType != scene.ActorObjectTypeMercenary {
		return false
	}
	if ctx.Session != nil {
		if actor.ID == ctx.Session.Homunculus.ID || actor.ID == ctx.Session.Mercenary.ID {
			return false
		}
	}
	return actor.GuildID == 0 || localGuildID(ctx) == 0 || actor.GuildID != localGuildID(ctx)
}

func localGuildID(ctx client.Context) uint32 {
	if ctx.Session == nil {
		return 0
	}
	if ctx.Session.Guild.ID != 0 {
		return ctx.Session.Guild.ID
	}
	return ctx.Session.GuildID
}

func upsertNetworkActor(ctx client.Context, entry network.ActorEntry) {
	if scene.IsLocalActor(ctx, entry.ID) {
		return
	}
	name := companion.ActorEntryName(ctx, entry)
	companion.ApplyActorEntry(ctx, entry)
	dir := entry.Dir
	if entry.Moving {
		dir = directionFromDelta(entry.FromX, entry.FromY, entry.ToX, entry.ToY, dir)
	}
	attackRange := 0
	if ctx.Session != nil {
		switch entry.ObjectType {
		case scene.ActorObjectTypeHomunculus:
			attackRange = ctx.Session.Homunculus.AttackRange
		case scene.ActorObjectTypeMercenary:
			attackRange = ctx.Session.Mercenary.AttackRange
		}
	}
	actor := worldstate.Actor{
		ID:               entry.ID,
		Name:             name,
		GuildID:          entry.GuildID,
		EmblemVersion:    entry.EmblemVersion,
		X:                entry.X,
		Y:                entry.Y,
		Dir:              dir,
		Job:              entry.Job,
		Head:             entry.Head,
		Weapon:           entry.Weapon,
		Shield:           entry.Shield,
		HeadTop:          entry.HeadTop,
		HeadMid:          entry.HeadMid,
		HeadLow:          entry.HeadLow,
		HeadPal:          entry.HeadPal,
		BodyPal:          entry.BodyPal,
		Sex:              entry.Sex,
		HeadDir:          entry.HeadDir,
		IsAdmin:          actorIsAdmin(ctx, entry.ID),
		Appearance:       entry.Appearance,
		Moving:           entry.Moving,
		FromX:            entry.FromX,
		FromY:            entry.FromY,
		ToX:              entry.ToX,
		ToY:              entry.ToY,
		MoveStartTick:    entry.MoveStartTick,
		HasMoveStartTick: entry.HasMoveStartTick,
		ObjectType:       entry.ObjectType,
		HasObjectType:    entry.HasObjectType,
		Speed:            entry.Speed,
		AttackRange:      attackRange,
		BodyState:        entry.BodyState,
		HealthState:      entry.HealthState,
		EffectState:      entry.EffectState,
		Opt3State:        entry.Opt3State,
		HasState:         entry.HasState,
		Level:            entry.Level,
		HasLevel:         entry.HasLevel,
	}
	if existing, ok := ctx.World.Actors[entry.ID]; ok {
		if !entry.HasOpt3State {
			actor.Opt3State = existing.Opt3State
		}
		if !actor.HasObjectType {
			actor.ObjectType = existing.ObjectType
			actor.HasObjectType = existing.HasObjectType
		}
		if actor.AttackRange <= 0 {
			actor.AttackRange = existing.AttackRange
		}
	}
	applyActorCartStateFromEffect(&actor)
	upsertActor(ctx, actor)
}

func (m *WorldMode) upsertNetworkActor(ctx client.Context, entry network.ActorEntry) {
	if ctx.World == nil {
		return
	}
	m.clearActorVanish(entry.ID)
	oldState := uint32(0)
	if existing, ok := ctx.World.Actors[entry.ID]; ok && existing.HasState {
		oldState = existing.EffectState
	}
	if entry.Moving {
		m.clearActorAction(ctx, entry.ID)
	}
	upsertNetworkActor(ctx, entry)
	m.syncActorSpiritEffects(ctx, entry.ID)
	m.effects().SyncActorEntryLevel99Aura(ctx, entry.ID)
	m.requestActorGuildEmblem(ctx, entry.GuildID, entry.EmblemVersion)
	if !entry.HasState {
		return
	}
	actor, ok := ctx.World.Actors[entry.ID]
	if !ok || !actor.HasState {
		return
	}
	m.applyActorEffectStateEffects(ctx, actor.ID, oldState, actor.EffectState)
}

func (m *WorldMode) applyWarpPortalEntry(ctx client.Context, entry network.ActorEntry) {
	effectID, ok := warpPortalActorEffectID(entry.Job)
	if !ok {
		return
	}
	m.removeOtherWarpPortalActorEffects(entry.ID, effectID)
	m.effects().AddBetweenIfMissing(ctx, effectID, entry.ID, 0, time.Now(), warpPortalActorEffectLifetime)
}

func warpPortalActorEffectID(job int16) (int, bool) {
	switch int(job) {
	case scene.ActorJobWarpPortal, scene.ActorJobWarpPortalActive:
		return effects.WarpZone2, true
	case scene.ActorJobWarpPortalWaiting:
		return effects.ReadyPortal, true
	default:
		return 0, false
	}
}

func (m *WorldMode) removeOtherWarpPortalActorEffects(actorID uint32, keepEffectID int) {
	for _, effectID := range []int{effects.WarpZone2, effects.ReadyPortal, effects.Portal} {
		if effectID != keepEffectID {
			m.effects().Remove(effectID, actorID)
		}
	}
}

const (
	actorVanishOutOfSight = 0
	actorVanishDeath      = 1
	actorVanishLogout     = 2
	actorVanishTeleport   = 3
)

const (
	actorVanishOutOfSightFadeDuration = time.Second
	warpPortalActorEffectLifetime     = 24 * time.Hour
)

type actorVanishFade struct {
	started  time.Time
	removeAt time.Time
}

func (m *WorldMode) applyActorVanish(ctx client.Context, vanish network.ActorVanish) {
	glog.Debugf("actor vanish id=%d reason=%d", vanish.ID, vanish.Reason)
	pendingHomunculusDelete := m.companionSystem.HomunculusDeleteID != 0 && vanish.ID == m.companionSystem.HomunculusDeleteID
	pendingMercenaryDelete := m.companionSystem.MercenaryDeleteID != 0 && vanish.ID == m.companionSystem.MercenaryDeleteID
	if m.attackFocusID == vanish.ID {
		m.clearAttackFocus()
	}
	if vanish.Reason == actorVanishDeath {
		m.clearActorVanish(vanish.ID)
		m.startActorDeath(ctx, vanish.ID)
		return
	}
	if vanish.Reason == actorVanishOutOfSight && !pendingHomunculusDelete && !pendingMercenaryDelete {
		if m.startActorOutOfSightVanish(ctx, vanish.ID) {
			return
		}
	}
	m.addActorVanishTeleportEffect(ctx, vanish)
	m.removeActorEffectStateEffects(vanish.ID)
	m.effects().RemoveLevel99AuraEffects(vanish.ID)
	m.removeActorNow(ctx, vanish.ID)
	if pendingHomunculusDelete {
		m.companions().ClearDeletedHomunculus(ctx)
	}
	if pendingMercenaryDelete {
		m.companions().ClearDeletedMercenary(ctx)
	}
}

func (m *WorldMode) startActorOutOfSightVanish(ctx client.Context, id uint32) bool {
	if ctx.World == nil {
		return false
	}
	actor, ok := ctx.World.Actors[id]
	if !ok {
		return false
	}
	now := time.Now()
	actor = freezeActorAtRenderedPosition(actor, now)
	ctx.World.Actors[id] = actor
	if m.actorVanishes == nil {
		m.actorVanishes = make(map[uint32]actorVanishFade)
	}
	m.actorVanishes[id] = actorVanishFade{
		started:  now,
		removeAt: now.Add(actorVanishOutOfSightFadeDuration),
	}
	m.removeActorEffectStateEffects(id)
	m.effects().RemoveLevel99AuraEffects(id)
	delete(m.actorLife, id)
	delete(m.actorCastBars, id)
	delete(m.speechBubbles, id)
	glog.Debugf("actor out-of-sight fade id=%d duration_ms=%d", id, actorVanishOutOfSightFadeDuration.Milliseconds())
	return true
}

func freezeActorAtRenderedPosition(actor worldstate.Actor, now time.Time) worldstate.Actor {
	x, y := scene.ActorRenderPosition(actor, now)
	actor.X = int(math.Round(x))
	actor.Y = int(math.Round(y))
	actor.Moving = false
	actor.FromX = actor.X
	actor.FromY = actor.Y
	actor.ToX = actor.X
	actor.ToY = actor.Y
	actor.MovePath = nil
	actor.HasMoveStart = false
	actor.MoveStartX = 0
	actor.MoveStartY = 0
	actor.WalkDistance = 0
	return actor
}

func (m *WorldMode) removeActorNow(ctx client.Context, id uint32) {
	if ctx.World != nil {
		ctx.World.RemoveActor(id)
	}
	m.clearRemovedActorState(id)
}

func (m *WorldMode) clearRemovedActorState(id uint32) {
	delete(m.effects().ActorBlurs, id)
	m.effects().RemoveActor(id)
	if m.scriptHighlight.id == id {
		m.clearScriptHighlight()
	}
	delete(m.actorAnims, id)
	delete(m.actorDeaths, id)
	delete(m.actorVanishes, id)
	delete(m.actorSoundFrames, id)
	delete(m.actorLife, id)
	delete(m.actorCastBars, id)
	delete(m.speechBubbles, id)
	delete(m.petAccessoryIDs, id)
}

func (m *WorldMode) clearActorVanish(id uint32) {
	delete(m.actorVanishes, id)
}

func (m *WorldMode) addActorVanishTeleportEffect(ctx client.Context, vanish network.ActorVanish) {
	if vanish.Reason != actorVanishLogout && vanish.Reason != actorVanishTeleport {
		return
	}
	if ctx.World == nil {
		return
	}
	now := time.Now()
	if scene.IsLocalActor(ctx, vanish.ID) {
		x, y := scene.ActorRenderPosition(ctx.World.Player, now)
		m.effects().AddCell(ctx, effects.Teleportation, 0, int(math.Round(x)), int(math.Round(y)), now, 0, 0)
		return
	}
	actor, ok := ctx.World.Actors[vanish.ID]
	if !ok {
		return
	}
	x, y := scene.ActorRenderPosition(actor, now)
	m.effects().AddCell(ctx, effects.Teleportation, 0, int(math.Round(x)), int(math.Round(y)), now, 0, 0)
}

func (m *WorldMode) cleanupDeadActors(ctx client.Context, now time.Time) {
	if len(m.actorDeaths) == 0 || ctx.World == nil {
		return
	}
	for id, removeAt := range m.actorDeaths {
		// Dead player characters remain in the world so resurrection skills can
		// target them. Their zero deadline is cleared by ZC_RESURRECTION, an
		// actor refresh, or a later non-death vanish packet.
		if removeAt.IsZero() {
			continue
		}
		if now.Before(removeAt) {
			continue
		}
		ctx.World.RemoveActor(id)
		m.removeActorEffectStateEffects(id)
		m.effects().RemoveLevel99AuraEffects(id)
		delete(m.actorDeaths, id)
		delete(m.actorAnims, id)
		delete(m.actorSoundFrames, id)
		delete(m.actorLife, id)
		delete(m.speechBubbles, id)
		delete(m.petAccessoryIDs, id)
		if m.pendingAttack.targetID == id {
			m.pendingAttack = attackIntent{}
		}
		if m.lockedAttackID == id {
			m.clearLockedAttack()
		}
		if m.attackFocusID == id {
			m.clearAttackFocus()
		}
		if m.scriptHighlight.id == id {
			m.clearScriptHighlight()
		}
		glog.Debugf("actor death removed id=%d", id)
	}
}

func (m *WorldMode) cleanupVanishedActors(ctx client.Context, now time.Time) {
	if len(m.actorVanishes) == 0 {
		return
	}
	for id, fade := range m.actorVanishes {
		if now.Before(fade.removeAt) {
			continue
		}
		m.removeActorNow(ctx, id)
		glog.Debugf("actor out-of-sight removed id=%d", id)
	}
}

func applyNPCSpriteChange(ctx client.Context, change network.NPCSpriteChange) {
	if ctx.World == nil || change.ID == 0 || scene.IsLocalActor(ctx, change.ID) || change.ID == ctx.World.Player.ID {
		return
	}
	if change.Job > math.MaxInt16 || res.HasPlayerJobToken(int(change.Job)) {
		return
	}
	actor, ok := ctx.World.Actors[change.ID]
	if !ok || scene.ActorRepresentsPlayer(actor) {
		return
	}
	actor.Job = int16(change.Job)
	actor.Appearance = true
	// Sprite/model caches are keyed by job, so the next draw selects the new
	// resources without changing views shared by other actors. Updating the
	// appearance leaves the actor's movement and other state intact.
	ctx.World.Actors[change.ID] = actor
}

func applyActorLookChange(ctx client.Context, look network.ActorLookChange) bool {
	if look.ID == 0 {
		return false
	}
	if scene.IsLocalActor(ctx, look.ID) {
		applyCharacterLookChange(ctx.Session, look)
		applyWorldActorLookChange(&ctx.World.Player, look)
		return true
	}
	actor, ok := ctx.World.Actors[look.ID]
	if !ok {
		actor = worldstate.Actor{ID: look.ID, Appearance: true}
	}
	applyWorldActorLookChange(&actor, look)
	upsertActor(ctx, actor)
	return false
}

func applyCharacterLookChange(sessionState *session.Session, look network.ActorLookChange) {
	update := func(character *session.Character) {
		switch look.Type {
		case 0:
			character.Job = int16(look.Value)
		case 1:
			character.Hair = int16(look.Value)
		case 2:
			weapon, shield := res.NormalizePlayerWeaponShield(int(look.Value&0xFFFF), int((look.Value>>16)&0xFFFF))
			character.Weapon = int16(weapon)
			character.Shield = int16(shield)
		case 3:
			character.HeadLow = int16(look.Value)
		case 4:
			character.HeadTop = int16(look.Value)
		case 5:
			character.HeadMid = int16(look.Value)
		case 6:
			character.HeadPal = int16(look.Value)
			if look.Value <= 255 {
				character.HairColor = uint8(look.Value)
			}
		case 7:
			character.BodyPal = int16(look.Value)
		case 8:
			character.Shield = int16(look.Value)
		}
	}
	update(&sessionState.Selected)
	for index := range sessionState.Characters {
		if sessionState.Characters[index].ID == sessionState.CharID || sessionState.Characters[index].ID == sessionState.Selected.ID {
			update(&sessionState.Characters[index])
		}
	}
}

func applyWorldActorLookChange(actor *worldstate.Actor, look network.ActorLookChange) {
	actor.Appearance = true
	switch look.Type {
	case 0:
		if actorHasMobObjectType(*actor) && res.HasPlayerJobToken(int(look.Value)) {
			glog.Debugf("ignored mob look-base player job id=%d old_job=%d value=%d", actor.ID, actor.Job, look.Value)
			return
		}
		actor.Job = int16(look.Value)
	case 1:
		actor.Head = int16(look.Value)
	case 2:
		weapon, shield := res.NormalizePlayerWeaponShield(int(look.Value&0xFFFF), int((look.Value>>16)&0xFFFF))
		actor.Weapon = int16(weapon)
		actor.Shield = int16(shield)
	case 3:
		actor.HeadLow = int16(look.Value)
	case 4:
		actor.HeadTop = int16(look.Value)
	case 5:
		actor.HeadMid = int16(look.Value)
	case 8:
		actor.Shield = int16(look.Value)
	}
}

func actorHasMobObjectType(actor worldstate.Actor) bool {
	if !actor.HasObjectType {
		return false
	}
	switch actor.ObjectType {
	case scene.ActorObjectTypeMob, scene.ActorObjectTypeNPCABR, scene.ActorObjectTypeNPCBionic:
		return true
	default:
		return false
	}
}

func actorIsAdmin(ctx client.Context, id uint32) bool {
	return ctx.Session != nil && ctx.Session.IsAdminID(id)
}

func localPlayerIsAdmin(ctx client.Context) bool {
	if ctx.Session == nil {
		return false
	}
	return ctx.Session.IsAdminID(ctx.Session.AccountID) || ctx.Session.IsAdminID(ctx.Session.CharID)
}

func applyMapAcceptEnter(ctx client.Context, enter network.MapAcceptEnter) {
	if ctx.Session == nil || ctx.World == nil {
		return
	}
	resetWorldForSelectedCharacterIfNeeded(ctx)
	ctx.Session.SyncServerTick(enter.ServerTick, time.Now())
	ctx.Session.PlayerX = enter.X
	ctx.Session.PlayerY = enter.Y
	ctx.Session.PlayerDir = enter.Dir
	ctx.Session.Playing = true
	ctx.World.SetPlayerPosition(enter.X, enter.Y, enter.Dir)
	ctx.World.Player.IsAdmin = localPlayerIsAdmin(ctx)
}

func resetWorldForSelectedCharacterIfNeeded(ctx client.Context) {
	if ctx.Session == nil || ctx.World == nil || ctx.Session.CharID == 0 {
		return
	}
	if ctx.Session.Playing && ctx.World.Player.ID == ctx.Session.CharID {
		return
	}
	ctx.World.Player = worldActorForSelectedCharacter(ctx)
	ctx.World.Actors = make(map[uint32]worldstate.Actor)
	ctx.World.Items = make(map[uint32]worldstate.FloorItem)
	ctx.World.Dir = 0
}

func worldActorForSelectedCharacter(ctx client.Context) worldstate.Actor {
	character := ctx.Session.SelectedCharacter()
	actor := worldstate.Actor{
		ID:            ctx.Session.CharID,
		Name:          character.Name,
		Job:           character.Job,
		Head:          character.Hair,
		Weapon:        character.Weapon,
		Shield:        character.Shield,
		HeadTop:       character.HeadTop,
		HeadMid:       character.HeadMid,
		HeadLow:       character.HeadLow,
		HeadPal:       character.HeadPal,
		BodyPal:       character.BodyPal,
		Sex:           ctx.Session.Sex,
		IsAdmin:       localPlayerIsAdmin(ctx),
		Appearance:    true,
		Speed:         defaultPlayerMoveSpeedMS,
		EffectState:   character.Option,
		HasState:      character.Option != 0,
		Level:         ctx.Session.Progress.BaseLevel,
		HasLevel:      ctx.Session.Progress.BaseLevel > 0,
		AttackRange:   ctx.Session.AttackRange,
		PartyName:     ctx.Session.Party.Name,
		GuildID:       ctx.Session.GuildID,
		EmblemVersion: ctx.Session.EmblemVersion,
		GuildName:     ctx.Session.GuildName,
		HasObjectType: true,
		ObjectType:    scene.ActorObjectTypePC,
	}
	if character.Option&actorEffectCartMask != 0 {
		applyActorCartStateFromEffect(&actor)
	}
	return actor
}

func applyActorNameAck(ctx client.Context, ack network.ActorNameAck) {
	name := scene.SanitizeActorName(ack.Name)
	if name == "" || ctx.World == nil {
		return
	}
	guildName := strings.TrimSpace(ack.GuildName)
	partyName := strings.TrimSpace(ack.PartyName)
	if scene.IsLocalActor(ctx, ack.ID) {
		ctx.World.Player.Name = name
		ctx.World.Player.PartyName = partyName
		if ack.HasGuildName || guildName != "" {
			ctx.World.Player.GuildName = guildName
			if guildName == "" {
				ctx.World.Player.GuildID = 0
				ctx.World.Player.EmblemVersion = 0
			}
		}
		if ctx.Session != nil {
			if ack.HasGuildName || guildName != "" {
				ctx.Session.GuildName = guildName
				if guildName == "" {
					ctx.Session.GuildID = 0
					ctx.Session.EmblemVersion = 0
					ctx.Session.Guild = session.Guild{}
					ctx.Session.PendingGuildName = ""
				}
			}
			if guildName != "" {
				ctx.Session.PendingGuildName = ""
			}
		}
		return
	}
	actor, ok := ctx.World.Actors[ack.ID]
	if !ok {
		return
	}
	actor.Name = name
	actor.PartyName = partyName
	actor.GuildName = guildName
	if ack.HasGuildName && guildName == "" {
		actor.GuildID = 0
		actor.EmblemVersion = 0
	}
	ctx.World.Actors[ack.ID] = actor
}

func (m *WorldMode) drawSceneActors(screen *render.Frame, ctx client.Context, projection scene.Projection) []scene.ActorDrawEntry {
	entries := m.collectSceneActorEntries(screen, ctx, projection)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Depth > entries[j].Depth
	})
	for _, entry := range entries {
		m.drawActorShadowEntry(screen, ctx, projection, entry)
	}
	for _, entry := range entries {
		m.drawSceneActorEntry(screen, ctx, projection, entry)
	}
	m.drawSceneActorFalcons(screen, ctx, projection, entries)
	return entries
}

func (m *WorldMode) drawSceneActorOverlays(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time, entries []scene.ActorDrawEntry) {
	m.drawSiegeGuildEmblems(screen, ctx, projection, now, entries)
	for _, entry := range entries {
		m.drawActorCastBar(screen, entry, now)
		m.drawActorLifeBar(screen, ctx, entry)
	}
	m.drawAttackFocusMarker(screen, ctx, now, entries)
	m.drawScriptHighlightMarker(screen, ctx, now, entries)
	entries = m.appendSimulatedVendingEntries(screen, ctx, entries)
	m.drawVendingBoardLabels(screen, ctx, entries)
	m.drawChatRoomBoardLabels(screen, ctx, entries)
	m.drawSpeechBubbles(screen, ctx, projection, entries, now)
	m.drawHoveredLocalPlayerNameLabel(screen, ctx, entries)
	m.drawHoveredActorNameLabel(screen, ctx, projection, now)
}

func (m *WorldMode) drawHoveredLocalPlayerNameLabel(screen *render.Frame, ctx client.Context, entries []scene.ActorDrawEntry) {
	if ctx.Input == nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsPlayer {
			continue
		}
		if !pointInActorPickBounds(float64(ctx.Input.MouseX), float64(ctx.Input.MouseY), entry.ScreenX, entry.ScreenY, entry.Scale*inputPickMultiplier(ctx)) {
			return
		}
		labelY := actorNameLabelY(entry.ScreenY, entry.Scale)
		if life, ok := m.actorLifeForDisplay(ctx, entry.Actor); ok {
			labelY = actorNameBelowLifeBarY(entry.ScreenY, entry.Scale, life)
		}
		drawActorNameLabelsAtY(screen, actorDisplayLabels(ctx, entry.Actor, true), m.actorGuildEmblem(ctx, entry.Actor, true), entry.ScreenX, labelY, actorNameLabelColor(entry.Actor, true))
		return
	}
}

func (m *WorldMode) collectSceneActorEntries(screen *render.Frame, ctx client.Context, projection scene.Projection) []scene.ActorDrawEntry {
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	now := time.Now()
	entries := make([]scene.ActorDrawEntry, 0, len(ctx.World.Actors)+1)
	player := ctx.World.Player
	player.ID = ctx.Session.CharID
	character := ctx.Session.SelectedCharacter()
	player.Job = character.Job
	player.Head = character.Hair
	player.Sex = ctx.Session.Sex
	player.IsAdmin = localPlayerIsAdmin(ctx)
	ctx.World.Player.IsAdmin = player.IsAdmin
	if !player.HasCartState {
		if player.HasState {
			player.EffectState = (player.EffectState &^ actorPersistentEffectOptionMask) | (character.Option & actorPersistentEffectOptionMask)
		} else {
			player.EffectState = character.Option
		}
		applyActorCartStateFromEffect(&player)
	}
	if character.Name != "" {
		player.Name = character.Name
	}
	if ctx.Session != nil {
		if player.GuildID == 0 {
			player.GuildID = localGuildID(ctx)
		}
		if player.EmblemVersion == 0 {
			player.EmblemVersion = ctx.Session.EmblemVersion
		}
	}
	player.Dir = ctx.World.Dir
	entries = appendActorDrawEntry(entries, ctx.World, projection, player, true, now, width, height)
	for _, actor := range ctx.World.Actors {
		if actor.ID == ctx.Session.AccountID || actor.ID == ctx.Session.CharID {
			continue
		}
		entries = appendActorDrawEntry(entries, ctx.World, projection, actor, false, now, width, height)
	}
	visible := entries[:0]
	for _, entry := range entries {
		entry.Actor.Dir = m.skillEffectDirection(ctx, entry.Actor.ID, entry.Actor.Dir, now)
		entry.Stealth = actorStealthView(ctx, entry.Actor, entry.IsPlayer)
		// The local gauges remain visible even when the body is absent.
		if entry.IsPlayer || entry.Stealth != scene.StealthHidden {
			visible = append(visible, entry)
		}
	}
	return visible
}

func (m *WorldMode) drawSceneActorEntry(screen *render.Frame, ctx client.Context, projection scene.Projection, entry scene.ActorDrawEntry) {
	if entry.Stealth == scene.StealthHidden || entry.Stealth == scene.StealthShadow {
		return
	}
	cameraYaw := projection.CameraYaw
	alpha := m.actorVisualAlpha(entry.Actor.ID, time.Now())
	if entry.IsPlayer {
		if !cartDrawAfterActor(entry.Actor, cameraYaw) {
			m.drawActorCart3D(screen, ctx, projection, entry, cameraYaw, entry.Shadow, alpha)
		}
		if m.drawPlayerSprite3D(ctx, screen, projection, entry, entry.Actor.Dir, cameraYaw, entry.Shadow, alpha) {
			if cartDrawAfterActor(entry.Actor, cameraYaw) {
				m.drawActorCart3D(screen, ctx, projection, entry, cameraYaw, entry.Shadow, alpha)
			}
			return
		}
		return
	}
	if visual := specialNPCVisualForActor(entry.Actor); visual != specialNPCVisualNone {
		if m.drawSpecialNPCVisual(screen, ctx, projection, entry, visual, time.Now()) {
			return
		}
	}
	if isWarpActor(entry.Actor) {
		return
	}
	if !cartDrawAfterActor(entry.Actor, cameraYaw) {
		m.drawActorCart3D(screen, ctx, projection, entry, cameraYaw, entry.Shadow, alpha)
	}
	if m.drawActorSprite3D(screen, ctx, projection, entry, cameraYaw, entry.Shadow) {
		if cartDrawAfterActor(entry.Actor, cameraYaw) {
			m.drawActorCart3D(screen, ctx, projection, entry, cameraYaw, entry.Shadow, alpha)
		}
		return
	}
}

func (m *WorldMode) drawActorShadowEntry(screen *render.Frame, ctx client.Context, projection scene.Projection, entry scene.ActorDrawEntry) {
	if !entry.CastShadow || m.shadowView == nil || m.shadowViewMiss {
		return
	}
	if entry.Stealth == scene.StealthHidden || entry.Stealth == scene.StealthSilhouette {
		return
	}
	if !entry.IsPlayer && m.nonPCActorHasGR2Model(ctx, entry.Actor) {
		return
	}
	now := time.Now()
	scale := m.actorShadowRenderScale(ctx, entry, now)
	if scale == 0 {
		return
	}
	if !m.actorShadowSuppressed(entry.Actor, now) {
		sprites.DrawFixedShadow3D(screen, projection, m.shadowView, entry.WorldX, entry.WorldY, entry.WorldZ+scene.ActorShadowTerrainLift, scale, m.actorVisualAlpha(entry.Actor.ID, now), entry.Shadow)
	}
	m.drawActorCartShadow3D(screen, ctx, projection, entry, projection.CameraYaw, now)
}

func (m *WorldMode) actorShadowRenderScale(ctx client.Context, entry scene.ActorDrawEntry, now time.Time) float64 {
	baseScale := m.actorRenderScale(entry.Actor.ID, entry.Scale, now)
	if entry.IsPlayer {
		baseScale = m.playerRenderScale(ctx, entry.Actor, entry.Scale, now)
	} else if scene.ActorRepresentsPlayer(entry.Actor) {
		baseScale = m.playerBodyRenderScale(entry.Actor.ID, entry.Actor.Job, entry.Scale, now)
	}
	scale := baseScale * entry.ShadowScale
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return 0
	}
	return scale
}

func appendActorDrawEntry(entries []scene.ActorDrawEntry, world *worldstate.World, projection scene.Projection, actor worldstate.Actor, isPlayer bool, now time.Time, screenWidth, screenHeight int) []scene.ActorDrawEntry {
	// Hidden warp NPCs anchor scripted effects, such as the airship explosions.
	// Keep them in the world for those effects, but give them no actor visuals.
	if actor.Job == scene.ActorJobHiddenWarpNPC {
		return entries
	}
	actorX, actorY := scene.ActorRenderPosition(actor, now)
	actor.Dir = scene.ActorRenderDirection(actor, now)
	actor = actorWithVisualJob(actor)
	terrainZ := scene.TerrainHeightAt(world, actorX, actorY)
	worldX, worldY := actorWorldAnchor(actor, actorX, actorY)
	point := projection.Project(worldX, worldY, terrainZ)
	scale := scene.ActorBillboardScreenScale(projection, worldX, worldY, terrainZ)
	if isPlayer || scene.ActorRepresentsPlayer(actor) {
		scale *= playerBodyScaleForJob(actor.Job)
	}
	if actorAnchorOutsideViewport(float64(point.X), float64(point.Y), screenWidth, screenHeight, scale) {
		return entries
	}
	depth := actorBillboardSortDepth(projection, worldX, worldY, terrainZ)
	shadowDepth := projection.Depth(worldX, worldY, terrainZ+scene.ActorShadowTerrainLift)
	return append(entries, scene.ActorDrawEntry{
		Actor:       actor,
		ScreenX:     float64(point.X),
		ScreenY:     float64(point.Y),
		WorldX:      worldX,
		WorldY:      worldY,
		WorldZ:      terrainZ,
		Scale:       scale,
		Shadow:      actorShadowFactor(world, actorX, actorY),
		CastShadow:  actorCastsShadow(actor),
		ShadowScale: actorShadowSize(actor),
		ShadowDepth: shadowDepth,
		Depth:       depth,
		IsPlayer:    isPlayer,
	})
}

func playerBodyScaleForJob(job int16) float64 {
	if db.IsBabyJob(int(job)) {
		return scene.BabyPlayerBodyScale
	}
	return 1
}

func actorAnchorOutsideViewport(anchorX, anchorY float64, screenWidth, screenHeight int, scale float64) bool {
	left, right, top, bottom := actorViewportCullMargins(scale)
	return anchorX < -left || anchorX > float64(screenWidth)+right || anchorY < -top || anchorY > float64(screenHeight)+bottom
}

func actorViewportCullMargins(scale float64) (left, right, top, bottom float64) {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	side := math.Max(128, float64(scene.HumanoidBillboardWidth)*scale)
	// Most entity pixels are above the feet/anchor. The lower screen edge needs
	// a larger margin so tall sprites do not disappear while their body is still
	// visible and only their feet are outside the viewport.
	topMargin := math.Max(96, float64(scene.HumanoidBillboardHeight-scene.HumanoidBillboardAnchorY)*scale*2)
	bottomMargin := math.Max(192, float64(scene.HumanoidBillboardAnchorY)*scale*1.6)
	return side, side, topMargin, bottomMargin
}

func actorCastsShadow(actor worldstate.Actor) bool {
	if isWarpActor(actor) || actorJobHasSpecialNoShadow(int(actor.Job)) {
		return false
	}
	return actorShadowSize(actor) > 0
}

func (m *WorldMode) actorShadowSuppressed(actor worldstate.Actor, now time.Time) bool {
	if actor.Sitting {
		return true
	}
	if anim, ok := m.actorAnimation(actor.ID, now); ok {
		// Action numbers depend on the actor: monster attack is player sit,
		// and monster death is player combat-ready.
		if anim.actionFamily == deathActionFamilyForActor(actor) {
			return true
		}
		return anim.actionFamily == scene.SpriteActionSit && (res.HasPlayerJobToken(int(actor.Job)) || companion.IsMercenary(actor))
	}
	return false
}

func actorShadowSize(actor worldstate.Actor) float64 {
	if size, ok := db.MonsterShadowSize[int(actor.Job)]; ok {
		return size
	}
	return 1
}

func actorShadowFactor(world *worldstate.World, x, y float64) float64 {
	if world == nil || world.GND == nil {
		return 1
	}
	if gndHeight, ok := maps.WorldGNDHeightAt(world, x, y); ok && scene.TerrainHeightAt(world, x, y)-gndHeight > 0.5 {
		return 1
	}
	shadowX, shadowY := maps.GNDShadowMapPoint(x, y)
	total := 0
	for dy := -3; dy < 3; dy++ {
		for dx := -3; dx < 3; dx++ {
			total += int(maps.GNDShadowMapAlpha(world.GND, shadowX+dx, shadowY+dy))
		}
	}
	return scene.ClampUnit(float64(total) / (6 * 6 * 255))
}

func actorBillboardSortDepth(projection scene.Projection, x, y, z float64) float64 {
	footDepth := projection.Depth(x, y, z)
	topDepth := projection.Depth(x, y, z+scene.ActorBillboardWorldHeightUnit)
	if topDepth <= 0 || !scene.IsFinite(topDepth) {
		return footDepth
	}
	return math.Min(footDepth, topDepth)
}

func actorDisplayLabels(ctx client.Context, actor worldstate.Actor, isPlayer bool) []string {
	name := actorDisplayPrimaryName(ctx, actor, isPlayer)
	if name == "" {
		return nil
	}
	if guildName := actorGuildDisplayName(ctx, actor, isPlayer); guildName != "" {
		return []string{name, guildName}
	}
	return []string{name}
}

func actorDisplayPrimaryName(ctx client.Context, actor worldstate.Actor, isPlayer bool) string {
	if actor.Job == scene.ActorJobHiddenWarpNPC {
		return ""
	}
	if isPlayer {
		if name := scene.SanitizeActorName(selectedCharacterName(ctx.Session)); name != "" {
			return actorDisplayNameWithParty(ctx, actor, name, true)
		}
		return actorDisplayNameWithParty(ctx, actor, scene.SanitizeActorName(actor.Name), true)
	}
	if isWarpActor(actor) {
		return ""
	}
	if name := scene.SanitizeActorName(actor.Name); name != "" {
		return actorDisplayNameWithParty(ctx, actor, name, false)
	}
	if res.HasPlayerJobToken(int(actor.Job)) || ctx.Resources == nil {
		return ""
	}
	if resourceName, ok := ctx.Resources.NonPCResourceName(int(actor.Job)); ok {
		return displayNameFromResource(resourceName)
	}
	if name, ok := db.MonsterDisplayName[int(actor.Job)]; ok {
		return name
	}
	return ""
}

func actorGuildDisplayName(ctx client.Context, actor worldstate.Actor, isPlayer bool) string {
	if !actorCanDisplayGuildName(actor, isPlayer) {
		return ""
	}
	if guildName := strings.TrimSpace(actor.GuildName); guildName != "" {
		return guildName
	}
	if isPlayer && ctx.Session != nil {
		return strings.TrimSpace(ctx.Session.GuildName)
	}
	return ""
}

func actorCanDisplayGuildName(actor worldstate.Actor, isPlayer bool) bool {
	if strings.TrimSpace(actor.GuildName) == "" && !isPlayer {
		return false
	}
	if isPlayer {
		return true
	}
	if actor.HasObjectType {
		return actor.ObjectType == scene.ActorObjectTypePC
	}
	return res.HasPlayerJobToken(int(actor.Job))
}

func actorDisplayNameWithParty(ctx client.Context, actor worldstate.Actor, name string, isPlayer bool) string {
	name = strings.TrimSpace(name)
	partyName := actorPartyDisplayName(ctx, actor, isPlayer)
	if name == "" || partyName == "" {
		return name
	}
	return name + " (" + partyName + ")"
}

func actorPartyDisplayName(ctx client.Context, actor worldstate.Actor, isPlayer bool) string {
	if name := strings.TrimSpace(actor.PartyName); name != "" {
		return name
	}
	if isPlayer && ctx.Session != nil {
		return strings.TrimSpace(ctx.Session.Party.Name)
	}
	return ""
}

func actorIsPartyMember(s *session.Session, actor worldstate.Actor, actorName string) bool {
	if s == nil || actor.ID == 0 {
		return false
	}
	for _, member := range s.Party.Members {
		if member.AccountID == actor.ID {
			return true
		}
		if actorName != "" && strings.EqualFold(scene.SanitizeActorName(member.Name), actorName) {
			return true
		}
	}
	return false
}

func (m *WorldMode) drawHoveredActorNameLabel(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time) {
	if ctx.Input == nil || ctx.World == nil {
		return
	}
	if _, ok := clickedGroundItem(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
		return
	}
	actor, ok := hoveredCursorActor(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths)
	if !ok || isWarpActor(actor) {
		return
	}
	labels := m.hoveredActorDisplayLabels(ctx, actor, now)
	if len(labels) == 0 {
		return
	}
	actorX, actorY := scene.ActorRenderPosition(actor, now)
	terrainZ := scene.TerrainHeightAt(ctx.World, actorX, actorY)
	point := projection.Project(scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
	scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
	isPlayer := scene.IsLocalActor(ctx, actor.ID)
	labelY := actorNameLabelY(float64(point.Y), scale)
	if life, ok := m.actorLifeForDisplay(ctx, actor); ok {
		labelY = actorNameBelowLifeBarY(float64(point.Y), scale, life)
	}
	drawActorNameLabelsAtY(screen, labels, m.actorGuildEmblem(ctx, actor, isPlayer), float64(point.X), labelY, actorNameLabelColor(actor, isPlayer))
}

func (m *WorldMode) hoveredActorDisplayLabels(ctx client.Context, actor worldstate.Actor, now time.Time) []string {
	if actor.Job == scene.ActorJobHiddenWarpNPC {
		return nil
	}
	if scene.IsLocalActor(ctx, actor.ID) {
		return actorDisplayLabels(ctx, actor, true)
	}
	if name := scene.SanitizeActorName(actor.Name); name != "" {
		return actorDisplayLabels(ctx, actor, false)
	}
	if shouldUseServerNameForHoverActor(actor) {
		m.requestActorName(ctx, actor.ID, now)
		if name := actorResourceDisplayName(ctx, actor); name != "" {
			return []string{name}
		}
		if res.HasPlayerJobToken(int(actor.Job)) {
			return []string{"Player"}
		}
		if scene.IsMonsterLikeHoverActor(actor) {
			return []string{"Monster"}
		}
		return []string{"NPC"}
	}
	if name := actorResourceDisplayName(ctx, actor); name != "" {
		return []string{name}
	}
	return []string{"Entity"}
}

func actorResourceDisplayName(ctx client.Context, actor worldstate.Actor) string {
	if ctx.Resources == nil {
		return ""
	}
	if resourceName, ok := ctx.Resources.NonPCResourceName(int(actor.Job)); ok {
		return displayNameFromResource(resourceName)
	}
	return ""
}

func (m *WorldMode) requestActorName(ctx client.Context, id uint32, now time.Time) {
	if id == 0 || ctx.Network == nil || scene.IsLocalActor(ctx, id) {
		return
	}
	if m.actorNameReqAt == nil {
		m.actorNameReqAt = make(map[uint32]time.Time)
	}
	if previous, ok := m.actorNameReqAt[id]; ok && now.Sub(previous) < actorNameRequestCooldown {
		return
	}
	if err := ctx.Network.SendNameRequest(id); err != nil {
		glog.Warnf("send name request failed id=%d: %v", id, err)
		return
	}
	m.actorNameReqAt[id] = now
}

func shouldUseServerNameForHoverActor(actor worldstate.Actor) bool {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return true
	}
	if scene.IsMonsterLikeHoverActor(actor) {
		return true
	}
	if !actor.HasObjectType {
		return false
	}
	switch actor.ObjectType {
	case scene.ActorObjectTypeNPC, scene.ActorObjectTypeHomunculus, scene.ActorObjectTypeMercenary:
		return true
	default:
		return false
	}
}

func selectedCharacterName(s *session.Session) string {
	if s == nil {
		return ""
	}
	return s.SelectedCharacter().Name
}

func displayNameFromResource(name string) string {
	name = strings.TrimSpace(strings.TrimSuffix(name, ".spr"))
	name = strings.TrimSuffix(name, ".act")
	name = strings.TrimSuffix(name, ".gr2")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ToLower(strings.TrimSpace(name))
	fields := strings.Fields(name)
	for i, field := range fields {
		fields[i] = titleASCIIWord(field)
	}
	return strings.Join(fields, " ")
}

func isWarpActor(actor worldstate.Actor) bool {
	return isWarpActorJob(int(actor.Job))
}

func isWarpActorJob(job int) bool {
	return job == scene.ActorJobWarpPortal || job == scene.ActorJobWarpPortalActive || job == scene.ActorJobWarpPortalWaiting
}

func titleASCIIWord(word string) string {
	if word == "" {
		return ""
	}
	if word[0] < 'a' || word[0] > 'z' {
		return word
	}
	return strings.ToUpper(word[:1]) + word[1:]
}

func actorNameLabelColor(actor worldstate.Actor, isPlayer bool) color.RGBA {
	if actor.IsAdmin {
		return color.RGBA{R: 255, G: 255, B: 0, A: 255}
	}
	if isPlayer {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	switch {
	case actor.HasObjectType && actor.ObjectType == scene.ActorObjectTypeNPC:
		return color.RGBA{R: 148, G: 189, B: 247, A: 255}
	case scene.IsMonsterLikeHoverActor(actor):
		return color.RGBA{R: 255, G: 198, B: 198, A: 255}
	default:
		return color.RGBA{R: 248, G: 248, B: 248, A: 255}
	}
}

func drawActorNameLabelsAtY(screen *render.Frame, labels []string, emblem *render.Image, centerX, labelY float64, foreground color.RGBA) {
	if len(labels) == 0 {
		return
	}
	labels = actorVisibleNameLabels(labels)
	if len(labels) == 0 {
		return
	}
	render.DrawActorUILabels(screen, labels, emblem, centerX, labelY, foreground, color.RGBA{A: 196})
}

func actorVisibleNameLabels(labels []string) []string {
	out := make([]string, 0, len(labels))
	for i, label := range labels {
		if i == 0 {
			label = scene.SanitizeActorName(label)
		} else {
			label = strings.TrimSpace(label)
		}
		if label != "" {
			out = append(out, label)
		}
	}
	return out
}

func actorSpriteTopY(baseY, scale float64) float64 {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	return baseY - float64(scene.HumanoidBillboardAnchorY)*scale
}

func actorNameLabelY(baseY, scale float64) float64 {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	return baseY + 13*scale
}

func actorLifeBarY(baseY, scale float64) float64 {
	return actorNameLabelY(baseY, scale) + 14
}

func actorCastBarY(baseY, scale float64) float64 {
	return actorSpriteTopY(baseY, scale) + 4
}

func (m *WorldMode) drawActorSprite3D(screen *render.Frame, ctx client.Context, projection scene.Projection, entry scene.ActorDrawEntry, cameraYaw float64, shadow float64) bool {
	actor := entry.Actor
	if companion.IsMercenary(actor) {
		return m.drawMercenarySprite3D(screen, ctx, projection, entry, cameraYaw, shadow)
	}
	if !res.HasPlayerJobToken(int(actor.Job)) {
		if m.nonPCActorHasGR2Model(ctx, actor) {
			return m.drawNonPCGR2Model3D(screen, ctx, entry, shadow)
		}
		return m.drawNonPCSprite3D(screen, ctx, projection, entry, cameraYaw, shadow)
	}
	weapon, shield := res.NormalizePlayerWeaponShield(int(actor.Weapon), int(actor.Shield))
	key := sprites.Appearance{
		Job:         int(actor.Job),
		Head:        int(actor.Head),
		Sex:         actor.Sex,
		Admin:       actor.IsAdmin,
		BodyPalette: int(actor.BodyPal),
		HeadPalette: int(actor.HeadPal),
		Weapon:      weapon,
		Shield:      shield,
		HeadTop:     int(actor.HeadTop),
		HeadMid:     int(actor.HeadMid),
		HeadLow:     int(actor.HeadLow),
	}
	if _, ok := m.actorViewMiss[key]; ok {
		return false
	}
		view, ok := m.actorViews[key]
	if !ok {
		loaded, status := sprites.LoadHumanoid(ctx.Resources, key, "actor")
		if loaded == nil {
			m.actorViewMiss[key] = struct{}{}
			glog.Warnf("actor sprite unavailable id=%d job=%d head=%d sex=%d: %s", actor.ID, key.Job, key.Head, key.Sex, status)
			return false
		}
		m.actorViews[key] = loaded
		view = loaded
		glog.Debugf("actor sprite resources id=%d job=%d head=%d sex=%d %s", actor.ID, key.Job, key.Head, key.Sex, status)
	}
	now := time.Now()
	state := sprites.State{
		ActionFamily: scene.SpriteActionIdle,
		Direction:    actor.Dir,
		HeadDir:      actor.HeadDir,
		HeadTurn:     true,
		CameraYaw:    cameraYaw,
		Moving:       scene.ActorIsMovingAt(actor, now),
		MoveSpeedMS:  actor.Speed,
	}
	if state.Moving {
		state.ActionFamily = scene.SpriteActionWalk
		state.Loop = true
		state.WalkDistance = actorRenderWalkDistance(actor, now)
	} else if actor.Sitting {
		state.ActionFamily = scene.SpriteActionSit
	}
	if anim, ok := m.actorAnimation(actor.ID, now); ok {
		applyActorAnimationToSpriteState(&state, anim, true)
	}
	if !isDeathActionFamily(state.ActionFamily) {
		applyActorBodyState(actor, &state)
	}
	state.SeparateEquipment = m.effects().LordKnightSeparateEquipment(ctx, actor, now)
	billboard, ok := sprites.HumanoidBillboard(view, state, now)
	if !ok {
		return false
	}
	m.drawActorSpriteWithEffects3D(ctx, screen, projection, entry, billboard, m.playerBodyRenderScale(actor.ID, actor.Job, entry.Scale, now), m.actorVisualAlpha(actor.ID, now), shadow, entry.Stealth.Tint(m.actorRenderTint(actor, now)), entry.Stealth.Blend(m.actorRenderBlend(actor.ID, now)), now)
	return true
}

func (m *WorldMode) drawMercenarySprite3D(screen *render.Frame, ctx client.Context, projection scene.Projection, entry scene.ActorDrawEntry, cameraYaw float64, shadow float64) bool {
	actor := entry.Actor
	view := m.mercenaryHumanoidSpriteView(ctx, actor)
	if view == nil {
		return false
	}
	now := time.Now()
	state := sprites.State{
		ActionFamily: scene.SpriteActionIdle,
		Direction:    actor.Dir,
		HeadDir:      actor.HeadDir,
		HeadTurn:     true,
		CameraYaw:    cameraYaw,
		Moving:       scene.ActorIsMovingAt(actor, now),
		MoveSpeedMS:  actor.Speed,
	}
	if state.Moving {
		state.ActionFamily = scene.SpriteActionWalk
		state.Loop = true
		state.WalkDistance = actorRenderWalkDistance(actor, now)
	} else if actor.Sitting {
		state.ActionFamily = scene.SpriteActionSit
	}
	if anim, ok := m.actorAnimation(actor.ID, now); ok {
		applyActorAnimationToSpriteState(&state, anim, true)
	}
	if !isDeathActionFamily(state.ActionFamily) {
		applyActorBodyState(actor, &state)
	}
	state.SeparateEquipment = m.effects().LordKnightSeparateEquipment(ctx, actor, now)
	billboard, ok := sprites.HumanoidBillboard(view, state, now)
	if !ok {
		return false
	}
	m.drawActorSpriteWithEffects3D(ctx, screen, projection, entry, billboard, m.actorRenderScale(actor.ID, entry.Scale, now), m.actorVisualAlpha(actor.ID, now), shadow, entry.Stealth.Tint(m.actorRenderTint(actor, now)), entry.Stealth.Blend(m.actorRenderBlend(actor.ID, now)), now)
	return true
}

func (m *WorldMode) mercenaryHumanoidSpriteView(ctx client.Context, actor worldstate.Actor) *sprites.Humanoid {
	if ctx.Config.Headless {
		return nil
	}
	key := sprites.MercenaryAppearanceForActor(actor)
	if _, ok := m.mercenaryViewMiss[key]; ok {
		return nil
	}
	if m.mercenaryViews == nil {
		m.mercenaryViews = make(map[sprites.Appearance]*sprites.Humanoid)
	}
	if view, ok := m.mercenaryViews[key]; ok {
		return view
	}
	if ctx.Resources == nil {
		return nil
	}
	loaded, status := sprites.LoadMercenary(ctx.Resources, key, "mercenary")
	if loaded == nil {
		if m.mercenaryViewMiss == nil {
			m.mercenaryViewMiss = make(map[sprites.Appearance]struct{})
		}
		m.mercenaryViewMiss[key] = struct{}{}
		glog.Warnf("mercenary sprite unavailable id=%d job=%d head=%d sex=%d: %s", actor.ID, key.Job, key.Head, key.Sex, status)
		return nil
	}
	m.mercenaryViews[key] = loaded
	glog.Debugf("mercenary sprite resources id=%d job=%d head=%d sex=%d %s", actor.ID, key.Job, key.Head, key.Sex, status)
	return loaded
}

func (m *WorldMode) drawNonPCSprite3D(screen *render.Frame, ctx client.Context, projection scene.Projection, entry scene.ActorDrawEntry, cameraYaw float64, shadow float64) bool {
	actor := entry.Actor
	view := m.nonPCSpriteView(ctx, actor)
	if view == nil {
		return false
	}
	now := time.Now()
	state := m.nonPCSpriteState(actor, now)
	state.CameraYaw = cameraYaw
	billboard, ok := sprites.BillboardForState(view, state, now)
	if !ok {
		return false
	}
	m.drawActorSpriteWithEffects3D(ctx, screen, projection, entry, billboard, m.actorRenderScale(actor.ID, entry.Scale, now), m.actorVisualAlpha(actor.ID, now), shadow, entry.Stealth.Tint(m.actorRenderTint(actor, now)), entry.Stealth.Blend(m.actorRenderBlend(actor.ID, now)), now)
	return true
}

func (m *WorldMode) nonPCSpriteState(actor worldstate.Actor, now time.Time) sprites.State {
	state := sprites.State{
		ActionFamily: scene.SpriteActionIdle,
		Direction:    actor.Dir,
		Moving:       scene.ActorIsMovingAt(actor, now),
		LoopIdle:     true,
		MoveSpeedMS:  actor.Speed,
	}
	if state.Moving {
		state.ActionFamily = scene.SpriteActionWalk
		state.Loop = true
		state.WalkDistance = actorRenderWalkDistance(actor, now)
	}
	if anim, ok := m.actorAnimation(actor.ID, now); ok {
		if applyActorAnimationToSpriteState(&state, anim, false) {
			state.LoopIdle = false
		}
	}
	if !isDeathActionFamily(state.ActionFamily) {
		applyActorBodyState(actor, &state)
	}
	return state
}

func isDeathActionFamily(actionFamily int) bool {
	return actionFamily == sprites.ActionPCDeath || actionFamily == sprites.ActionNonPCDeath
}

func (m *WorldMode) nonPCSpriteView(ctx client.Context, actor worldstate.Actor) *scene.SpriteView {
	if ctx.Config.Headless {
		return nil
	}
	job := int(actor.Job)
	if _, ok := m.nonPCViewMiss[job]; ok {
		return nil
	}
	if m.nonPCViews == nil {
		m.nonPCViews = make(map[int]*scene.SpriteView)
	}
		view, ok := m.nonPCViews[job]
	if ok {
		return m.petAccessorySpriteView(ctx, actor, view)
	}
	if ctx.Resources == nil {
		return nil
	}
	loaded, status := sprites.LoadNonPC(ctx.Resources, job, "nonpc")
	if loaded == nil {
		if m.nonPCViewMiss == nil {
			m.nonPCViewMiss = make(map[int]struct{})
		}
		m.nonPCViewMiss[job] = struct{}{}
		glog.Warnf("nonpc sprite unavailable id=%d job=%d: %s", actor.ID, job, status)
		return nil
	}
	m.nonPCViews[job] = loaded
	m.prefetchSpriteSounds(ctx.Resources, loaded.Act)
	glog.Debugf("nonpc sprite resources id=%d job=%d %s", actor.ID, job, status)
	return m.petAccessorySpriteView(ctx, actor, loaded)
}

func (m *WorldMode) petAccessorySpriteView(ctx client.Context, actor worldstate.Actor, base *scene.SpriteView) *scene.SpriteView {
	accessoryID := m.petAccessoryIDs[actor.ID]
	if accessoryID == 0 {
		return base
	}
	key := petAccessorySpriteKey{job: int(actor.Job), accessoryID: accessoryID}
	if view, ok := m.petAccessoryViews[key]; ok {
		return view
	}
	if _, miss := m.petAccessoryMiss[key]; miss {
		return base
	}
	view, status := sprites.LoadPetAccessory(ctx.Resources, base, accessoryID, "pet accessory")
	if view == nil {
		if m.petAccessoryMiss == nil {
			m.petAccessoryMiss = make(map[petAccessorySpriteKey]struct{})
		}
		m.petAccessoryMiss[key] = struct{}{}
		glog.Warnf("pet accessory sprite unavailable id=%d job=%d accessory=%d: %s", actor.ID, actor.Job, accessoryID, status)
		return base
	}
	if m.petAccessoryViews == nil {
		m.petAccessoryViews = make(map[petAccessorySpriteKey]*scene.SpriteView)
	}
	m.petAccessoryViews[key] = view
	glog.Debugf("pet accessory sprite resources id=%d job=%d accessory=%d %s", actor.ID, actor.Job, accessoryID, status)
	return view
}

type petAccessorySpriteKey struct {
	job         int
	accessoryID uint32
}

func (m *WorldMode) drawPlayerSprite3D(ctx client.Context, screen *render.Frame, projection scene.Projection, entry scene.ActorDrawEntry, direction int, cameraYaw float64, shadow float64, alpha float64) bool {
	now := time.Now()
	actor := entry.Actor
	moving := scene.ActorIsMovingAt(actor, now)
	state := sprites.State{
		SeparateEquipment: m.effects().LordKnightSeparateEquipment(ctx, actor, now),
		ActionFamily:      scene.SpriteActionIdle,
		Direction:         direction,
		HeadDir:           actor.HeadDir,
		HeadTurn:          true,
		CameraYaw:         cameraYaw,
		Moving:            moving,
		MoveSpeedMS:       actor.Speed,
	}
	if moving {
		state.ActionFamily = scene.SpriteActionWalk
		state.Loop = true
		state.WalkDistance = actorRenderWalkDistance(actor, now)
	} else if actor.Sitting {
		state.ActionFamily = scene.SpriteActionSit
	}
	if ctx.Session != nil {
		if anim, ok := m.actorAnimation(ctx.Session.CharID, now); ok {
			applyActorAnimationToSpriteState(&state, anim, true)
		} else if anim, ok := m.actorAnimation(ctx.Session.AccountID, now); ok {
			applyActorAnimationToSpriteState(&state, anim, true)
		}
	}
	if !isDeathActionFamily(state.ActionFamily) {
		applyActorBodyState(actor, &state)
	}
	billboard, ok := sprites.HumanoidBillboard(m.playerView, state, now)
	if !ok {
		return false
	}
	m.drawActorSpriteWithEffects3D(ctx, screen, projection, entry, billboard, m.playerRenderScale(ctx, actor, entry.Scale, now), alpha, shadow, entry.Stealth.Tint(m.playerRenderTint(ctx, actor, now)), entry.Stealth.Blend(m.playerRenderBlend(ctx, actor, now)), now)
	return true
}

func applyActorAnimationToSpriteState(state *sprites.State, anim actorAnimation, playerLike bool) bool {
	if state == nil {
		return false
	}
	if state.Moving && !actorAnimationOverridesWalk(anim, playerLike) {
		return false
	}
	state.ActionFamily = anim.actionFamily
	state.Started = anim.started
	state.Loop = anim.loop
	state.PingPong = anim.pingPong
	state.Play = anim.play
	state.HasPlay = anim.hasPlay
	state.Length = anim.length
	state.HasLength = anim.hasLength
	state.FrameOffset = anim.frameOffset
	state.HasFrameOffset = anim.hasFrameOffset
	state.Moving = false
	state.FixedMotion = anim.fixedMotion
	state.HasFixedMotion = anim.hasFixedMotion
	state.Speed = anim.speed
	state.HasSpeed = anim.hasSpeed
	return true
}

func actorAnimationOverridesWalk(anim actorAnimation, playerLike bool) bool {
	if playerLike {
		return anim.actionFamily == sprites.ActionPCDeath
	}
	return anim.actionFamily == sprites.ActionNonPCDeath
}

func (m *WorldMode) drawActorSpriteBillboardTintAlpha3D(screen *render.Frame, projection scene.Projection, billboard *scene.Billboard, worldX, worldY, worldZ, scale float64, alpha float64, shadow float64, tintColor color.RGBA, blend render.Blend) {
	m.effects().DrawActorSpriteBillboard(screen, projection, billboard, worldX, worldY, worldZ, scale, alpha, shadow, tintColor, blend)
}

func (m *WorldMode) drawSpriteBillboardAlpha3D(screen *render.Frame, projection scene.Projection, billboard *scene.Billboard, worldX, worldY, worldZ, scale float64, alpha float64, shadow float64) {
	options := scene.SpriteBillboardTriangleDrawOptions()
	options.DepthWrite = true
	options.Filter = render.FilterSprite
	scene.DrawSpriteBillboardTintAlpha3DWithOptions(screen, projection, billboard, m.spriteTexture(billboard.Image), worldX, worldY, worldZ, scale, alpha, shadow, color.RGBA{R: 255, G: 255, B: 255, A: 255}, options)
}
