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

type skillLogic struct{}

var skillLogicInst = singleton.NewContainer[global.SkillLogic]("SkillLogic")

func init() {
	skillLogicInst.Register(&skillLogic{})
	registerAssetHandler(&skillAssetHandler{})
}

// Skill 获取技能业务单例。
func Skill() global.SkillLogic { return skillLogicInst.MustGet() }

// SkillList 技能列表：只下发持有状态，静态属性由客户端按 skill_id 关联技能配置表。
func (this *skillLogic) SkillList(ctx context.Context, playerID int64) (*pb.SkillListResp, error) {
	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return nil, err
	}

	entries := make([]*pb.SkillEntry, 0, row.Skills.Len())
	row.Skills.Range(func(_ int32, s model.SkillInfo) bool {
		entries = append(entries, toSkillEntry(s))
		return true
	})
	// map 遍历无序，按配置ID排序保证客户端展示稳定
	sort.Slice(entries, func(i, j int) bool { return entries[i].CfgId < entries[j].CfgId })
	return &pb.SkillListResp{SkillEntries: entries}, nil
}

// toSkillEntry 持有数据 → wire 条目（仅状态）。
func toSkillEntry(s model.SkillInfo) *pb.SkillEntry {
	return &pb.SkillEntry{
		CfgId:    s.SkillID,
		Level:    s.Level,
		Equipped: s.Equipped,
	}
}

// SkillAdd 新增技能：配置ID必须存在，已拥有返回错误。
func (this *skillLogic) SkillAdd(ctx context.Context, playerID int64, req *pb.SkillAddReq) (*pb.SkillAddResp, error) {
	if !tpl.T().Skill.Has(req.GetCfgId()) {
		return nil, errs.CodeInvalidParams.Err()
	}

	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return nil, err
	}

	s := model.SkillInfo{SkillID: req.GetCfgId()}
	if !row.AddSkill(s) {
		return nil, errs.CodeInvalidParams.WithMessage("技能已拥有").Err()
	}

	global.PlayerSkillCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleSkill, playerID, model.RecordActionGain, req.GetCfgId(), 1)
	return &pb.SkillAddResp{SkillEntry: toSkillEntry(s)}, nil
}

// SkillUpdate 更新技能：整体覆盖持有状态，未拥有返回错误。
func (this *skillLogic) SkillUpdate(ctx context.Context, playerID int64, req *pb.SkillUpdateReq) (*pb.SkillUpdateResp, error) {
	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return nil, err
	}

	s := model.SkillInfo{
		SkillID:  req.GetCfgId(),
		Level:    req.GetLevel(),
		Equipped: req.GetEquipped(),
	}
	if !row.UpdateSkill(s) {
		return nil, errs.CodeInvalidParams.WithMessage("技能未拥有").Err()
	}

	global.PlayerSkillCache.Set(playerID, row)
	return &pb.SkillUpdateResp{SkillEntry: toSkillEntry(s)}, nil
}

// SkillUpgrade 技能升级：星级档位沿用出战英雄，消耗配置表升级素材，等级 +1。
func (this *skillLogic) SkillUpgrade(ctx context.Context, playerID int64, req *pb.SkillUpgradeReq) (*pb.SkillUpgradeResp, error) {
	_, hero, err := equippedHero(ctx, playerID)
	if err != nil {
		return nil, err
	}

	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return nil, err
	}
	s, ok := row.Skills.Get(req.GetCfgId())
	if !ok {
		return nil, errs.CodeInvalidParams.WithMessage("技能未拥有").Err()
	}

	cfg := tpl.T().AttributeSkill.Get(attrID(clampTier(hero.Star), s.SkillID))
	if cfg == nil {
		return nil, errs.CodeInvalidParams.WithMessage("技能属性配置不存在").Err()
	}
	lo, hi := levelBounds(0, 0, false)
	if cfg.Level != nil {
		lo, hi = levelBounds(cfg.Level.X, cfg.Level.Y, true)
	}
	lvl := clampLevel(s.Level, lo, hi)
	if lvl >= hi {
		return nil, errs.CodeInvalidParams.WithMessage("技能已满级").Err()
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

	s.Level = lvl + 1
	if !row.UpdateSkill(s) {
		return nil, errs.CodeInvalidParams.WithMessage("技能未拥有").Err()
	}
	global.PlayerSkillCache.Set(playerID, row)

	_, attr, err := computeHeroAttribute(ctx, playerID, 0)
	if err != nil {
		return nil, err
	}
	return &pb.SkillUpgradeResp{SkillEntry: toSkillEntry(s), Attribute: attr}, nil
}

// EnterGame 进入游戏：加载玩家技能持有，未建档则初始化空集合。
func (this *skillLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if _, err := loadPlayerSkill(ctx, playerID); err != nil {
		return nil, err
	}
	// 技能数据由 SkillList 下发，这里只预热缓存
	return nil, nil
}

// LeaveGame 离开游戏：技能持有数据同步落库，并逐出内存缓存。
func (this *skillLogic) LeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	if err := flushAndEvict(ctx, global.PlayerSkillCache, playerID); err != nil {
		return nil, err
	}
	return &pb.LeaveGameResp{}, nil
}

// loadPlayerSkill 加载玩家技能持有；未建档则初始化空 map 并挂入缓存（异步落库）。
func loadPlayerSkill(ctx context.Context, playerID int64) (*model.PlayerSkill, error) {
	row, err := global.PlayerSkillCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = &model.PlayerSkill{PlayerID: playerID, Skills: util.NewSafeMap[int32, model.SkillInfo]()}
		global.PlayerSkillCache.Set(playerID, row)
		return row, nil
	}
	if err != nil {
		return nil, err
	}
	if row.Skills == nil {
		row.Skills = util.NewSafeMap[int32, model.SkillInfo]()
	}
	return row, nil
}

// ============ 资产接口实现（AssetHandler，唯一持有型） ============

// skillAssetHandler 技能资产实现：落在 player_skill。
type skillAssetHandler struct{}

// AssetType 资产类型：技能。
func (this *skillAssetHandler) AssetType() tpl.AssetType { return AssetTypeSkill }

// Add 发放技能；已拥有返回错误。
func (this *skillAssetHandler) Add(ctx context.Context, playerID int64, cfgID, _ int32) error {
	if !tpl.T().Skill.Has(cfgID) {
		return errs.CodeInvalidParams.WithMessage("技能配置不存在").Err()
	}

	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return err
	}
	if !row.AddSkill(model.SkillInfo{SkillID: cfgID}) {
		return errs.CodeInvalidParams.WithMessage("技能已拥有").Err()
	}

	global.PlayerSkillCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleSkill, playerID, model.RecordActionGain, cfgID, 1)
	return nil
}

// Remove 扣除技能；未拥有返回错误。
func (this *skillAssetHandler) Remove(ctx context.Context, playerID int64, cfgID, _ int32) error {
	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return err
	}
	if !row.Skills.Delete(cfgID) {
		return errs.CodeInvalidParams.WithMessage("技能未拥有").Err()
	}

	global.PlayerSkillCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleSkill, playerID, model.RecordActionConsume, cfgID, 1)
	return nil
}

// Update 覆盖持有：count<=0 移除，>0 确保持有。
func (this *skillAssetHandler) Update(ctx context.Context, playerID int64, cfgID, count int32) error {
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
func (this *skillAssetHandler) Query(ctx context.Context, playerID int64, cfgID int32) (int32, error) {
	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return 0, err
	}
	if _, ok := row.Skills.Get(cfgID); !ok {
		return 0, nil
	}
	return 1, nil
}

// List 列出全部技能（按配置ID升序）。
func (this *skillAssetHandler) List(ctx context.Context, playerID int64) ([]*tpl.Asset, error) {
	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return nil, err
	}

	assets := make([]*tpl.Asset, 0, row.Skills.Len())
	row.Skills.Range(func(cfgID int32, _ model.SkillInfo) bool {
		assets = append(assets, &tpl.Asset{Type: AssetTypeSkill, Id: cfgID, Count: 1})
		return true
	})
	sort.Slice(assets, func(i, j int) bool { return assets[i].Id < assets[j].Id })
	return assets, nil
}
