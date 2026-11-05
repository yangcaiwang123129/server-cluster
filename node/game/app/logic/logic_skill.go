package logic

import (
	"context"
	"sort"

	"server-cluster/internal/cache"
	"server-cluster/internal/record"
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

type skillLogic struct {
	// cache 玩家技能持有（player_skill，一玩家一行，技能 map JSON）；读走 LRU，写异步批量 upsert。
	cache *cache.Cache[int64, *model.PlayerSkill]
}

var skillLogicInst = singleton.NewContainer[global.SkillLogic]("SkillLogic")

func init() {
	c := cache.NewPlayerTableCache[model.PlayerSkill](cache.CacherConfig{Name: enum.TablePlayerSkill.String()})
	global.RegisterCacheCloser(c)
	skillLogicInst.Register(&skillLogic{cache: c})
	registerAsset(&skillAssetStore{})
}

// Skill 获取技能业务单例。
func Skill() global.SkillLogic { return skillLogicInst.MustGet() }

// newSkillInfo 新获得技能的默认状态：等级 1、品质 1、突破 0。
func newSkillInfo(cfgID int32) model.SkillInfo {
	return model.SkillInfo{SkillID: cfgID, Level: 1, Quality: 1}
}

func (this *skillLogic) SkillList(ctx context.Context, playerID int64) ([]*pb.SkillEntry, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}

	entries := make([]*pb.SkillEntry, 0, row.Skills.Len())
	row.Skills.Range(func(_ int32, s model.SkillInfo) bool {
		entries = append(entries, convert.SkillEntry(s))
		return true
	})
	// map 遍历无序，按配置ID排序保证客户端展示稳定
	sort.Slice(entries, func(i, j int) bool { return entries[i].CfgId < entries[j].CfgId })
	return entries, nil
}

func (this *skillLogic) SkillUpdate(ctx context.Context, playerID int64, req *pb.SkillUpdateReq) (*pb.SkillUpdateResp, error) {
	row, err := this.Load(ctx, playerID)
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
		return &pb.SkillUpdateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "技能未拥有")}, nil
	}

	this.Save(playerID, row)
	notifyHeroBySkill(ctx, playerID, s.SkillID)
	return &pb.SkillUpdateResp{SkillEntry: convert.SkillEntry(s)}, nil
}

// skillOwned 取玩家指定技能持有与其绑定英雄ID。
func skillOwned(ctx context.Context, playerID int64, cfgID int32) (*model.PlayerSkill, model.SkillInfo, int32, error) {
	row, err := Skill().Load(ctx, playerID)
	if err != nil {
		return nil, model.SkillInfo{}, 0, err
	}
	s, ok := row.Skills.Get(cfgID)
	if !ok {
		return nil, model.SkillInfo{}, 0, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "技能未拥有")
	}
	heroID, ok := heroIDBySkill(cfgID)
	if !ok {
		return nil, model.SkillInfo{}, 0, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "技能未绑定英雄")
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
		return model.SkillInfo{}, 0, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "技能维度配置不存在")
	}
	_, _, hasNext := rowOf(dim + 1)
	if err := dimConsume(ctx, playerID, material, use, hasNext, fullMsg); err != nil {
		return model.SkillInfo{}, 0, err
	}
	set(&s, dim+1)
	if !row.UpdateSkill(s) {
		return model.SkillInfo{}, 0, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "技能未拥有")
	}
	Skill().(*skillLogic).cache.Set(playerID, row)
	notifyHeroByID(ctx, playerID, heroID)
	return s, heroID, nil
}

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
		if r := codedResult(err); r != nil {
			return &pb.SkillUpgradeResp{Result: r}, nil
		}
		return nil, err
	}
	record.Record(ctx, model.RecordModuleSkill, playerID, model.RecordActionConsume, req.GetCfgId(), 1)

	_, attr, err := computeHeroAttribute(ctx, playerID, heroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.SkillUpgradeResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.SkillUpgradeResp{SkillEntry: convert.SkillEntry(s), Attribute: attr}, nil
}

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
		if r := codedResult(err); r != nil {
			return &pb.SkillUpQualityResp{Result: r}, nil
		}
		return nil, err
	}
	record.Record(ctx, model.RecordModuleSkill, playerID, model.RecordActionConsume, req.GetCfgId(), 1)

	_, attr, err := computeHeroAttribute(ctx, playerID, heroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.SkillUpQualityResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.SkillUpQualityResp{SkillEntry: convert.SkillEntry(s), Attribute: attr}, nil
}

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
		if r := codedResult(err); r != nil {
			return &pb.SkillBreakthroughResp{Result: r}, nil
		}
		return nil, err
	}
	record.Record(ctx, model.RecordModuleSkill, playerID, model.RecordActionConsume, req.GetCfgId(), 1)

	_, attr, err := computeHeroAttribute(ctx, playerID, heroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.SkillBreakthroughResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.SkillBreakthroughResp{SkillEntry: convert.SkillEntry(s), Attribute: attr}, nil
}

func (this *skillLogic) SkillEnsureForHero(ctx context.Context, playerID int64, heroCfgID int32) error {
	cfg := tpl.T().RoleInfo.Get(heroCfgID)
	if cfg == nil {
		return nil
	}
	row, err := this.Load(ctx, playerID)
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
		this.Save(playerID, row)
	}
	return nil
}

func (this *skillLogic) OnEnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq, _ *pb.EnterGameResp) error {
	if _, err := this.Load(ctx, playerID); err != nil {
		return err
	}
	heroes, err := Hero().Load(ctx, playerID)
	if err != nil {
		return err
	}
	if heroes.Heroes != nil {
		var ids []int32
		heroes.Heroes.Range(func(id int32, _ model.HeroInfo) bool {
			ids = append(ids, id)
			return true
		})
		for _, id := range ids {
			if err := this.SkillEnsureForHero(ctx, playerID, id); err != nil {
				return err
			}
		}
	}
	// 技能数据由 SkillList 下发，这里只预热缓存
	return nil
}

func (this *skillLogic) OnLeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq, _ *pb.LeaveGameResp) error {
	if err := flushAndEvict(ctx, this.cache, playerID); err != nil {
		return err
	}
	return nil
}

func (this *skillLogic) Load(ctx context.Context, playerID int64) (*model.PlayerSkill, error) {
	return this.cache.GetOrInit(ctx, playerID,
		func(id int64) *model.PlayerSkill {
			return &model.PlayerSkill{PlayerID: id, Skills: util.NewSafeMap[int32, model.SkillInfo]()}
		},
		func(row *model.PlayerSkill) {
			if row.Skills == nil {
				row.Skills = util.NewSafeMap[int32, model.SkillInfo]()
			}
		})
}

func (this *skillLogic) Save(playerID int64, row *model.PlayerSkill) {
	this.cache.Set(playerID, row)
}

func (this *skillLogic) Flush(ctx context.Context, playerID int64) error {
	return flushAndEvict(ctx, this.cache, playerID)
}

// ============ 资产接口实现（AssetStore，唯一持有型：key 为技能配置ID） ============

// skillAssetStore 技能资产实现：落在 player_skill。技能由英雄配置绑定，发放幂等。
type skillAssetStore struct{}

func (this *skillAssetStore) AssetType() tpl.AssetType { return tpl.AssetType_Skill }

func (this *skillAssetStore) Add(ctx context.Context, playerID int64, cfgID, _ int32) error {
	if !tpl.T().Skill.Has(cfgID) {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "技能配置不存在")
	}
	row, err := Skill().Load(ctx, playerID)
	if err != nil {
		return err
	}
	if row.Skills.Has(cfgID) {
		return nil
	}
	if !row.AddSkill(newSkillInfo(cfgID)) {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "技能已拥有")
	}

	Skill().(*skillLogic).cache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleSkill, playerID, model.RecordActionGain, cfgID, 1)
	notifyHeroBySkill(ctx, playerID, cfgID)
	return nil
}

func (this *skillAssetStore) Remove(ctx context.Context, playerID int64, cfgID, _ int32) error {
	row, err := Skill().Load(ctx, playerID)
	if err != nil {
		return err
	}
	if !row.Skills.Delete(cfgID) {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "技能未拥有")
	}

	Skill().(*skillLogic).cache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleSkill, playerID, model.RecordActionConsume, cfgID, 1)
	notifyHeroBySkill(ctx, playerID, cfgID)
	return nil
}

func (this *skillAssetStore) Update(ctx context.Context, playerID int64, cfgID, count int32) error {
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

func (this *skillAssetStore) Query(ctx context.Context, playerID int64, cfgID int32) (int32, error) {
	row, err := Skill().Load(ctx, playerID)
	if err != nil {
		return 0, err
	}
	if row.Skills.Has(cfgID) {
		return 1, nil
	}
	return 0, nil
}

func (this *skillAssetStore) List(ctx context.Context, playerID int64) ([]*tpl.Asset, error) {
	row, err := Skill().Load(ctx, playerID)
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
