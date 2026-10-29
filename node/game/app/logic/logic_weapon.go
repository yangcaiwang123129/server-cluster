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

type weaponLogic struct{}

var weaponLogicInst = singleton.NewContainer[global.WeaponLogic]("WeaponLogic")

// init 装配期自注册实现。
func init() {
	weaponLogicInst.Register(&weaponLogic{})
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
	sort.Slice(entries, func(i, j int) bool { return entries[i].WeaponId < entries[j].WeaponId })
	return &pb.WeaponListResp{WeaponEntries: entries}, nil
}

// toWeaponEntry 持有数据 → wire 条目（仅状态）。
func toWeaponEntry(w model.WeaponInfo) *pb.WeaponEntry {
	return &pb.WeaponEntry{
		WeaponId:   w.WeaponID,
		Level:      w.Level,
		Durability: w.Durability,
		Equipped:   w.Equipped,
	}
}

// WeaponAdd 新增武器：配置ID必须存在，已拥有返回错误。
func (this *weaponLogic) WeaponAdd(ctx context.Context, playerID int64, req *pb.WeaponAddReq) (*pb.WeaponAddResp, error) {
	if !tpl.T().Weapon.Has(req.GetWeaponId()) {
		return nil, errs.CodeInvalidParams.Err()
	}

	row, err := loadPlayerWeapon(ctx, playerID)
	if err != nil {
		return nil, err
	}

	w := model.WeaponInfo{WeaponID: req.GetWeaponId()}
	if !row.AddWeapon(w) {
		return nil, errs.CodeInvalidParams.WithMessage("武器已拥有").Err()
	}

	global.PlayerWeaponCache.Set(playerID, row)
	return &pb.WeaponAddResp{WeaponEntry: toWeaponEntry(w)}, nil
}

// WeaponUpdate 更新武器：整体覆盖持有状态，未拥有返回错误。
func (this *weaponLogic) WeaponUpdate(ctx context.Context, playerID int64, req *pb.WeaponUpdateReq) (*pb.WeaponUpdateResp, error) {
	row, err := loadPlayerWeapon(ctx, playerID)
	if err != nil {
		return nil, err
	}

	w := model.WeaponInfo{
		WeaponID:   req.GetWeaponId(),
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
			WeaponId:   w.WeaponID,
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
