package logic

import (
	"context"
	"errors"
	"sort"

	"gorm.io/gorm"

	"server-cluster/node/game/app/global"
	"server-cluster/share/consts"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

type gearLogic struct{}

var gearLogicInst = singleton.NewContainer[global.GearLogic]("GearLogic")

// init 装配期自注册实现。
func init() {
	gearLogicInst.Register(&gearLogic{})
}

// Gear 获取装备业务单例。
func Gear() global.GearLogic { return gearLogicInst.MustGet() }

// GearList 装备列表：只下发持有状态，静态属性由客户端按 gear_id 关联装备配置表。
func (this *gearLogic) GearList(ctx context.Context, playerID int64) (*pb.GearListResp, error) {
	row, err := loadPlayerGear(ctx, playerID)
	if err != nil {
		return nil, err
	}

	entries := make([]*pb.GearEntry, 0, row.Gears.Len())
	row.Gears.Range(func(_ int32, g model.GearInfo) bool {
		entries = append(entries, toGearEntry(g))
		return true
	})
	// map 遍历无序，按配置ID排序保证客户端展示稳定
	sort.Slice(entries, func(i, j int) bool { return entries[i].GearId < entries[j].GearId })
	return &pb.GearListResp{GearEntries: entries}, nil
}

// toGearEntry 持有数据 → wire 条目（仅状态）。
func toGearEntry(g model.GearInfo) *pb.GearEntry {
	return &pb.GearEntry{
		GearId:   g.GearID,
		Level:    g.Level,
		Equipped: g.Equipped,
	}
}

// GearAdd 新增装备：配置ID必须存在，已拥有返回错误。
func (this *gearLogic) GearAdd(ctx context.Context, playerID int64, req *pb.GearAddReq) (*pb.GearAddResp, error) {
	if !tpl.T().Gear.Has(req.GetGearId()) {
		return nil, errs.ErrInvalidParams
	}

	row, err := loadPlayerGear(ctx, playerID)
	if err != nil {
		return nil, err
	}

	g := model.GearInfo{GearID: req.GetGearId()}
	if !row.AddGear(g) {
		return nil, errs.WithMessage(consts.CodeInvalidParams, "装备已拥有")
	}

	global.PlayerGearCache.Set(playerID, row)
	return &pb.GearAddResp{GearEntry: toGearEntry(g)}, nil
}

// GearUpdate 更新装备：整体覆盖持有状态，未拥有返回错误。
func (this *gearLogic) GearUpdate(ctx context.Context, playerID int64, req *pb.GearUpdateReq) (*pb.GearUpdateResp, error) {
	row, err := loadPlayerGear(ctx, playerID)
	if err != nil {
		return nil, err
	}

	g := model.GearInfo{
		GearID:   req.GetGearId(),
		Level:    req.GetLevel(),
		Equipped: req.GetEquipped(),
	}
	if !row.UpdateGear(g) {
		return nil, errs.WithMessage(consts.CodeInvalidParams, "装备未拥有")
	}

	global.PlayerGearCache.Set(playerID, row)
	return &pb.GearUpdateResp{GearEntry: toGearEntry(g)}, nil
}

// EnterGame 进入游戏：加载玩家装备持有，未建档则初始化空集合。
func (this *gearLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if _, err := loadPlayerGear(ctx, playerID); err != nil {
		return nil, err
	}
	return &pb.EnterGameResp{}, nil
}

// LeaveGame 离开游戏：装备持有数据同步落库，并逐出内存缓存。
func (this *gearLogic) LeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	if err := flushAndEvict(ctx, global.PlayerGearCache, playerID); err != nil {
		return nil, err
	}
	return &pb.LeaveGameResp{}, nil
}

// loadPlayerGear 加载玩家装备持有；未建档则初始化空 map 并挂入缓存（异步落库）。
func loadPlayerGear(ctx context.Context, playerID int64) (model.PlayerGear, error) {
	row, err := global.PlayerGearCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = model.PlayerGear{PlayerID: playerID, Gears: util.NewSafeMap[int32, model.GearInfo]()}
		global.PlayerGearCache.Set(playerID, row)
		return row, nil
	}
	if err != nil {
		return model.PlayerGear{}, err
	}
	if row.Gears == nil {
		row.Gears = util.NewSafeMap[int32, model.GearInfo]()
	}
	return row, nil
}
