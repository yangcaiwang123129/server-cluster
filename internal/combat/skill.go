package combat

import (
	"math"
)

// 战斗数值常量。
const (
	// effectAmountScale 技能效果 Amount 相对攻击力的百分比基数。
	effectAmountScale = 100
	// bpUnit 万分比基数（10000 = 100%），用于防御减伤与属性加成。
	bpUnit = 10000
)

// CastSkill 释放技能：校验存在性、冷却与施法距离后依次执行效果。
func (r *CombatRoom) CastSkill(caster *PlayerState, skillID int, targetID int64, tx, ty int) {
	cfg, ok := SkillTable[skillID]
	if !ok || caster.HP <= 0 {
		return
	}
	if caster.Cooldowns[skillID] > 0 {
		return
	}
	if !inRange(caster.GridX, caster.GridY, tx, ty, cfg.Range) {
		return
	}
	for _, eff := range cfg.Effects {
		r.execEffect(caster, eff, targetID, tx, ty)
	}
	caster.Cooldowns[skillID] = cfg.Cooldown
}

// inRange 判断 (x,y) 是否在 (ox,oy) 半径 rangeLimit 内（欧氏距离平方比较）。
func inRange(ox, oy, x, y, rangeLimit int) bool {
	dx, dy := x-ox, y-oy
	return dx*dx+dy*dy <= rangeLimit*rangeLimit
}

// execEffect 按效果类型执行单个技能效果。
func (r *CombatRoom) execEffect(caster *PlayerState, eff SkillEffectCfg, targetID int64, tx, ty int) {
	switch eff.Type {
	case EffectDamage:
		if target := r.resolveTarget(caster, eff.Target, targetID); target != nil {
			dmg := eff.Amount * caster.Attack / effectAmountScale
			r.applyDamage(target, dmg*bpUnit/(bpUnit+target.Defense))
		}
	case EffectProjectile:
		r.spawnProjectile(caster, eff, tx, ty)
	case EffectBuff:
		if target := r.resolveTarget(caster, eff.Target, targetID); target != nil {
			r.ApplyBuff(target, eff.BuffID, caster.ID)
		}
	case EffectAOE:
		for _, eid := range r.AOI.GetNearby(tx, ty) {
			if eid == caster.ID {
				continue
			}
			target, ok := r.Players[eid]
			if !ok || target.HP <= 0 {
				continue
			}
			if inRange(tx, ty, target.GridX, target.GridY, eff.Radius) {
				r.applyDamage(target, eff.Amount)
			}
		}
	case EffectDisplacement:
		if target := r.resolveTarget(caster, eff.Target, targetID); target != nil && r.Grid.IsWalkable(tx, ty) {
			r.AOI.Move(target.ID, target.GridX, target.GridY, tx, ty)
			target.GridX, target.GridY = tx, ty
		}
	}
}

// resolveTarget 解析效果作用目标。
func (r *CombatRoom) resolveTarget(caster *PlayerState, targetType string, targetID int64) *PlayerState {
	switch targetType {
	case TargetSelf:
		return caster
	case TargetPrimary:
		if t, ok := r.Players[targetID]; ok && t.HP > 0 {
			return t
		}
	}
	return nil
}

// spawnProjectile 朝 (tx,ty) 方向生成一枚弹道。
func (r *CombatRoom) spawnProjectile(caster *PlayerState, eff SkillEffectCfg, tx, ty int) {
	dx, dy := float64(tx-caster.GridX), float64(ty-caster.GridY)
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist == 0 {
		return
	}
	r.nextProjectileID++
	r.Projectiles = append(r.Projectiles, &Projectile{
		ID:      r.nextProjectileID,
		OwnerID: caster.ID,
		X:       float64(caster.GridX),
		Y:       float64(caster.GridY),
		DX:      dx / dist,
		DY:      dy / dist,
		Speed:   eff.Speed,
		Damage:  eff.Amount,
		Life:    defaultProjectileLife,
	})
}

// tickCooldowns 按帧推进冷却；到期的条目直接清除。
func (r *CombatRoom) tickCooldowns(dt float64) {
	dtMs := int64(dt * 1000)
	for _, p := range r.Players {
		for skillID, remain := range p.Cooldowns {
			remain -= dtMs
			if remain > 0 {
				p.Cooldowns[skillID] = remain
			} else {
				delete(p.Cooldowns, skillID)
			}
		}
	}
}
