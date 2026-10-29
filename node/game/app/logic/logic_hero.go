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

type heroLogic struct{}

var heroLogicInst = singleton.NewContainer[global.HeroLogic]("HeroLogic")

// init 装配期自注册实现。
func init() {
	heroLogicInst.Register(&heroLogic{})
}

// Hero 获取英雄业务单例。
func Hero() global.HeroLogic { return heroLogicInst.MustGet() }

// HeroList 英雄列表：只下发持有状态，静态属性由客户端按 hero_id 关联英雄配置表。
func (this *heroLogic) HeroList(ctx context.Context, playerID int64) (*pb.HeroListResp, error) {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return nil, err
	}

	entries := make([]*pb.HeroEntry, 0, row.Heroes.Len())
	row.Heroes.Range(func(_ int32, h model.HeroInfo) bool {
		entries = append(entries, toHeroEntry(h))
		return true
	})
	// map 遍历无序，按配置ID排序保证客户端展示稳定
	sort.Slice(entries, func(i, j int) bool { return entries[i].HeroId < entries[j].HeroId })
	return &pb.HeroListResp{HeroEntries: entries}, nil
}

// toHeroEntry 持有数据 → wire 条目（仅状态）。
func toHeroEntry(h model.HeroInfo) *pb.HeroEntry {
	return &pb.HeroEntry{
		HeroId:   h.HeroID,
		Level:    h.Level,
		Exp:      h.Exp,
		Star:     h.Star,
		SkinId:   h.SkinID,
		Equipped: h.Equipped,
	}
}

// HeroAdd 新增英雄：配置ID必须存在，已拥有返回错误。
func (this *heroLogic) HeroAdd(ctx context.Context, playerID int64, req *pb.HeroAddReq) (*pb.HeroAddResp, error) {
	if !tpl.T().RoleInfo.Has(req.GetHeroId()) {
		return nil, errs.CodeInvalidParams.Err()
	}

	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return nil, err
	}

	h := model.HeroInfo{HeroID: req.GetHeroId()}
	if !row.AddHero(h) {
		return nil, errs.CodeInvalidParams.WithMessage("英雄已拥有").Err()
	}

	global.PlayerHeroCache.Set(playerID, row)
	return &pb.HeroAddResp{HeroEntry: toHeroEntry(h)}, nil
}

// HeroUpdate 更新英雄：整体覆盖持有状态，未拥有返回错误。
func (this *heroLogic) HeroUpdate(ctx context.Context, playerID int64, req *pb.HeroUpdateReq) (*pb.HeroUpdateResp, error) {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return nil, err
	}

	h := model.HeroInfo{
		HeroID:   req.GetHeroId(),
		Level:    req.GetLevel(),
		Exp:      req.GetExp(),
		Star:     req.GetStar(),
		SkinID:   req.GetSkinId(),
		Equipped: req.GetEquipped(),
	}
	if !row.UpdateHero(h) {
		return nil, errs.CodeInvalidParams.WithMessage("英雄未拥有").Err()
	}

	global.PlayerHeroCache.Set(playerID, row)
	return &pb.HeroUpdateResp{HeroEntry: toHeroEntry(h)}, nil
}

// EnterGame 进入游戏：加载玩家英雄持有，未建档则初始化空集合。
func (this *heroLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return nil, err
	}
	// 只下发当前出战英雄的状态，静态属性由客户端关联英雄配置表
	return &pb.EnterGameResp{HeroEntry: equippedHeroEntry(row)}, nil
}

// equippedHeroEntry 当前出战英雄；未出战返回 nil。
func equippedHeroEntry(row *model.PlayerHero) *pb.HeroEntry {
	if row.Heroes == nil {
		return nil
	}
	var entry *pb.HeroEntry
	row.Heroes.Range(func(_ int32, h model.HeroInfo) bool {
		if !h.Equipped {
			return true
		}
		entry = &pb.HeroEntry{
			HeroId:   h.HeroID,
			Level:    h.Level,
			Exp:      h.Exp,
			Star:     h.Star,
			SkinId:   h.SkinID,
			Equipped: true,
		}
		return false
	})
	return entry
}

// LeaveGame 离开游戏：英雄持有数据同步落库，并逐出内存缓存。
func (this *heroLogic) LeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	if err := flushAndEvict(ctx, global.PlayerHeroCache, playerID); err != nil {
		return nil, err
	}
	return &pb.LeaveGameResp{}, nil
}

// loadPlayerHero 加载玩家英雄持有；未建档则初始化空 map 并挂入缓存（异步落库）。
func loadPlayerHero(ctx context.Context, playerID int64) (*model.PlayerHero, error) {
	row, err := global.PlayerHeroCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = &model.PlayerHero{PlayerID: playerID, Heroes: util.NewSafeMap[int32, model.HeroInfo]()}
		global.PlayerHeroCache.Set(playerID, row)
		return row, nil
	}
	if err != nil {
		return nil, err
	}
	// DB 中该列为 NULL 时字段为 nil，补成空集合避免业务写操作 panic
	if row.Heroes == nil {
		row.Heroes = util.NewSafeMap[int32, model.HeroInfo]()
	}
	return row, nil
}
