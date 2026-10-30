package logic

import (
	"context"
	"errors"
	"server-cluster/internal/record"
	"sort"

	"gorm.io/gorm"

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
	registerAssetHandler(&equipAssetHandler{})
}

// Equip 获取装备业务单例。
func Equip() global.EquipLogic { return equipLogicInst.MustGet() }

// EquipList 装备列表：只下发持有状态，静态属性由客户端按 equip_id 关联装备配置表。
func (this *equipLogic) EquipList(ctx context.Context, playerID int64) (*pb.EquipListResp, error) {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}

	entries := make([]*pb.EquipEntry, 0, row.Equips.Len())
	row.Equips.Range(func(_ int32, g model.EquipInfo) bool {
		entries = append(entries, toEquipEntry(g))
		return true
	})
	// map 遍历无序，按配置ID排序保证客户端展示稳定
	sort.Slice(entries, func(i, j int) bool { return entries[i].CfgId < entries[j].CfgId })
	return &pb.EquipListResp{EquipEntries: entries}, nil
}

// toEquipEntry 持有数据 → wire 条目（仅状态）。
func toEquipEntry(g model.EquipInfo) *pb.EquipEntry {
	return &pb.EquipEntry{
		CfgId:    g.EquipID,
		Level:    g.Level,
		Equipped: g.Equipped,
	}
}

// EquipAdd 新增装备：配置ID必须存在，已拥有返回错误。
func (this *equipLogic) EquipAdd(ctx context.Context, playerID int64, req *pb.EquipAddReq) (*pb.EquipAddResp, error) {
	if !tpl.T().Equip.Has(req.GetCfgId()) {
		return nil, errs.CodeInvalidParams.Err()
	}

	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}

	g := model.EquipInfo{EquipID: req.GetCfgId()}
	if !row.AddEquip(g) {
		return nil, errs.CodeInvalidParams.WithMessage("装备已拥有").Err()
	}

	global.PlayerEquipCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleEquip, playerID, model.RecordActionGain, req.GetCfgId(), 1)
	return &pb.EquipAddResp{EquipEntry: toEquipEntry(g)}, nil
}

// EquipUpdate 更新装备：整体覆盖持有状态，未拥有返回错误。
func (this *equipLogic) EquipUpdate(ctx context.Context, playerID int64, req *pb.EquipUpdateReq) (*pb.EquipUpdateResp, error) {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}

	g := model.EquipInfo{
		EquipID:  req.GetCfgId(),
		Level:    req.GetLevel(),
		Equipped: req.GetEquipped(),
	}
	if !row.UpdateEquip(g) {
		return nil, errs.CodeInvalidParams.WithMessage("装备未拥有").Err()
	}

	global.PlayerEquipCache.Set(playerID, row)
	return &pb.EquipUpdateResp{EquipEntry: toEquipEntry(g)}, nil
}

// EquipUpgrade 装备升级：星级档位沿用出战英雄，消耗配置表升级素材，等级 +1。
func (this *equipLogic) EquipUpgrade(ctx context.Context, playerID int64, req *pb.EquipUpgradeReq) (*pb.EquipUpgradeResp, error) {
	_, hero, err := equippedHero(ctx, playerID)
	if err != nil {
		return nil, err
	}

	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}
	g, ok := row.Equips.Get(req.GetCfgId())
	if !ok {
		return nil, errs.CodeInvalidParams.WithMessage("装备未拥有").Err()
	}

	cfg := tpl.T().AttributeGear.Get(attrID(clampTier(hero.Star), g.EquipID))
	if cfg == nil {
		return nil, errs.CodeInvalidParams.WithMessage("装备属性配置不存在").Err()
	}
	lo, hi := levelBounds(0, 0, false)
	if cfg.Level != nil {
		lo, hi = levelBounds(cfg.Level.X, cfg.Level.Y, true)
	}
	lvl := clampLevel(g.Level, lo, hi)
	if lvl >= hi {
		return nil, errs.CodeInvalidParams.WithMessage("装备已满级").Err()
	}

	cost := costAt(cfg.UpgradeUse, cfg.GrowUGMaterial, lvl, lo)
	if cost > 0 {
		if err := Asset().AssetRemove(ctx, playerID, &tpl.Asset{
			Type:  tpl.AssetType_Item,
			Id:    cfg.UpgradeMaterial,
			Count: cost,
		}); err != nil {
			return nil, err
		}
	}

	g.Level = lvl + 1
	if !row.UpdateEquip(g) {
		return nil, errs.CodeInvalidParams.WithMessage("装备未拥有").Err()
	}
	global.PlayerEquipCache.Set(playerID, row)

	_, attr, err := computeHeroAttribute(ctx, playerID, 0)
	if err != nil {
		return nil, err
	}
	return &pb.EquipUpgradeResp{EquipEntry: toEquipEntry(g), Attribute: attr}, nil
}

// EnterGame 进入游戏：加载玩家装备持有，未建档则初始化空集合。
func (this *equipLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if _, err := loadPlayerEquip(ctx, playerID); err != nil {
		return nil, err
	}
	// 装备数据由 EquipList 下发，这里只预热缓存
	return nil, nil
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
		row = &model.PlayerEquip{PlayerID: playerID, Equips: util.NewSafeMap[int32, model.EquipInfo]()}
		global.PlayerEquipCache.Set(playerID, row)
		return row, nil
	}
	if err != nil {
		return nil, err
	}
	if row.Equips == nil {
		row.Equips = util.NewSafeMap[int32, model.EquipInfo]()
	}
	return row, nil
}

// ============ 资产接口实现（AssetHandler，唯一持有型） ============

// equipAssetHandler 装备资产实现：落在 player_equip。
type equipAssetHandler struct{}

// AssetType 资产类型：装备。
func (this *equipAssetHandler) AssetType() tpl.AssetType { return tpl.AssetType_Equip }

// Add 发放装备；已拥有返回错误。
func (this *equipAssetHandler) Add(ctx context.Context, playerID int64, cfgID, _ int32) error {
	if !tpl.T().Equip.Has(cfgID) {
		return errs.CodeInvalidParams.WithMessage("装备配置不存在").Err()
	}

	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return err
	}
	if !row.AddEquip(model.EquipInfo{EquipID: cfgID}) {
		return errs.CodeInvalidParams.WithMessage("装备已拥有").Err()
	}

	global.PlayerEquipCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleEquip, playerID, model.RecordActionGain, cfgID, 1)
	return nil
}

// Remove 扣除装备；未拥有返回错误。
func (this *equipAssetHandler) Remove(ctx context.Context, playerID int64, cfgID, _ int32) error {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return err
	}
	if !row.Equips.Delete(cfgID) {
		return errs.CodeInvalidParams.WithMessage("装备未拥有").Err()
	}

	global.PlayerEquipCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleEquip, playerID, model.RecordActionConsume, cfgID, 1)
	return nil
}

// Update 覆盖持有：count<=0 移除，>0 确保持有。
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

// Query 查询是否持有：持有 1，未持有 0。
func (this *equipAssetHandler) Query(ctx context.Context, playerID int64, cfgID int32) (int32, error) {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return 0, err
	}
	if _, ok := row.Equips.Get(cfgID); !ok {
		return 0, nil
	}
	return 1, nil
}

// List 列出全部装备（按配置ID升序）。
func (this *equipAssetHandler) List(ctx context.Context, playerID int64) ([]*tpl.Asset, error) {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}

	assets := make([]*tpl.Asset, 0, row.Equips.Len())
	row.Equips.Range(func(cfgID int32, _ model.EquipInfo) bool {
		assets = append(assets, &tpl.Asset{Type: tpl.AssetType_Equip, Id: cfgID, Count: 1})
		return true
	})
	sort.Slice(assets, func(i, j int) bool { return assets[i].Id < assets[j].Id })
	return assets, nil
}
