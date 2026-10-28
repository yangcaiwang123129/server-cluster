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

type weaponLogic struct {
	once sync.Once
	resp *pb.WeaponListResp
}

var weaponLogicInst = singleton.NewContainer[global.WeaponLogic]("WeaponLogic")

// init 装配期自注册实现。
func init() {
	weaponLogicInst.Register(&weaponLogic{})
}

// Weapon 获取武器业务单例。
func Weapon() global.WeaponLogic { return weaponLogicInst.MustGet() }

// WeaponList 武器列表：读内存配置表，首次请求转成 pb 后缓存复用。
func (this *weaponLogic) WeaponList(context.Context) (*pb.WeaponListResp, error) {
	this.once.Do(func() {
		cfgs := tpl.T().Weapon.GetDataList()
		items := make([]*pb.WeaponEntry, 0, len(cfgs))
		for _, cfg := range cfgs {
			items = append(items, &pb.WeaponEntry{
				Id:             cfg.Id,
				WeaponName:     cfg.WeaponName,
				Icon:           cfg.Icon,
				WeaponDamage:   firstInt32(cfg.WeaponDamage),
				PenetrateLevel: firstInt32(cfg.PenetrateLevel),
				FireRate:       firstInt32(cfg.FireRate),
				ReloadTime:     firstInt32(cfg.ReloadTime),
				FieldAttack:    firstInt32(cfg.FieldAttack),
				ScatterAttack:  firstInt32(cfg.ScatterAttack),
				ScatterMove:    firstInt32(cfg.ScatterMove),
				ScatterAim:     firstInt32(cfg.ScatterAim),
				TimeAim:        firstInt32(cfg.TimeAim),
			})
		}
		this.resp = &pb.WeaponListResp{WeaponEntries: items}
	})
	return this.resp, nil
}

// EnterGame 进入游戏：加载玩家武器持有，未建档则初始化空集合。
func (this *weaponLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if _, err := loadPlayerWeapon(ctx, playerID); err != nil {
		return nil, err
	}
	return &pb.EnterGameResp{}, nil
}

// LeaveGame 离开游戏：武器持有数据同步落库，并逐出内存缓存。
func (this *weaponLogic) LeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	if err := flushAndEvict(ctx, global.PlayerWeaponCache, playerID); err != nil {
		return nil, err
	}
	return &pb.LeaveGameResp{}, nil
}

// loadPlayerWeapon 加载玩家武器持有；未建档则初始化空 map 并挂入缓存（异步落库）。
func loadPlayerWeapon(ctx context.Context, playerID int64) (model.PlayerWeapon, error) {
	row, err := global.PlayerWeaponCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = model.PlayerWeapon{PlayerID: playerID, Weapons: util.NewSafeMap[int32, model.WeaponInfo]()}
		global.PlayerWeaponCache.Set(playerID, row)
		return row, nil
	}
	if err != nil {
		return model.PlayerWeapon{}, err
	}
	if row.Weapons == nil {
		row.Weapons = util.NewSafeMap[int32, model.WeaponInfo]()
	}
	return row, nil
}
