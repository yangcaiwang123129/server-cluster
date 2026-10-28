package combat

type BuffInstance struct {
	Cfg      *BuffConfig
	Source   int64
	Stacks   int
	RemainMs int64
	NextTick int64
}

func (r *CombatRoom) ApplyBuff(target *PlayerState, buffID int, source int64) {
	cfg, ok := BuffTable[buffID]
	if !ok {
		return
	}
	for _, b := range target.Buffs {
		if b.Cfg.ID == buffID {
			if b.Stacks < cfg.MaxStacks {
				b.Stacks++
			}
			b.RemainMs = cfg.DurationMs
			b.NextTick = cfg.TickMs
			return
		}
	}
	target.Buffs = append(target.Buffs, &BuffInstance{
		Cfg: cfg, Source: source, Stacks: 1,
		RemainMs: cfg.DurationMs, NextTick: cfg.TickMs,
	})
	r.recalcAttrs(target)
}

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
						p.HP -= b.Cfg.TickDamage * b.Stacks
						if p.HP <= 0 {
							r.onPlayerDeath(p)
							break
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

func (r *CombatRoom) recalcAttrs(p *PlayerState) {
	atkPct, defPct, spdPct := 0, 0, 0
	for _, b := range p.Buffs {
		v := b.Cfg.AttrValue * b.Stacks
		switch b.Cfg.AttrType {
		case "attack":
			atkPct += v
		case "defense":
			defPct += v
		case "speed":
			spdPct += v
		}
	}
	p.Attack = p.BaseAttack * (10000 + atkPct) / 10000
	p.Defense = p.BaseDefense * (10000 + defPct) / 10000
	p.Speed = p.BaseSpeed * (10000 + spdPct) / 10000
}

func (r *CombatRoom) onPlayerDeath(p *PlayerState) {
	p.HP = 0
	r.AOI.Remove(p.ID, p.GridX, p.GridY)
}
