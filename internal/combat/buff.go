package combat

// BuffInstance 玩家身上的 buff 实例。
type BuffInstance struct {
	Cfg      *BuffConfig
	Source   int64
	Stacks   int
	RemainMs int64 // 剩余持续时间（毫秒）
	NextTick int64 // 距下次周期结算（毫秒）
}

// ApplyBuff 施加 buff：已存在则叠层并刷新时长，否则新增；随后重算属性。
func (r *CombatRoom) ApplyBuff(target *PlayerState, buffID int, source int64) {
	cfg, ok := BuffTable[buffID]
	if !ok {
		return
	}
	for _, b := range target.Buffs {
		if b.Cfg.ID != buffID {
			continue
		}
		if b.Stacks < cfg.MaxStacks {
			b.Stacks++
		}
		b.RemainMs = cfg.DurationMs
		b.NextTick = cfg.TickMs
		r.recalcAttrs(target)
		return
	}
	target.Buffs = append(target.Buffs, &BuffInstance{
		Cfg: cfg, Source: source, Stacks: 1,
		RemainMs: cfg.DurationMs, NextTick: cfg.TickMs,
	})
	r.recalcAttrs(target)
}

// tickBuffs 按帧推进 buff：周期伤害结算与到期移除。
func (r *CombatRoom) tickBuffs(dt float64) {
	dtMs := int64(dt * 1000)
	for _, p := range r.Players {
		if p.HP <= 0 {
			continue
		}
		removed := false
		for i := len(p.Buffs) - 1; i >= 0; i-- {
			b := p.Buffs[i]
			b.RemainMs -= dtMs
			if b.Cfg.TickMs > 0 {
				b.NextTick -= dtMs
				if b.NextTick <= 0 {
					if b.Cfg.TickDamage > 0 {
						r.applyDamage(p, b.Cfg.TickDamage*b.Stacks)
						if p.HP <= 0 {
							break // 玩家已死亡，停止结算其剩余 buff
						}
					}
					b.NextTick += b.Cfg.TickMs
				}
			}
			if b.RemainMs <= 0 {
				p.Buffs = append(p.Buffs[:i], p.Buffs[i+1:]...)
				removed = true
			}
		}
		if removed {
			r.recalcAttrs(p)
		}
	}
}

// recalcAttrs 依据身上 buff 重算实时属性（加成按万分比叠加）。
func (r *CombatRoom) recalcAttrs(p *PlayerState) {
	atkBp, defBp, spdBp := 0, 0, 0
	for _, b := range p.Buffs {
		v := b.Cfg.AttrValue * b.Stacks
		if b.Cfg.AttrPercent {
			v *= bpUnit / effectAmountScale // 百分比 → 万分比
		}
		switch b.Cfg.AttrType {
		case AttrAttack:
			atkBp += v
		case AttrDefense:
			defBp += v
		case AttrSpeed:
			spdBp += v
		}
	}
	p.Attack = p.BaseAttack * (bpUnit + atkBp) / bpUnit
	p.Defense = p.BaseDefense * (bpUnit + defBp) / bpUnit
	p.Speed = p.BaseSpeed * (bpUnit + spdBp) / bpUnit
}
