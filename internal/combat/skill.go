package combat

import (
	"math"
)

func (r *CombatRoom) CastSkill(caster *PlayerState, skillID int, targetID int64, tx, ty int) {
	cfg, ok := SkillTable[skillID]
	if !ok || caster.HP <= 0 {
		return
	}
	if cd := caster.Cooldowns[skillID]; cd > 0 {
		return
	}
	dx, dy := tx-caster.GridX, ty-caster.GridY
	if dx*dx+dy*dy > cfg.Range*cfg.Range {
		return
	}
	for _, eff := range cfg.Effects {
		r.execEffect(caster, eff, targetID, tx, ty)
	}
	caster.Cooldowns[skillID] = cfg.Cooldown
}

func (r *CombatRoom) execEffect(caster *PlayerState, eff SkillEffectCfg, targetID int64, tx, ty int) {
	switch eff.Type {
	case "damage":
		target := r.resolveTarget(caster, eff.Target, targetID)
		if target != nil {
			dmg := eff.Amount * caster.Attack / 100
			dmg = dmg * 10000 / (10000 + target.Defense)
			target.HP -= dmg
			if target.HP <= 0 {
				r.onPlayerDeath(target)
			}
		}
	case "projectile":
		r.spawnProjectile(caster, eff, tx, ty)
	case "buff":
		target := r.resolveTarget(caster, eff.Target, targetID)
		if target != nil {
			r.ApplyBuff(target, eff.BuffID, caster.ID)
		}
	case "aoe":
		for _, eid := range r.AOI.GetNearby(tx, ty) {
			if eid == caster.ID {
				continue
			}
			target, ok := r.Players[eid]
			if !ok || target.HP <= 0 {
				continue
			}
			dx, dy := target.GridX-tx, target.GridY-ty
			if dx*dx+dy*dy <= eff.Radius*eff.Radius {
				target.HP -= eff.Amount
				if target.HP <= 0 {
					r.onPlayerDeath(target)
				}
			}
		}
	case "displacement":
		target := r.resolveTarget(caster, eff.Target, targetID)
		if target != nil && r.Grid.IsWalkable(tx, ty) {
			r.AOI.Move(target.ID, target.GridX, target.GridY, tx, ty)
			target.GridX, target.GridY = tx, ty
		}
	}
}

func (r *CombatRoom) resolveTarget(caster *PlayerState, targetType string, targetID int64) *PlayerState {
	switch targetType {
	case "self":
		return caster
	case "primary":
		if t, ok := r.Players[targetID]; ok && t.HP > 0 {
			return t
		}
	}
	return nil
}

func (r *CombatRoom) spawnProjectile(caster *PlayerState, eff SkillEffectCfg, tx, ty int) {
	dx, dy := float64(tx-caster.GridX), float64(ty-caster.GridY)
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return
	}
	r.nextProjID++
	r.Projectiles = append(r.Projectiles, &Projectile{
		ID: r.nextProjID, OwnerID: caster.ID,
		X: float64(caster.GridX), Y: float64(caster.GridY),
		DX: dx / length, DY: dy / length,
		Speed: eff.Speed, Damage: eff.Amount, Life: 2.0,
	})
}

func (r *CombatRoom) tickCooldowns(dt float64) {
	dtMs := int64(dt * 1000)
	for _, p := range r.Players {
		for id, cd := range p.Cooldowns {
			if cd > 0 {
				p.Cooldowns[id] = cd - dtMs
			}
		}
	}
}
