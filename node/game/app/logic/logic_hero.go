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

type heroLogic struct {
	once sync.Once
	resp *pb.HeroListResp
}

var heroLogicInst = singleton.NewContainer[global.HeroLogic]("HeroLogic")

// init 装配期自注册实现。
func init() {
	heroLogicInst.Register(&heroLogic{})
}

// Hero 获取英雄业务单例。
func Hero() global.HeroLogic { return heroLogicInst.MustGet() }

// HeroList 英雄列表（角色配置）：读内存配置表，首次请求转成 pb 后缓存复用。
func (this *heroLogic) HeroList(context.Context) (*pb.HeroListResp, error) {
	this.once.Do(func() {
		cfgs := tpl.T().RoleInfo.GetDataList()
		items := make([]*pb.HeroEntry, 0, len(cfgs))
		for _, cfg := range cfgs {
			items = append(items, &pb.HeroEntry{
				Id:                cfg.Id,
				Name:              cfg.Name,
				Job:               int32(cfg.Job),
				WeaponId:          cfg.WeaponId,
				ControlDifficulty: cfg.ControlDifficulty,
				CharacterShards:   cfg.CharacterShards,
				Rarity:            cfg.Rarity,
				CharacterId:       cfg.CharacterId,
				SkillId:           firstInt32(cfg.SkillID),
				GearId:            firstInt32(cfg.GearID),
			})
		}
		this.resp = &pb.HeroListResp{HeroEntries: items}
	})
	return this.resp, nil
}

// EnterGame 进入游戏：加载玩家英雄持有，未建档则初始化空集合。
func (this *heroLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if _, err := loadPlayerHero(ctx, playerID); err != nil {
		return nil, err
	}
	return &pb.EnterGameResp{}, nil
}

// LeaveGame 离开游戏：英雄持有数据同步落库，并逐出内存缓存。
func (this *heroLogic) LeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	if err := flushAndEvict(ctx, global.PlayerHeroCache, playerID); err != nil {
		return nil, err
	}
	return &pb.LeaveGameResp{}, nil
}

// loadPlayerHero 加载玩家英雄持有；未建档则初始化空 map 并挂入缓存（异步落库）。
func loadPlayerHero(ctx context.Context, playerID int64) (model.PlayerHero, error) {
	row, err := global.PlayerHeroCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = model.PlayerHero{PlayerID: playerID, Heroes: util.NewSafeMap[int32, model.HeroInfo]()}
		global.PlayerHeroCache.Set(playerID, row)
		return row, nil
	}
	if err != nil {
		return model.PlayerHero{}, err
	}
	// DB 中该列为 NULL 时字段为 nil，补成空集合避免业务写操作 panic
	if row.Heroes == nil {
		row.Heroes = util.NewSafeMap[int32, model.HeroInfo]()
	}
	return row, nil
}

// firstInt32 取多值配置的首个元素；配置为多档/多段，pb 当前为单值，空列表返回 0。
func firstInt32(values []int32) int32 {
	if len(values) == 0 {
		return 0
	}
	return values[0]
}
