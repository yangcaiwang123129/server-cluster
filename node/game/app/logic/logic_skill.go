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
	registerAsset(&skillAssetHandler{})
}

// Skill 获取技能业务单例。
func Skill() global.SkillLogic { return skillLogicInst.MustGet() }

// toSkillEntry 持有数据 → wire 条目（仅状态）。
func toSkillEntry(s model.SkillInfo) *pb.SkillEntry {
	return &pb.SkillEntry{
		CfgId:        s.SkillID,
		Level:        s.Level,
		Quality:      s.Quality,
		Breakthrough: s.Breakthrough,
	}
}

// newSkillInfo 新获得技能的默认状态：等级 1、品质 1、突破 0。
func newSkillInfo(cfgID int32) model.SkillInfo {
	return model.SkillInfo{SkillID: cfgID, Level: 1, Quality: 1}
}

// SkillList 技能持有列表（供 HeroList 聚合下发与 GM 查询）。
func (this *skillLogic) SkillList(ctx context.Context, playerID int64) ([]*pb.SkillEntry, error) {
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
	return entries, nil
}

// SkillUpdate 更新技能：整体覆盖持有状态，未拥有返回错误。
func (this *skillLogic) SkillUpdate(ctx context.Context, playerID int64, req *pb.SkillUpdateReq) (*pb.SkillUpdateResp, error) {
	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return nil, err
	}

	s := model.SkillInfo{
		SkillID:      req.GetCfgId(),
		Level:        clampDim(req.GetLevel(), 1),
		Quality:      clampDim(req.GetQuality(), 1),
		Breakthrough: clampDim(req.GetBreakthrough(), 0),
	}
	if !row.UpdateSkill(s) {
		return nil, errs.CodeInvalidParams.WithMessage("技能未拥有").Err()
	}

	global.PlayerSkillCache.Set(playerID, row)
	notifyHeroBySkill(ctx, playerID, s.SkillID)
	return &pb.SkillUpdateResp{SkillEntry: toSkillEntry(s)}, nil
}

// skillOwned 取玩家指定技能持有与其绑定英雄ID。
func skillOwned(ctx context.Context, playerID int64, cfgID int32) (*model.PlayerSkill, model.SkillInfo, int32, error) {
	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return nil, model.SkillInfo{}, 0, err
	}
	s, ok := row.Skills.Get(cfgID)
	if !ok {
		return nil, model.SkillInfo{}, 0, errs.CodeInvalidParams.WithMessage("技能未拥有").Err()
	}
	heroID, ok := heroIDBySkill(cfgID)
	if !ok {
		return nil, model.SkillInfo{}, 0, errs.CodeInvalidParams.WithMessage("技能未绑定英雄").Err()
	}
	return row, s, heroID, nil
}

// skillDimGrow 维度提升通用流程：校验持有 → 取当前维度行得消耗 → 校验下一维度行存在 → 扣素材 → 维度+1。
func skillDimGrow(ctx context.Context, playerID int64, cfgID int32,
	cur func(s model.SkillInfo) int32,
	set func(s *model.SkillInfo, v int32),
	rowOf func(dim int32) (material, use int32, ok bool),
	fullMsg string,
) (model.SkillInfo, int32, error) {
	row, s, heroID, err := skillOwned(ctx, playerID, cfgID)
	if err != nil {
		return model.SkillInfo{}, 0, err
	}
	dim := cur(s)
	material, use, ok := rowOf(dim)
	if !ok {
		return model.SkillInfo{}, 0, errs.CodeInvalidParams.WithMessage("技能维度配置不存在").Err()
	}
	_, _, hasNext := rowOf(dim + 1)
	if err := dimConsume(ctx, playerID, material, use, hasNext, fullMsg); err != nil {
		return model.SkillInfo{}, 0, err
	}
	set(&s, dim+1)
	if !row.UpdateSkill(s) {
		return model.SkillInfo{}, 0, errs.CodeInvalidParams.WithMessage("技能未拥有").Err()
	}
	global.PlayerSkillCache.Set(playerID, row)
	notifyHero(ctx, playerID, heroID)
	return s, heroID, nil
}

// SkillUpgrade 技能升级：消耗 SkillLevel 表当前等级的升级素材，等级 +1。
func (this *skillLogic) SkillUpgrade(ctx context.Context, playerID int64, req *pb.SkillUpgradeReq) (*pb.SkillUpgradeResp, error) {
	s, heroID, err := skillDimGrow(ctx, playerID, req.GetCfgId(),
		func(s model.SkillInfo) int32 { return s.Level },
		func(s *model.SkillInfo, v int32) { s.Level = v },
		func(dim int32) (int32, int32, bool) {
			c := tpl.T().SkillLevel.Get(dimID(req.GetCfgId(), dim))
			if c == nil {
				return 0, 0, false
			}
			return c.UpgradeMaterial, c.UpgradeUse, true
		},
		"技能已满级")
	if err != nil {
		return nil, err
	}
	record.Record(ctx, model.RecordModuleSkill, playerID, model.RecordActionConsume, req.GetCfgId(), 1)

	_, attr, err := computeHeroAttribute(ctx, playerID, heroID)
	if err != nil {
		return nil, err
	}
	return &pb.SkillUpgradeResp{SkillEntry: toSkillEntry(s), Attribute: attr}, nil
}

// SkillUpQuality 技能升品：消耗 SkillQuality 表当前品质的升品素材，品质 +1。
func (this *skillLogic) SkillUpQuality(ctx context.Context, playerID int64, req *pb.SkillUpQualityReq) (*pb.SkillUpQualityResp, error) {
	s, heroID, err := skillDimGrow(ctx, playerID, req.GetCfgId(),
		func(s model.SkillInfo) int32 { return s.Quality },
		func(s *model.SkillInfo, v int32) { s.Quality = v },
		func(dim int32) (int32, int32, bool) {
			c := tpl.T().SkillQuality.Get(dimID(req.GetCfgId(), dim))
			if c == nil {
				return 0, 0, false
			}
			return c.QualityMaterial, c.QualityUse, true
		},
		"技能已满品质")
	if err != nil {
		return nil, err
	}
	record.Record(ctx, model.RecordModuleSkill, playerID, model.RecordActionConsume, req.GetCfgId(), 1)

	_, attr, err := computeHeroAttribute(ctx, playerID, heroID)
	if err != nil {
		return nil, err
	}
	return &pb.SkillUpQualityResp{SkillEntry: toSkillEntry(s), Attribute: attr}, nil
}

// SkillBreakthrough 技能突破：消耗 SkillBreakthrough 表当前突破的素材，突破 +1。
func (this *skillLogic) SkillBreakthrough(ctx context.Context, playerID int64, req *pb.SkillBreakthroughReq) (*pb.SkillBreakthroughResp, error) {
	s, heroID, err := skillDimGrow(ctx, playerID, req.GetCfgId(),
		func(s model.SkillInfo) int32 { return s.Breakthrough },
		func(s *model.SkillInfo, v int32) { s.Breakthrough = v },
		func(dim int32) (int32, int32, bool) {
			c := tpl.T().SkillBreakthrough.Get(dimID(req.GetCfgId(), dim))
			if c == nil {
				return 0, 0, false
			}
			return c.BreakthroughMaterial, c.BreakthroughUse, true
		},
		"技能已满突破")
	if err != nil {
		return nil, err
	}
	record.Record(ctx, model.RecordModuleSkill, playerID, model.RecordActionConsume, req.GetCfgId(), 1)

	_, attr, err := computeHeroAttribute(ctx, playerID, heroID)
	if err != nil {
		return nil, err
	}
	return &pb.SkillBreakthroughResp{SkillEntry: toSkillEntry(s), Attribute: attr}, nil
}

// SkillEnsureForHero 按英雄配置初始化固定绑定技能：未持有则补建默认状态。
func (this *skillLogic) SkillEnsureForHero(ctx context.Context, playerID int64, heroCfgID int32) error {
	cfg := tpl.T().RoleInfo.Get(heroCfgID)
	if cfg == nil {
		return nil
	}
	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return err
	}
	changed := false
	for _, skillID := range cfg.SkillID {
		if skillID <= 0 || row.Skills.Has(skillID) {
			continue
		}
		if row.AddSkill(newSkillInfo(skillID)) {
			changed = true
		}
	}
	if changed {
		global.PlayerSkillCache.Set(playerID, row)
	}
	return nil
}

// EnterGame 进入游戏：加载玩家技能持有，未建档则初始化空集合；并按已拥有英雄兜底补建绑定技能。
func (this *skillLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if _, err := loadPlayerSkill(ctx, playerID); err != nil {
		return nil, err
	}
	heroes, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if heroes.Heroes != nil {
		var ids []int32
		heroes.Heroes.Range(func(id int32, _ model.HeroInfo) bool {
			ids = append(ids, id)
			return true
		})
		for _, id := range ids {
			if err := this.SkillEnsureForHero(ctx, playerID, id); err != nil {
				return nil, err
			}
		}
	}
	// 技能数据由 SkillList 下发，这里只预热缓存
	return &pb.EnterGameResp{}, nil
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

// ============ 资产接口实现（AssetHandler，唯一持有型：key 为技能配置ID） ============

// skillAssetHandler 技能资产实现：落在 player_skill。技能由英雄配置绑定，发放幂等。
type skillAssetHandler struct{}

// AssetType 资产类型：技能。
func (this *skillAssetHandler) AssetType() tpl.AssetType { return tpl.AssetType_Skill }

// Add 发放技能：已拥有视为成功（技能随英雄自动补建，避免奖励重复发放失败）。
func (this *skillAssetHandler) Add(ctx context.Context, playerID int64, cfgID, _ int32) error {
	if !tpl.T().Skill.Has(cfgID) {
		return errs.CodeInvalidParams.WithMessage("技能配置不存在").Err()
	}
	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return err
	}
	if row.Skills.Has(cfgID) {
		return nil
	}
	if !row.AddSkill(newSkillInfo(cfgID)) {
		return errs.CodeInvalidParams.WithMessage("技能已拥有").Err()
	}

	global.PlayerSkillCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleSkill, playerID, model.RecordActionGain, cfgID, 1)
	notifyHeroBySkill(ctx, playerID, cfgID)
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
	notifyHeroBySkill(ctx, playerID, cfgID)
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
	if row.Skills.Has(cfgID) {
		return 1, nil
	}
	return 0, nil
}

// List 列出全部技能（按配置ID升序）。
func (this *skillAssetHandler) List(ctx context.Context, playerID int64) ([]*tpl.Asset, error) {
	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return nil, err
	}

	assets := make([]*tpl.Asset, 0, row.Skills.Len())
	row.Skills.Range(func(cfgID int32, _ model.SkillInfo) bool {
		assets = append(assets, &tpl.Asset{Type: tpl.AssetType_Skill, Id: cfgID, Count: 1})
		return true
	})
	sort.Slice(assets, func(i, j int) bool { return assets[i].Id < assets[j].Id })
	return assets, nil
}
