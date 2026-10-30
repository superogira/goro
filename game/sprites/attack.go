package sprites

import (
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/world"
)

func AttackActionFamily(manager *res.Manager, actor world.Actor) int {
	if companion.IsMercenary(actor) {
		return mercenaryAttackActionFamily(int(actor.Job), actor.Sex, int(actor.Weapon))
	}
	job := scene.ActorVisualJob(actor)
	if res.HasPlayerJobToken(job) {
		weapon, _ := res.NormalizePlayerWeaponShield(int(actor.Weapon), int(actor.Shield))
		viewID := res.PlayerWeaponViewID(manager, weapon)
		if db.PlayerWeaponType(viewID) > 0 {
			weapon = viewID
		}
		switch db.PlayerWeaponAction(job, actor.Sex, weapon) {
		case db.PlayerWeaponActionAttack2:
			return ActionPCAttack2
		case db.PlayerWeaponActionAttack3:
			return ActionPCAttack3
		default:
			return ActionPCAttack1
		}
	}
	return ActionNonPCAttack
}

func mercenaryAttackActionFamily(job int, sex byte, weapon int) int {
	weapon = companion.MercenaryWeaponForAppearance(job, weapon)
	if weapon <= 0 {
		return ActionPCAttack1
	}
	switch db.PlayerWeaponAction(companion.MercenaryWeaponBaseJob(job), companion.MercenarySexForJob(job, sex), weapon) {
	case db.PlayerWeaponActionAttack2:
		return ActionPCAttack2
	case db.PlayerWeaponActionAttack3:
		return ActionPCAttack3
	default:
		return ActionPCAttack1
	}
}
