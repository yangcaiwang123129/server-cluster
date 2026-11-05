package logic

import (
	"context"
	"sort"

	"server-cluster/internal/cache"
	"server-cluster/internal/record"
	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
	"server-cluster/share/convert"
	"server-cluster/share/enum"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

var equipCache = cache.NewPlayerTableCache[model.PlayerEquip](cache.CacherConfig{Name: enum.TablePlayerEquip.String()})

type equipLogic struct{}

var equipLogicInst = singleton.NewContainer[global.EquipLogic]("EquipLogic")

func init() {
	equipLogicInst.Register(&equipLogic{})
	registerAsset(&equipAssetStore{})
}

// Equip 获取装备业务单例。
func Equip() global.EquipLogic { return equipLogicInst.MustGet() }

func (this *equipLogic) EquipList(ctx context.Context, playerID int64) ([]*pb.EquipEntry, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}

	entries := make([]*pb.EquipEntry, 0, row.Equips.Len())
	row.Equips.Range(func(_ int64, g model.EquipInfo) bool {
		entries = append(entries, convert.EquipEntry(g))
		return true
	})
	// map 遍历无序，按实例ID排序保证客户端展示稳定
	sort.Slice(entries, func(i, j int) bool { return entries[i].Id < entries[j].Id })
	return entries, nil
}

func (this *equipLogic) EquipAdd(ctx context.Context, playerID int64, req *pb.EquipAddReq) (*pb.EquipAddResp, error) {
	if !tpl.T().Equip.Has(req.GetCfgId()) {
		return &pb.EquipAddResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "")}, nil
	}

	_, hero, err := ownedHero(ctx, playerID, req.GetHeroId())
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.EquipAddResp{Result: r}, nil
		}
		return nil, err
	}

	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}

	g := model.EquipInfo{EquipID: svc.NextID(), CfgID: req.GetCfgId(), HeroID: hero.HeroID, Level: 1, Quality: 1}
	if !row.AddEquip(g) {
		return &pb.EquipAddResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "装备实例已存在")}, nil
	}

	this.Save(playerID, row)
	record.Record(ctx, model.RecordModuleEquip, playerID, model.RecordActionGain, req.GetCfgId(), 1)
	notifyHero(ctx, playerID, g.HeroID)
	return &pb.EquipAddResp{EquipEntry: convert.EquipEntry(g)}, nil
}

func (this *equipLogic) EquipUpdate(ctx context.Context, playerID int64, req *pb.EquipUpdateReq) (*pb.EquipUpdateResp, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}

	old, ok := row.Equips.Get(req.GetId())
	if !ok {
		return &pb.EquipUpdateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "装备未拥有")}, nil
	}

	oldHeroID := old.HeroID
	heroID := oldHeroID
	if req.GetHeroId() > 0 {
		if _, _, err := ownedHero(ctx, playerID, req.GetHeroId()); err != nil {
			if r := codedResult(err); r != nil {
				return &pb.EquipUpdateResp{Result: r}, nil
			}
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
		return &pb.EquipUpdateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "装备未拥有")}, nil
	}

	this.Save(playerID, row)
	if oldHeroID != heroID {
		notifyHero(ctx, playerID, oldHeroID)
	}
	notifyHero(ctx, playerID, heroID)
	return &pb.EquipUpdateResp{EquipEntry: convert.EquipEntry(g)}, nil
}

func (this *equipLogic) EquipWear(ctx context.Context, playerID int64, req *pb.EquipWearReq) (*pb.EquipWearResp, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}

	g, ok := row.Equips.Get(req.GetId())
	if !ok {
		return &pb.EquipWearResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "装备未拥有")}, nil
	}

	oldHeroID := g.HeroID
	_, hero, err := ownedHero(ctx, playerID, req.GetHeroId())
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.EquipWearResp{Result: r}, nil
		}
		return nil, err
	}

	if err := row.Equips.UpdateAll(func(m map[int64]model.EquipInfo) error {
		if _, exists := m[g.EquipID]; !exists {
			return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "装备未拥有")
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
		if r := codedResult(err); r != nil {
			return &pb.EquipWearResp{Result: r}, nil
		}
		return nil, err
	}

	this.Save(playerID, row)
	if oldHeroID != hero.HeroID {
		notifyHero(ctx, playerID, oldHeroID)
	}
	notifyHero(ctx, playerID, hero.HeroID)
	_, attr, err := computeHeroAttribute(ctx, playerID, hero.HeroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.EquipWearResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.EquipWearResp{EquipEntry: convert.EquipEntry(g), Attribute: attr}, nil
}

func (this *equipLogic) EquipUnwear(ctx context.Context, playerID int64, req *pb.EquipUnwearReq) (*pb.EquipUnwearResp, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}

	g, ok := row.Equips.Get(req.GetId())
	if !ok {
		return &pb.EquipUnwearResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "装备未拥有")}, nil
	}
	if !g.Equipped {
		return &pb.EquipUnwearResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "装备未穿戴")}, nil
	}

	heroID := g.HeroID
	g.Equipped = false
	if !row.UpdateEquip(g) {
		return &pb.EquipUnwearResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "装备未拥有")}, nil
	}

	this.Save(playerID, row)
	notifyHero(ctx, playerID, heroID)
	attr := &pb.HeroAttribute{}
	if heroID > 0 {
		_, computed, cErr := computeHeroAttribute(ctx, playerID, heroID)
		if cErr != nil {
			if r := codedResult(cErr); r != nil {
				return &pb.EquipUnwearResp{Result: r}, nil
			}
			return nil, cErr
		}
		attr = computed
	}
	return &pb.EquipUnwearResp{EquipEntry: convert.EquipEntry(g), Attribute: attr}, nil
}

// equipDimGrow 维度提升通用流程：校验持有与穿戴 → 取当前维度行得消耗 → 校验下一维度行存在 → 扣素材 → 维度+1。
func equipDimGrow(ctx context.Context, playerID int64, id int64,
	cur func(g model.EquipInfo) int32,
	set func(g *model.EquipInfo, v int32),
	rowOf func(cfgID, dim int32) (material, use int32, ok bool),
	fullMsg string,
) (model.EquipInfo, error) {
	row, err := Equip().Load(ctx, playerID)
	if err != nil {
		return model.EquipInfo{}, err
	}
	g, ok := row.Equips.Get(id)
	if !ok {
		return model.EquipInfo{}, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "装备未拥有")
	}
	if g.HeroID <= 0 {
		return model.EquipInfo{}, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "请先将装备穿戴到英雄")
	}

	dim := cur(g)
	material, use, ok := rowOf(g.CfgID, dim)
	if !ok {
		return model.EquipInfo{}, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "装备维度配置不存在")
	}
	_, _, hasNext := rowOf(g.CfgID, dim+1)
	if err := dimConsume(ctx, playerID, material, use, hasNext, fullMsg); err != nil {
		return model.EquipInfo{}, err
	}

	set(&g, dim+1)
	if !row.UpdateEquip(g) {
		return model.EquipInfo{}, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "装备未拥有")
	}
	equipCache.Set(playerID, row)
	notifyHero(ctx, playerID, g.HeroID)
	return g, nil
}

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
		if r := codedResult(err); r != nil {
			return &pb.EquipUpgradeResp{Result: r}, nil
		}
		return nil, err
	}

	_, attr, err := computeHeroAttribute(ctx, playerID, g.HeroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.EquipUpgradeResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.EquipUpgradeResp{EquipEntry: convert.EquipEntry(g), Attribute: attr}, nil
}

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
		if r := codedResult(err); r != nil {
			return &pb.EquipUpQualityResp{Result: r}, nil
		}
		return nil, err
	}

	_, attr, err := computeHeroAttribute(ctx, playerID, g.HeroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.EquipUpQualityResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.EquipUpQualityResp{EquipEntry: convert.EquipEntry(g), Attribute: attr}, nil
}

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
		if r := codedResult(err); r != nil {
			return &pb.EquipBreakthroughResp{Result: r}, nil
		}
		return nil, err
	}

	_, attr, err := computeHeroAttribute(ctx, playerID, g.HeroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.EquipBreakthroughResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.EquipBreakthroughResp{EquipEntry: convert.EquipEntry(g), Attribute: attr}, nil
}

func (this *equipLogic) OnEnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq, _ *pb.EnterGameResp) error {
	if _, err := this.Load(ctx, playerID); err != nil {
		return err
	}
	// 装备数据由 EquipList 下发，这里只预热缓存
	return nil
}

func (this *equipLogic) OnLeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq, _ *pb.LeaveGameResp) error {
	if err := flushAndEvict(ctx, equipCache, playerID); err != nil {
		return err
	}
	return nil
}

func (this *equipLogic) Load(ctx context.Context, playerID int64) (*model.PlayerEquip, error) {
	return equipCache.GetOrInit(ctx, playerID,
		func(id int64) *model.PlayerEquip {
			return &model.PlayerEquip{PlayerID: id, Equips: util.NewSafeMap[int64, model.EquipInfo]()}
		},
		func(row *model.PlayerEquip) {
			if row.Equips == nil {
				row.Equips = util.NewSafeMap[int64, model.EquipInfo]()
			}
		})
}

func (this *equipLogic) Save(playerID int64, row *model.PlayerEquip) {
	equipCache.Set(playerID, row)
}

func (this *equipLogic) Flush(ctx context.Context, playerID int64) error {
	return flushAndEvict(ctx, equipCache, playerID)
}

// ============ 资产接口实现（AssetStore，可叠加型：同配置可多件） ============

// equipAssetStore 装备资产实现：落在 player_equip。
type equipAssetStore struct{}

func (this *equipAssetStore) AssetType() tpl.AssetType { return tpl.AssetType_Equip }

func (this *equipAssetStore) Add(ctx context.Context, playerID int64, cfgID, _ int32) error {
	if !tpl.T().Equip.Has(cfgID) {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "装备配置不存在")
	}

	row, err := Equip().Load(ctx, playerID)
	if err != nil {
		return err
	}
	g := model.EquipInfo{EquipID: svc.NextID(), CfgID: cfgID, HeroID: equippedHeroID(ctx, playerID), Level: 1, Quality: 1}
	if !row.AddEquip(g) {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "装备实例已存在")
	}

	equipCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleEquip, playerID, model.RecordActionGain, cfgID, 1)
	notifyHero(ctx, playerID, g.HeroID)
	return nil
}

func (this *equipAssetStore) Remove(ctx context.Context, playerID int64, cfgID, _ int32) error {
	row, err := Equip().Load(ctx, playerID)
	if err != nil {
		return err
	}
	equipID, info, ok := row.Equips.First(func(_ int64, g model.EquipInfo) bool {
		return g.CfgID == cfgID
	})
	if !ok {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "装备未拥有")
	}
	row.DeleteEquip(equipID)

	equipCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleEquip, playerID, model.RecordActionConsume, cfgID, 1)
	notifyHero(ctx, playerID, info.HeroID)
	return nil
}

func (this *equipAssetStore) Update(ctx context.Context, playerID int64, cfgID, count int32) error {
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

func (this *equipAssetStore) Query(ctx context.Context, playerID int64, cfgID int32) (int32, error) {
	row, err := Equip().Load(ctx, playerID)
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

func (this *equipAssetStore) List(ctx context.Context, playerID int64) ([]*tpl.Asset, error) {
	row, err := Equip().Load(ctx, playerID)
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

func (this *equipLogic) AttributeBonus(ctx context.Context, playerID int64, heroID int32) (*pb.HeroAttribute, error) {
	row, err := this.Load(ctx, playerID)
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

// gearLevelCoef 装备等级维度系数查询（行缺失按 1）。
func gearLevelCoef(cfgID, level int32) (hp, armor float32) {
	if c := tpl.T().GearLevel.Get(dimID(cfgID, level)); c != nil {
		return c.HpCoef, c.ArmorCoef
	}
	return 1, 1
}

// gearQualityCoef 装备品质维度系数查询（行缺失按 1）。
func gearQualityCoef(cfgID, quality int32) (hp, armor float32) {
	if c := tpl.T().GearQuality.Get(dimID(cfgID, quality)); c != nil {
		return c.HpCoef, c.ArmorCoef
	}
	return 1, 1
}

// gearBreakthroughCoef 装备突破维度系数查询（行缺失按 1）。
func gearBreakthroughCoef(cfgID, breakthrough int32) (hp, armor float32) {
	if c := tpl.T().GearBreakthrough.Get(dimID(cfgID, breakthrough)); c != nil {
		return c.HpCoef, c.ArmorCoef
	}
	return 1, 1
}
