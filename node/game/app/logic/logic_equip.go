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

type equipLogic struct{}

var equipLogicInst = singleton.NewContainer[global.EquipLogic]("EquipLogic")

// init 装配期自注册实现。
func init() {
	equipLogicInst.Register(&equipLogic{})
}

// Equip 获取装备业务单例。
func Equip() global.EquipLogic { return equipLogicInst.MustGet() }

// EquipList 装备列表：只下发持有状态，静态属性由客户端按 equip_id 关联装备配置表。
func (this *equipLogic) EquipList(ctx context.Context, playerID int64) (*pb.EquipListResp, error) {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}

	entries := make([]*pb.EquipEntry, 0, row.Equips.Len())
	row.Equips.Range(func(_ int32, g model.EquipInfo) bool {
		entries = append(entries, toEquipEntry(g))
		return true
	})
	// map 遍历无序，按配置ID排序保证客户端展示稳定
	sort.Slice(entries, func(i, j int) bool { return entries[i].CfgId < entries[j].CfgId })
	return &pb.EquipListResp{EquipEntries: entries}, nil
}

// toEquipEntry 持有数据 → wire 条目（仅状态）。
func toEquipEntry(g model.EquipInfo) *pb.EquipEntry {
	return &pb.EquipEntry{
		CfgId:    g.EquipID,
		Level:    g.Level,
		Equipped: g.Equipped,
	}
}

// EquipAdd 新增装备：配置ID必须存在，已拥有返回错误。
func (this *equipLogic) EquipAdd(ctx context.Context, playerID int64, req *pb.EquipAddReq) (*pb.EquipAddResp, error) {
	if !tpl.T().Equip.Has(req.GetCfgId()) {
		return nil, errs.CodeInvalidParams.Err()
	}

	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}

	g := model.EquipInfo{EquipID: req.GetCfgId()}
	if !row.AddEquip(g) {
		return nil, errs.CodeInvalidParams.WithMessage("装备已拥有").Err()
	}

	global.PlayerEquipCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleEquip, playerID, model.RecordActionGain, req.GetCfgId(), 1)
	return &pb.EquipAddResp{EquipEntry: toEquipEntry(g)}, nil
}

// EquipUpdate 更新装备：整体覆盖持有状态，未拥有返回错误。
func (this *equipLogic) EquipUpdate(ctx context.Context, playerID int64, req *pb.EquipUpdateReq) (*pb.EquipUpdateResp, error) {
	row, err := loadPlayerEquip(ctx, playerID)
	if err != nil {
		return nil, err
	}

	g := model.EquipInfo{
		EquipID:  req.GetCfgId(),
		Level:    req.GetLevel(),
		Equipped: req.GetEquipped(),
	}
	if !row.UpdateEquip(g) {
		return nil, errs.CodeInvalidParams.WithMessage("装备未拥有").Err()
	}

	global.PlayerEquipCache.Set(playerID, row)
	return &pb.EquipUpdateResp{EquipEntry: toEquipEntry(g)}, nil
}

// EnterGame 进入游戏：加载玩家装备持有，未建档则初始化空集合。
func (this *equipLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if _, err := loadPlayerEquip(ctx, playerID); err != nil {
		return nil, err
	}
	// 装备数据由 EquipList 下发，这里只预热缓存
	return nil, nil
}

// LeaveGame 离开游戏：装备持有数据同步落库，并逐出内存缓存。
func (this *equipLogic) LeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	if err := flushAndEvict(ctx, global.PlayerEquipCache, playerID); err != nil {
		return nil, err
	}
	return &pb.LeaveGameResp{}, nil
}

// loadPlayerEquip 加载玩家装备持有；未建档则初始化空 map 并挂入缓存（异步落库）。
func loadPlayerEquip(ctx context.Context, playerID int64) (*model.PlayerEquip, error) {
	row, err := global.PlayerEquipCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = &model.PlayerEquip{PlayerID: playerID, Equips: util.NewSafeMap[int32, model.EquipInfo]()}
		global.PlayerEquipCache.Set(playerID, row)
		return row, nil
	}
	if err != nil {
		return nil, err
	}
	if row.Equips == nil {
		row.Equips = util.NewSafeMap[int32, model.EquipInfo]()
	}
	return row, nil
}
