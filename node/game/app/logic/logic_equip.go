package logic

import (
	"context"
	"errors"
	"server-cluster/internal/record"
	"sort"

	"gorm.io/gorm"

	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

type equipLogic struct{}

var equipLogicInst = singleton.NewContainer[global.EquipLogic]("EquipLogic")

func init() {
	equipLogicInst.Register(&equipLogic{})
	registerAsset(&equipAssetHandler{})
}

// Equip 获取装备业务单例。
func Equip() global.EquipLogic { return equipLogicInst.MustGet() }

// EquipList 装备持有列表（供 HeroList 聚合下发与 GM 查询）。
func (this *equipLogic) EquipList(ctx context.Context, playerID int64) ([]*pb.EquipEntry, error) {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}

	entries := make([]*pb.EquipEntry, 0, row.Equips.Len())
	row.Equips.Range(func(_ int64, g model.EquipInfo) bool {
		entries = append(entries, toEquipEntry(g))
		return true
	})
	// map 遍历无序，按实例ID排序保证客户端展示稳定
	sort.Slice(entries, func(i, j int) bool { return entries[i].Id < entries[j].Id })
	return entries, nil
}

// toEquipEntry 持有数据 → wire 条目（仅状态）。
func toEquipEntry(g model.EquipInfo) *pb.EquipEntry {
	return &pb.EquipEntry{
		Id:           g.EquipID,
		CfgId:        g.CfgID,
		HeroId:       g.HeroID,
		Level:        g.Level,
		Equipped:     g.Equipped,
		Quality:      g.Quality,
		Breakthrough: g.Breakthrough,
	}
}

// EquipAdd 新增装备：配置ID必须存在，装备归属于指定英雄（0 表示出战英雄），生成新的实例ID。
func (this *equipLogic) EquipAdd(ctx context.Context, playerID int64, req *pb.EquipAddReq) (*pb.EquipAddResp, error) {
	if !tpl.T().Equip.Has(req.GetCfgId()) {
		return nil, errs.CodeInvalidParams.Err()
	}

	_, hero, err := ownedHero(ctx, playerID, req.GetHeroId())
	if err != nil {
		return nil, err
	}

	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}

	g := model.EquipInfo{EquipID: svc.NextID(), CfgID: req.GetCfgId(), HeroID: hero.HeroID, Level: 1, Quality: 1}
	if !row.AddEquip(g) {
		return nil, errs.CodeInvalidParams.WithMessage("装备实例已存在").Err()
	}

	global.PlayerEquipCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleEquip, playerID, model.RecordActionGain, req.GetCfgId(), 1)
	return &pb.EquipAddResp{EquipEntry: toEquipEntry(g)}, nil
}

// EquipUpdate 更新装备：hero_id>0 时改归该英雄，0 保持原归属；按实例ID定位。
func (this *equipLogic) EquipUpdate(ctx context.Context, playerID int64, req *pb.EquipUpdateReq) (*pb.EquipUpdateResp, error) {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}

	old, ok := row.Equips.Get(req.GetId())
	if !ok {
		return nil, errs.CodeInvalidParams.WithMessage("装备未拥有").Err()
	}

	heroID := old.HeroID
	if req.GetHeroId() > 0 {
		if _, _, err := ownedHero(ctx, playerID, req.GetHeroId()); err != nil {
			return nil, err
		}
		heroID = req.GetHeroId()
	}

	g := model.EquipInfo{
		EquipID:      old.EquipID,
		CfgID:        old.CfgID,
		HeroID:       heroID,
		Level:        clampDim(req.GetLevel(), 1),
		Equipped:     req.GetEquipped() && heroID > 0,
		Quality:      clampDim(req.GetQuality(), 1),
		Breakthrough: clampDim(req.GetBreakthrough(), 0),
	}
	if !row.UpdateEquip(g) {
		return nil, errs.CodeInvalidParams.WithMessage("装备未拥有").Err()
	}

	global.PlayerEquipCache.Set(playerID, row)
	return &pb.EquipUpdateResp{EquipEntry: toEquipEntry(g)}, nil
}

// EquipWear 穿戴装备：归属改到目标英雄（hero_id=0 取当前出战英雄）。
// 同一英雄、同一装备配置ID只保留一件穿戴状态，重复穿戴时自动顶替旧的。
func (this *equipLogic) EquipWear(ctx context.Context, playerID int64, req *pb.EquipWearReq) (*pb.EquipWearResp, error) {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}

	g, ok := row.Equips.Get(req.GetId())
	if !ok {
		return nil, errs.CodeInvalidParams.WithMessage("装备未拥有").Err()
	}

	_, hero, err := ownedHero(ctx, playerID, req.GetHeroId())
	if err != nil {
		return nil, err
	}

	if err := row.Equips.UpdateAll(func(m map[int64]model.EquipInfo) error {
		if _, exists := m[g.EquipID]; !exists {
			return errs.CodeInvalidParams.WithMessage("装备未拥有").Err()
		}
		// 同英雄同配置互斥：先卸载该英雄已穿戴的同款，再写入本次穿戴
		for id, other := range m {
			if id != g.EquipID && other.Equipped && other.HeroID == hero.HeroID && other.CfgID == g.CfgID {
				other.Equipped = false
				m[id] = other
			}
		}
		g.Equipped = true
		g.HeroID = hero.HeroID
		m[g.EquipID] = g
		return nil
	}); err != nil {
		return nil, err
	}

	global.PlayerEquipCache.Set(playerID, row)
	_, attr, err := computeHeroAttribute(ctx, playerID, hero.HeroID)
	if err != nil {
		return nil, err
	}
	return &pb.EquipWearResp{EquipEntry: toEquipEntry(g), Attribute: attr}, nil
}

// EquipUnwear 卸载装备：清穿戴状态，归属英雄保留（0 表示未分配）。
func (this *equipLogic) EquipUnwear(ctx context.Context, playerID int64, req *pb.EquipUnwearReq) (*pb.EquipUnwearResp, error) {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}

	g, ok := row.Equips.Get(req.GetId())
	if !ok {
		return nil, errs.CodeInvalidParams.WithMessage("装备未拥有").Err()
	}
	if !g.Equipped {
		return nil, errs.CodeInvalidParams.WithMessage("装备未穿戴").Err()
	}

	heroID := g.HeroID
	g.Equipped = false
	if !row.UpdateEquip(g) {
		return nil, errs.CodeInvalidParams.WithMessage("装备未拥有").Err()
	}

	global.PlayerEquipCache.Set(playerID, row)
	attr := &pb.HeroAttribute{}
	if heroID > 0 {
		_, computed, cErr := computeHeroAttribute(ctx, playerID, heroID)
		if cErr != nil {
			return nil, cErr
		}
		attr = computed
	}
	return &pb.EquipUnwearResp{EquipEntry: toEquipEntry(g), Attribute: attr}, nil
}

// equipDimGrow 维度提升通用流程：校验持有与穿戴 → 取当前维度行得消耗 → 校验下一维度行存在 → 扣素材 → 维度+1。
func equipDimGrow(ctx context.Context, playerID int64, id int64,
	cur func(g model.EquipInfo) int32,
	set func(g *model.EquipInfo, v int32),
	rowOf func(cfgID, dim int32) (material, use int32, ok bool),
	fullMsg string,
) (model.EquipInfo, error) {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return model.EquipInfo{}, err
	}
	g, ok := row.Equips.Get(id)
	if !ok {
		return model.EquipInfo{}, errs.CodeInvalidParams.WithMessage("装备未拥有").Err()
	}
	if g.HeroID <= 0 {
		return model.EquipInfo{}, errs.CodeInvalidParams.WithMessage("请先将装备穿戴到英雄").Err()
	}

	dim := cur(g)
	material, use, ok := rowOf(g.CfgID, dim)
	if !ok {
		return model.EquipInfo{}, errs.CodeInvalidParams.WithMessage("装备维度配置不存在").Err()
	}
	_, _, hasNext := rowOf(g.CfgID, dim+1)
	if err := dimConsume(ctx, playerID, material, use, hasNext, fullMsg); err != nil {
		return model.EquipInfo{}, err
	}

	set(&g, dim+1)
	if !row.UpdateEquip(g) {
		return model.EquipInfo{}, errs.CodeInvalidParams.WithMessage("装备未拥有").Err()
	}
	global.PlayerEquipCache.Set(playerID, row)
	return g, nil
}

// EquipUpgrade 装备升级：消耗 GearLevel 表当前等级的升级素材，等级 +1。
func (this *equipLogic) EquipUpgrade(ctx context.Context, playerID int64, req *pb.EquipUpgradeReq) (*pb.EquipUpgradeResp, error) {
	g, err := equipDimGrow(ctx, playerID, req.GetId(),
		func(g model.EquipInfo) int32 { return g.Level },
		func(g *model.EquipInfo, v int32) { g.Level = v },
		func(cfgID, dim int32) (int32, int32, bool) {
			c := tpl.T().GearLevel.Get(dimID(cfgID, dim))
			if c == nil {
				return 0, 0, false
			}
			return c.UpgradeMaterial, c.UpgradeUse, true
		},
		"装备已满级")
	if err != nil {
		return nil, err
	}

	_, attr, err := computeHeroAttribute(ctx, playerID, g.HeroID)
	if err != nil {
		return nil, err
	}
	return &pb.EquipUpgradeResp{EquipEntry: toEquipEntry(g), Attribute: attr}, nil
}

// EquipUpQuality 装备升品：消耗 GearQuality 表当前品质的升品素材，品质 +1。
func (this *equipLogic) EquipUpQuality(ctx context.Context, playerID int64, req *pb.EquipUpQualityReq) (*pb.EquipUpQualityResp, error) {
	g, err := equipDimGrow(ctx, playerID, req.GetId(),
		func(g model.EquipInfo) int32 { return g.Quality },
		func(g *model.EquipInfo, v int32) { g.Quality = v },
		func(cfgID, dim int32) (int32, int32, bool) {
			c := tpl.T().GearQuality.Get(dimID(cfgID, dim))
			if c == nil {
				return 0, 0, false
			}
			return c.QualityMaterial, c.QualityUse, true
		},
		"装备已满品质")
	if err != nil {
		return nil, err
	}

	_, attr, err := computeHeroAttribute(ctx, playerID, g.HeroID)
	if err != nil {
		return nil, err
	}
	return &pb.EquipUpQualityResp{EquipEntry: toEquipEntry(g), Attribute: attr}, nil
}

// EquipBreakthrough 装备突破：消耗 GearBreakthrough 表当前突破的素材，突破 +1。
func (this *equipLogic) EquipBreakthrough(ctx context.Context, playerID int64, req *pb.EquipBreakthroughReq) (*pb.EquipBreakthroughResp, error) {
	g, err := equipDimGrow(ctx, playerID, req.GetId(),
		func(g model.EquipInfo) int32 { return g.Breakthrough },
		func(g *model.EquipInfo, v int32) { g.Breakthrough = v },
		func(cfgID, dim int32) (int32, int32, bool) {
			c := tpl.T().GearBreakthrough.Get(dimID(cfgID, dim))
			if c == nil {
				return 0, 0, false
			}
			return c.BreakthroughMaterial, c.BreakthroughUse, true
		},
		"装备已满突破")
	if err != nil {
		return nil, err
	}

	_, attr, err := computeHeroAttribute(ctx, playerID, g.HeroID)
	if err != nil {
		return nil, err
	}
	return &pb.EquipBreakthroughResp{EquipEntry: toEquipEntry(g), Attribute: attr}, nil
}

// EnterGame 进入游戏：加载玩家装备持有，未建档则初始化空集合。
func (this *equipLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if _, err := loadPlayerEquip(ctx, playerID); err != nil {
		return nil, err
	}
	// 装备数据由 EquipList 下发，这里只预热缓存
	return &pb.EnterGameResp{}, nil
}

// LeaveGame 离开游戏：装备持有数据同步落库，并逐出内存缓存。
func (this *equipLogic) LeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	if err := flushAndEvict(ctx, global.PlayerEquipCache, playerID); err != nil {
		return nil, err
	}
	return &pb.LeaveGameResp{}, nil
}

// loadPlayerEquip 加载玩家装备持有；未建档则初始化空 map 并挂入缓存（异步落库）。
func loadPlayerEquip(ctx context.Context, playerID int64) (*model.PlayerEquip, error) {
	row, err := global.PlayerEquipCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = &model.PlayerEquip{PlayerID: playerID, Equips: util.NewSafeMap[int64, model.EquipInfo]()}
		global.PlayerEquipCache.Set(playerID, row)
		return row, nil
	}
	if err != nil {
		return nil, err
	}
	if row.Equips == nil {
		row.Equips = util.NewSafeMap[int64, model.EquipInfo]()
	}
	return row, nil
}

// ============ 资产接口实现（AssetHandler，可叠加型：同配置可多件） ============

// equipAssetHandler 装备资产实现：落在 player_equip。
type equipAssetHandler struct{}

// AssetType 资产类型：装备。
func (this *equipAssetHandler) AssetType() tpl.AssetType { return tpl.AssetType_Equip }

// Add 发放装备：生成新的实例ID，默认归属当前出战英雄（未出战则未分配）。
func (this *equipAssetHandler) Add(ctx context.Context, playerID int64, cfgID, _ int32) error {
	if !tpl.T().Equip.Has(cfgID) {
		return errs.CodeInvalidParams.WithMessage("装备配置不存在").Err()
	}

	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return err
	}
	g := model.EquipInfo{EquipID: svc.NextID(), CfgID: cfgID, HeroID: equippedHeroID(ctx, playerID), Level: 1, Quality: 1}
	if !row.AddEquip(g) {
		return errs.CodeInvalidParams.WithMessage("装备实例已存在").Err()
	}

	global.PlayerEquipCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleEquip, playerID, model.RecordActionGain, cfgID, 1)
	return nil
}

// Remove 扣除一件指定配置ID的装备实例；未拥有返回错误。
func (this *equipAssetHandler) Remove(ctx context.Context, playerID int64, cfgID, _ int32) error {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return err
	}
	var target int64
	found := false
	row.Equips.Range(func(equipID int64, g model.EquipInfo) bool {
		if g.CfgID == cfgID {
			target, found = equipID, true
			return false
		}
		return true
	})
	if !found {
		return errs.CodeInvalidParams.WithMessage("装备未拥有").Err()
	}
	row.DeleteEquip(target)

	global.PlayerEquipCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleEquip, playerID, model.RecordActionConsume, cfgID, 1)
	return nil
}

// Update 覆盖持有：count<=0 移除一件，>0 不足时补一件。
func (this *equipAssetHandler) Update(ctx context.Context, playerID int64, cfgID, count int32) error {
	owned, err := this.Query(ctx, playerID, cfgID)
	if err != nil {
		return err
	}
	switch {
	case count <= 0 && owned > 0:
		return this.Remove(ctx, playerID, cfgID, 1)
	case count > 0 && owned == 0:
		return this.Add(ctx, playerID, cfgID, 1)
	default:
		return nil
	}
}

// Query 查询指定配置ID的装备持有数量。
func (this *equipAssetHandler) Query(ctx context.Context, playerID int64, cfgID int32) (int32, error) {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return 0, err
	}
	var count int32
	row.Equips.Range(func(_ int64, g model.EquipInfo) bool {
		if g.CfgID == cfgID {
			count++
		}
		return true
	})
	return count, nil
}

// List 列出全部装备（按配置ID聚合数量，再按配置ID升序）。
func (this *equipAssetHandler) List(ctx context.Context, playerID int64) ([]*tpl.Asset, error) {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}

	counts := make(map[int32]int32)
	row.Equips.Range(func(_ int64, g model.EquipInfo) bool {
		counts[g.CfgID]++
		return true
	})
	assets := make([]*tpl.Asset, 0, len(counts))
	for cfgID, count := range counts {
		assets = append(assets, &tpl.Asset{Type: tpl.AssetType_Equip, Id: cfgID, Count: count})
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Id < assets[j].Id })
	return assets, nil
}

// AttributeBonus 装备模块对英雄的生命/护甲贡献：基础值 × 等级系数 × 品质系数 × 突破系数。
// 基础值取 AttributeGear 的 1 星档行；只统计穿戴在该英雄身上的装备。
func (this *equipLogic) AttributeBonus(ctx context.Context, playerID int64, heroID int32) (*pb.HeroAttribute, error) {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}

	hp, armor := int32(0), int32(0)
	row.Equips.Range(func(_ int64, g model.EquipInfo) bool {
		if !g.Equipped || g.HeroID != heroID {
			return true
		}
		base := tpl.T().AttributeGear.Get(attrID(1, g.CfgID))
		if base == nil {
			return true
		}
		lhp, larmor := gearLevelCoef(g.CfgID, g.Level)
		qhp, qarmor := gearQualityCoef(g.CfgID, g.Quality)
		bhp, barmor := gearBreakthroughCoef(g.CfgID, g.Breakthrough)
		hp += int32(float64(base.Hp) * coefOr1(lhp) * coefOr1(qhp) * coefOr1(bhp))
		armor += int32(float64(base.Armor) * coefOr1(larmor) * coefOr1(qarmor) * coefOr1(barmor))
		return true
	})
	return &pb.HeroAttribute{Hp: hp, Armor: armor}, nil
}

// 装备维度系数查询（行缺失按 1）。
func gearLevelCoef(cfgID, level int32) (hp, armor float32) {
	if c := tpl.T().GearLevel.Get(dimID(cfgID, level)); c != nil {
		return c.HpCoef, c.ArmorCoef
	}
	return 1, 1
}

func gearQualityCoef(cfgID, quality int32) (hp, armor float32) {
	if c := tpl.T().GearQuality.Get(dimID(cfgID, quality)); c != nil {
		return c.HpCoef, c.ArmorCoef
	}
	return 1, 1
}

func gearBreakthroughCoef(cfgID, breakthrough int32) (hp, armor float32) {
	if c := tpl.T().GearBreakthrough.Get(dimID(cfgID, breakthrough)); c != nil {
		return c.HpCoef, c.ArmorCoef
	}
	return 1, 1
}
