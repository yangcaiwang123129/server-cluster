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

type weaponLogic struct{}

var weaponLogicInst = singleton.NewContainer[global.WeaponLogic]("WeaponLogic")

func init() {
	weaponLogicInst.Register(&weaponLogic{})
	registerAssetHandler(&weaponAssetHandler{})
}

// Weapon 获取武器业务单例。
func Weapon() global.WeaponLogic { return weaponLogicInst.MustGet() }

// WeaponList 武器列表：只下发持有状态，静态属性由客户端按 weapon_id 关联武器配置表。
func (this *weaponLogic) WeaponList(ctx context.Context, playerID int64) (*pb.WeaponListResp, error) {
	row, err := loadPlayerWeapon(ctx, playerID)
	if err != nil {
		return nil, err
	}

	entries := make([]*pb.WeaponEntry, 0, row.Weapons.Len())
	row.Weapons.Range(func(_ int32, w model.WeaponInfo) bool {
		entries = append(entries, toWeaponEntry(w))
		return true
	})
	// map 遍历无序，按配置ID排序保证客户端展示稳定
	sort.Slice(entries, func(i, j int) bool { return entries[i].CfgId < entries[j].CfgId })
	return &pb.WeaponListResp{WeaponEntries: entries}, nil
}

// toWeaponEntry 持有数据 → wire 条目（仅状态）。
func toWeaponEntry(w model.WeaponInfo) *pb.WeaponEntry {
	return &pb.WeaponEntry{
		CfgId:      w.WeaponID,
		Level:      w.Level,
		Durability: w.Durability,
		Equipped:   w.Equipped,
	}
}

// WeaponAdd 新增武器：配置ID必须存在，已拥有返回错误。
func (this *weaponLogic) WeaponAdd(ctx context.Context, playerID int64, req *pb.WeaponAddReq) (*pb.WeaponAddResp, error) {
	if !tpl.T().Weapon.Has(req.GetCfgId()) {
		return nil, errs.CodeInvalidParams.Err()
	}

	row, err := loadPlayerWeapon(ctx, playerID)
	if err != nil {
		return nil, err
	}

	w := model.WeaponInfo{WeaponID: req.GetCfgId()}
	if !row.AddWeapon(w) {
		return nil, errs.CodeInvalidParams.WithMessage("武器已拥有").Err()
	}

	global.PlayerWeaponCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleWeapon, playerID, model.RecordActionGain, req.GetCfgId(), 1)
	return &pb.WeaponAddResp{WeaponEntry: toWeaponEntry(w)}, nil
}

// WeaponUpdate 更新武器：整体覆盖持有状态，未拥有返回错误。
func (this *weaponLogic) WeaponUpdate(ctx context.Context, playerID int64, req *pb.WeaponUpdateReq) (*pb.WeaponUpdateResp, error) {
	row, err := loadPlayerWeapon(ctx, playerID)
	if err != nil {
		return nil, err
	}

	w := model.WeaponInfo{
		WeaponID:   req.GetCfgId(),
		Level:      req.GetLevel(),
		Durability: req.GetDurability(),
		Equipped:   req.GetEquipped(),
	}
	if !row.UpdateWeapon(w) {
		return nil, errs.CodeInvalidParams.WithMessage("武器未拥有").Err()
	}

	global.PlayerWeaponCache.Set(playerID, row)
	return &pb.WeaponUpdateResp{WeaponEntry: toWeaponEntry(w)}, nil
}

// WeaponUpgrade 武器升级：星级档位沿用出战英雄，消耗配置表升级素材，等级 +1。
func (this *weaponLogic) WeaponUpgrade(ctx context.Context, playerID int64, req *pb.WeaponUpgradeReq) (*pb.WeaponUpgradeResp, error) {
	_, hero, err := equippedHero(ctx, playerID)
	if err != nil {
		return nil, err
	}

	row, err := loadPlayerWeapon(ctx, playerID)
	if err != nil {
		return nil, err
	}
	w, ok := row.Weapons.Get(req.GetCfgId())
	if !ok {
		return nil, errs.CodeInvalidParams.WithMessage("武器未拥有").Err()
	}

	cfg := tpl.T().AttributeWeapon.Get(attrID(clampTier(hero.Star), w.WeaponID))
	if cfg == nil {
		return nil, errs.CodeInvalidParams.WithMessage("武器属性配置不存在").Err()
	}
	lo, hi := levelBounds(0, 0, false)
	if cfg.Level != nil {
		lo, hi = levelBounds(cfg.Level.X, cfg.Level.Y, true)
	}
	lvl := clampLevel(w.Level, lo, hi)
	if lvl >= hi {
		return nil, errs.CodeInvalidParams.WithMessage("武器已满级").Err()
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

	w.Level = lvl + 1
	if !row.UpdateWeapon(w) {
		return nil, errs.CodeInvalidParams.WithMessage("武器未拥有").Err()
	}
	global.PlayerWeaponCache.Set(playerID, row)

	_, attr, err := computeHeroAttribute(ctx, playerID, 0)
	if err != nil {
		return nil, err
	}
	return &pb.WeaponUpgradeResp{WeaponEntry: toWeaponEntry(w), Attribute: attr}, nil
}

// EnterGame 进入游戏：加载玩家武器持有，未建档则初始化空集合。
func (this *weaponLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	row, err := loadPlayerWeapon(ctx, playerID)
	if err != nil {
		return nil, err
	}
	// 只下发当前装备武器的状态，静态属性由客户端关联武器配置表
	return &pb.EnterGameResp{WeaponEntry: equippedWeaponEntry(row)}, nil
}

// equippedWeaponEntry 当前装备武器；未装备返回 nil。
func equippedWeaponEntry(row *model.PlayerWeapon) *pb.WeaponEntry {
	if row.Weapons == nil {
		return nil
	}
	var entry *pb.WeaponEntry
	row.Weapons.Range(func(_ int32, w model.WeaponInfo) bool {
		if !w.Equipped {
			return true
		}
		entry = &pb.WeaponEntry{
			CfgId:      w.WeaponID,
			Level:      w.Level,
			Durability: w.Durability,
			Equipped:   true,
		}
		return false
	})
	return entry
}

// LeaveGame 离开游戏：武器持有数据同步落库，并逐出内存缓存。
func (this *weaponLogic) LeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	if err := flushAndEvict(ctx, global.PlayerWeaponCache, playerID); err != nil {
		return nil, err
	}
	return &pb.LeaveGameResp{}, nil
}

// loadPlayerWeapon 加载玩家武器持有；未建档则初始化空 map 并挂入缓存（异步落库）。
func loadPlayerWeapon(ctx context.Context, playerID int64) (*model.PlayerWeapon, error) {
	row, err := global.PlayerWeaponCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = &model.PlayerWeapon{PlayerID: playerID, Weapons: util.NewSafeMap[int32, model.WeaponInfo]()}
		global.PlayerWeaponCache.Set(playerID, row)
		return row, nil
	}
	if err != nil {
		return nil, err
	}
	if row.Weapons == nil {
		row.Weapons = util.NewSafeMap[int32, model.WeaponInfo]()
	}
	return row, nil
}

// ============ 资产接口实现（AssetHandler，唯一持有型） ============

// weaponAssetHandler 武器资产实现：落在 player_weapon。
type weaponAssetHandler struct{}

// AssetType 资产类型：武器。
func (this *weaponAssetHandler) AssetType() tpl.AssetType { return tpl.AssetType_Weapon }

// Add 发放武器；已拥有返回错误。
func (this *weaponAssetHandler) Add(ctx context.Context, playerID int64, cfgID, _ int32) error {
	if !tpl.T().Weapon.Has(cfgID) {
		return errs.CodeInvalidParams.WithMessage("武器配置不存在").Err()
	}

	row, err := loadPlayerWeapon(ctx, playerID)
	if err != nil {
		return err
	}
	if !row.AddWeapon(model.WeaponInfo{WeaponID: cfgID}) {
		return errs.CodeInvalidParams.WithMessage("武器已拥有").Err()
	}

	global.PlayerWeaponCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleWeapon, playerID, model.RecordActionGain, cfgID, 1)
	return nil
}

// Remove 扣除武器；未拥有返回错误。
func (this *weaponAssetHandler) Remove(ctx context.Context, playerID int64, cfgID, _ int32) error {
	row, err := loadPlayerWeapon(ctx, playerID)
	if err != nil {
		return err
	}
	if !row.Weapons.Delete(cfgID) {
		return errs.CodeInvalidParams.WithMessage("武器未拥有").Err()
	}

	global.PlayerWeaponCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleWeapon, playerID, model.RecordActionConsume, cfgID, 1)
	return nil
}

// Update 覆盖持有：count<=0 移除，>0 确保持有。
func (this *weaponAssetHandler) Update(ctx context.Context, playerID int64, cfgID, count int32) error {
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
func (this *weaponAssetHandler) Query(ctx context.Context, playerID int64, cfgID int32) (int32, error) {
	row, err := loadPlayerWeapon(ctx, playerID)
	if err != nil {
		return 0, err
	}
	if _, ok := row.Weapons.Get(cfgID); !ok {
		return 0, nil
	}
	return 1, nil
}

// List 列出全部武器（按配置ID升序）。
func (this *weaponAssetHandler) List(ctx context.Context, playerID int64) ([]*tpl.Asset, error) {
	row, err := loadPlayerWeapon(ctx, playerID)
	if err != nil {
		return nil, err
	}

	assets := make([]*tpl.Asset, 0, row.Weapons.Len())
	row.Weapons.Range(func(cfgID int32, _ model.WeaponInfo) bool {
		assets = append(assets, &tpl.Asset{Type: tpl.AssetType_Weapon, Id: cfgID, Count: 1})
		return true
	})
	sort.Slice(assets, func(i, j int) bool { return assets[i].Id < assets[j].Id })
	return assets, nil
}
