package logic

import (
	"context"

	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/tpl"
)

// 属性计算框架：
//   - 英雄本体：加法成长，属性 = 基础值 + 成长值×(等级-起始等级)；星级决定取哪一档 AttributeRole 行。
//   - 武器/技能/装备：模块属性 = 基础值(1 星档) × 等级系数 × 品质系数 × 突破系数。
//
// 英雄本体行 ID = 英雄配置ID + 星级；武器/装备/技能属性行 ID = 星级*1e6 + 配置ID；维度表行 ID = 配置ID*1000 + 维度值。
// 各模块的属性贡献由各自 Logic 的 AttributeBonus 计算，computeHeroAttribute 按固定顺序汇总。
const attrTierStride int32 = 1000000

// 维度表行 ID 规则：配置ID*1000 + 维度值（等级/品质/突破）。
const dimStride int32 = 1000

// attrID 由星级档位与配置ID组装属性行 ID。
func attrID(tier, cfgID int32) int32 { return tier*attrTierStride + cfgID }

// dimID 由配置ID与维度值组装维度表行 ID。
func dimID(cfgID, dim int32) int32 { return cfgID*dimStride + dim }

// coefOr1 系数收敛：配置缺失或为 0 时按 1（不影响基础值）。
func coefOr1(v float32) float64 {
	if v <= 0 {
		return 1
	}
	return float64(v)
}

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

// levelBounds 属性表的等级区间，缺省且区间无效时回落到 1~10。
func levelBounds(lo, hi float32, hasRange bool) (int32, int32) {
	if !hasRange {
		return 1, 10
	}
	l, h := int32(lo), int32(hi)
	if h <= l {
		return 1, 10
	}
	return l, h
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

// ----- 维度公共逻辑 -----

// clampDim 维度下限收敛：等级/品质至少 1，突破至少 0。
func clampDim(v, min int32) int32 {
	if v < min {
		return min
	}
	return v
}

// dimConsume 维度提升前的公共校验与扣费：
// hasNext 为 false 表示已达上限（返回 fullMsg）；否则扣除当前维度行的素材消耗（use<=0 不扣）。
func dimConsume(ctx context.Context, playerID int64, material, use int32, hasNext bool, fullMsg string) error {
	if !hasNext {
		return errs.CodeInvalidParams.WithMessage(fullMsg).Err()
	}
	if use <= 0 {
		return nil
	}
	return Asset().AssetRemove(ctx, playerID, &tpl.Asset{
		Type:  tpl.AssetType_Item,
		Id:    material,
		Count: use,
	})
}

// ----- 英雄定位 -----

// equippedHero 取出战英雄；未出战返回错误。
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

// ownedHero 取玩家指定英雄；heroID<=0 时取当前出战英雄。
func ownedHero(ctx context.Context, playerID int64, heroID int32) (*model.PlayerHero, model.HeroInfo, error) {
	if heroID <= 0 {
		return equippedHero(ctx, playerID)
	}
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return nil, model.HeroInfo{}, err
	}
	hero, ok := row.Heroes.Get(heroID)
	if !ok {
		return nil, model.HeroInfo{}, errs.CodeInvalidParams.WithMessage("英雄未拥有").Err()
	}
	return row, hero, nil
}

// equippedHeroID 当前出战英雄ID；未出战返回 0（资产奖励按此默认归属，0 表示未分配）。
func equippedHeroID(ctx context.Context, playerID int64) int32 {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil || row.Heroes == nil {
		return 0
	}
	heroID := int32(0)
	row.Heroes.Range(func(_ int32, h model.HeroInfo) bool {
		if h.Equipped {
			heroID = h.HeroID
			return false
		}
		return true
	})
	return heroID
}

// computeHeroAttribute 计算英雄累计属性：英雄本体 + 武器 + 已穿戴装备的属性贡献。
// 核心属性为加法关系，战力 = 伤害 + 护甲 + 生命；技能只影响技能参数，不贡献核心属性。
func computeHeroAttribute(ctx context.Context, playerID int64, heroID int32) (*model.HeroInfo, *pb.HeroAttribute, error) {
	_, hero, err := ownedHero(ctx, playerID, heroID)
	if err != nil {
		return nil, nil, err
	}

	// 各模块贡献按固定顺序汇总：本体 → 武器 → 装备
	bonuses := make([]*pb.HeroAttribute, 0, 3)
	for _, bonus := range []func(context.Context, int64, int32) (*pb.HeroAttribute, error){
		Hero().AttributeBonus,
		Weapon().AttributeBonus,
		Equip().AttributeBonus,
	} {
		attr, err := bonus(ctx, playerID, hero.HeroID)
		if err != nil {
			return nil, nil, err
		}
		bonuses = append(bonuses, attr)
	}

	total := &pb.HeroAttribute{}
	for _, attr := range bonuses {
		if attr == nil {
			continue
		}
		total.Hp += attr.Hp
		total.Armor += attr.Armor
		total.Damage += attr.Damage
	}
	total.Power = total.Hp + total.Armor + total.Damage
	return &hero, total, nil
}
