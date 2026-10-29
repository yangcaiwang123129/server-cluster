package logic

import (
	"context"
	"errors"
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

// init 装配期自注册实现。
func init() {
	skillLogicInst.Register(&skillLogic{})
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
	sort.Slice(entries, func(i, j int) bool { return entries[i].SkillId < entries[j].SkillId })
	return &pb.SkillListResp{SkillEntries: entries}, nil
}

// toSkillEntry 持有数据 → wire 条目（仅状态）。
func toSkillEntry(s model.SkillInfo) *pb.SkillEntry {
	return &pb.SkillEntry{
		SkillId:  s.SkillID,
		Level:    s.Level,
		Equipped: s.Equipped,
	}
}

// SkillAdd 新增技能：配置ID必须存在，已拥有返回错误。
func (this *skillLogic) SkillAdd(ctx context.Context, playerID int64, req *pb.SkillAddReq) (*pb.SkillAddResp, error) {
	if !tpl.T().Skill.Has(req.GetSkillId()) {
		return nil, errs.ErrInvalidParams
	}

	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return nil, err
	}

	s := model.SkillInfo{SkillID: req.GetSkillId()}
	if !row.AddSkill(s) {
		return nil, errs.WithMessage(errs.CodeInvalidParams, "技能已拥有")
	}

	global.PlayerSkillCache.Set(playerID, row)
	return &pb.SkillAddResp{SkillEntry: toSkillEntry(s)}, nil
}

// SkillUpdate 更新技能：整体覆盖持有状态，未拥有返回错误。
func (this *skillLogic) SkillUpdate(ctx context.Context, playerID int64, req *pb.SkillUpdateReq) (*pb.SkillUpdateResp, error) {
	row, err := loadPlayerSkill(ctx, playerID)
	if err != nil {
		return nil, err
	}

	s := model.SkillInfo{
		SkillID:  req.GetSkillId(),
		Level:    req.GetLevel(),
		Equipped: req.GetEquipped(),
	}
	if !row.UpdateSkill(s) {
		return nil, errs.WithMessage(errs.CodeInvalidParams, "技能未拥有")
	}

	global.PlayerSkillCache.Set(playerID, row)
	return &pb.SkillUpdateResp{SkillEntry: toSkillEntry(s)}, nil
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
