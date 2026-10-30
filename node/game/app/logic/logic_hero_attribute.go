package logic

import (
	"context"

	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/tpl"
)

// 属性档位规则：同一英雄按星级分 6 档（1~6），武器/装备/技能沿用出战英雄的星级。
// AttributeRole 行 ID = 英雄ID + 星级；AttributeWeapon/Gear/Skill 行 ID = 星级*1e6 + 配置ID。
const attrTierStride int32 = 1000000

// attrID 由星级档位与配置ID组装属性行 ID。
func attrID(tier, cfgID int32) int32 { return tier*attrTierStride + cfgID }

// clampTier 星级档位收敛到 1~6（属性表每项配置只有 6 档）。
func clampTier(star int32) int32 {
	if star < 1 {
		return 1
	}
	if star > 6 {
		return 6
	}
	return star
}

// levelBounds 属性表的等级区间，缺省 1~10。
func levelBounds(lo, hi float32, hasRange bool) (int32, int32) {
	if !hasRange {
		return 1, 10
	}
	return int32(lo), int32(hi)
}

// statAt 等级 lvl 时的属性值：基础值 + 成长值*(lvl-lo)。
func statAt(base, grow, lvl, lo int32) int32 {
	d := lvl - lo
	if d < 0 {
		d = 0
	}
	return base + grow*d
}

// costAt 从 lvl 升到 lvl+1 的素材消耗：基础消耗 + 增长*(lvl-lo)。
func costAt(use, grow, lvl, lo int32) int32 {
	d := lvl - lo
	if d < 0 {
		d = 0
	}
	cost := use + grow*d
	if cost < 0 {
		cost = 0
	}
	return cost
}

// clampLevel 等级收敛到配置区间。
func clampLevel(lvl, lo, hi int32) int32 {
	if lvl < lo {
		return lo
	}
	if lvl > hi {
		return hi
	}
	return lvl
}

// equippedHero 取出出战英雄；未出战返回错误。
func equippedHero(ctx context.Context, playerID int64) (*model.PlayerHero, model.HeroInfo, error) {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return nil, model.HeroInfo{}, err
	}
	var hero model.HeroInfo
	found := false
	if row.Heroes != nil {
		row.Heroes.Range(func(_ int32, h model.HeroInfo) bool {
			if h.Equipped {
				hero, found = h, true
				return false
			}
			return true
		})
	}
	if !found {
		return nil, model.HeroInfo{}, errs.CodeInvalidParams.WithMessage("请先出战英雄").Err()
	}
	return row, hero, nil
}

// computeHeroAttribute 计算英雄累计属性：
// 英雄基础属性（星级档位 + 等级）+ 已装备武器伤害 + 已穿戴装备生命/护甲；
// 战力 = 伤害 + 护甲 + 生命。技能属性表只提升技能参数（CD/范围/时长），不计入核心属性。
func computeHeroAttribute(ctx context.Context, playerID int64, heroID int32) (*model.HeroInfo, *pb.HeroAttribute, error) {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return nil, nil, err
	}

	var hero model.HeroInfo
	if heroID > 0 {
		h, ok := row.Heroes.Get(heroID)
		if !ok {
			return nil, nil, errs.CodeInvalidParams.WithMessage("英雄未拥有").Err()
		}
		hero = h
	} else {
		_, h, err := equippedHero(ctx, playerID)
		if err != nil {
			return nil, nil, err
		}
		hero = h
	}

	tier := clampTier(hero.Star)
	roleCfg := tpl.T().AttributeRole.Get(hero.HeroID + tier)
	if roleCfg == nil {
		return nil, nil, errs.CodeInvalidParams.WithMessage("英雄属性配置不存在").Err()
	}

	lo, hi := levelBounds(0, 0, false)
	if roleCfg.Level != nil {
		lo, hi = levelBounds(roleCfg.Level.X, roleCfg.Level.Y, true)
	}
	lvl := clampLevel(hero.Level, lo, hi)
	hp := statAt(roleCfg.Hp, roleCfg.GrowHp, lvl, lo)
	armor := statAt(roleCfg.Armor, roleCfg.GrowArmor, lvl, lo)
	damage := statAt(roleCfg.Damage, roleCfg.GrowDamage, lvl, lo)

	// 已装备武器：只贡献伤害
	wrow, err := loadPlayerWeapon(ctx, playerID)
	if err != nil {
		return nil, nil, err
	}
	if wrow.Weapons != nil {
		wrow.Weapons.Range(func(_ int32, w model.WeaponInfo) bool {
			if !w.Equipped {
				return true
			}
			cfg := tpl.T().AttributeWeapon.Get(attrID(tier, w.WeaponID))
			if cfg == nil {
				return true
			}
			wlo, whi := levelBounds(0, 0, false)
			if cfg.Level != nil {
				wlo, whi = levelBounds(cfg.Level.X, cfg.Level.Y, true)
			}
			wlvl := clampLevel(w.Level, wlo, whi)
			damage += statAt(cfg.WeaponDamage, cfg.GrowDamage, wlvl, wlo)
			return true
		})
	}

	// 已穿戴装备：贡献生命与护甲
	grow, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, nil, err
	}
	if grow.Equips != nil {
		grow.Equips.Range(func(_ int32, g model.EquipInfo) bool {
			if !g.Equipped {
				return true
			}
			cfg := tpl.T().AttributeGear.Get(attrID(tier, g.EquipID))
			if cfg == nil {
				return true
			}
			// 装备表按配置ID索引（属性行ID 已含星级档位）
			clo, chi := levelBounds(0, 0, false)
			if cfg.Level != nil {
				clo, chi = levelBounds(cfg.Level.X, cfg.Level.Y, true)
			}
			clvl := clampLevel(g.Level, clo, chi)
			hp += statAt(cfg.Hp, cfg.GrowHp, clvl, clo)
			armor += statAt(cfg.Armor, cfg.GrowArmor, clvl, clo)
			return true
		})
	}

	return &hero, &pb.HeroAttribute{
		Power:  damage + armor + hp,
		Hp:     hp,
		Armor:  armor,
		Damage: damage,
	}, nil
}
