package logic

import (
	"context"
	"errors"
	"sync"

	"gorm.io/gorm"

	"server-cluster/node/game/app/global"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

type skillLogic struct {
	once sync.Once
	resp *pb.SkillListResp
}

var skillLogicInst = singleton.NewContainer[global.SkillLogic]("SkillLogic")

// init 装配期自注册实现。
func init() {
	skillLogicInst.Register(&skillLogic{})
}

// Skill 获取技能业务单例。
func Skill() global.SkillLogic { return skillLogicInst.MustGet() }

// SkillList 技能列表：读内存配置表，首次请求转成 pb 后缓存复用。
func (this *skillLogic) SkillList(context.Context) (*pb.SkillListResp, error) {
	this.once.Do(func() {
		cfgs := tpl.T().Skill.GetDataList()
		items := make([]*pb.SkillItem, 0, len(cfgs))
		for _, cfg := range cfgs {
			item := &pb.SkillItem{
				Id:               cfg.Id,
				SkillTemplate:    cfg.SkillTemplate,
				SkillName:        cfg.SkillName,
				Desc:             cfg.Desc,
				DescDetail:       cfg.DescDetail,
				Damage:           cfg.Damage,
				AutoFireDistance: cfg.AutoFireDistance,
				Capacity:         cfg.Capacity,
			}
			if cfg.AutoFireAngle != nil {
				item.AutoFireAngleX = cfg.AutoFireAngle.X
				item.AutoFireAngleY = cfg.AutoFireAngle.Y
			}
			items = append(items, item)
		}
		this.resp = &pb.SkillListResp{Items: items}
	})
	return this.resp, nil
}

// EnterGame 进入游戏：加载玩家技能持有，未建档则初始化空集合。
func (this *skillLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if _, err := loadPlayerSkill(ctx, playerID); err != nil {
		return nil, err
	}
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
func loadPlayerSkill(ctx context.Context, playerID int64) (model.PlayerSkill, error) {
	row, err := global.PlayerSkillCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = model.PlayerSkill{PlayerID: playerID, Skills: util.NewSafeMap[int32, model.SkillInfo]()}
		global.PlayerSkillCache.Set(playerID, row)
		return row, nil
	}
	if err != nil {
		return model.PlayerSkill{}, err
	}
	if row.Skills == nil {
		row.Skills = util.NewSafeMap[int32, model.SkillInfo]()
	}
	return row, nil
}
