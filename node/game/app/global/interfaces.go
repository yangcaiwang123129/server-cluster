// Package global 全局契约与装配：业务接口定义 + 基础设施初始化。
// 依赖方向：handler -> logic -> global；logic 通过包级 init 自注册单例。
package global

import (
	"context"

	"server-cluster/share/pb"
	"server-cluster/share/tpl"
)

// ModuleLogic 玩家模块生命周期接口：进入游戏预热/建档该模块数据，离开游戏同步落库并清除缓存。
// 所有持有玩家数据的模块都实现它，由 PlayerLogic 的 EnterGame/LeaveGame 统一驱动。
type ModuleLogic interface {
	// EnterGame 进入游戏：加载并预热该模块的玩家数据（未建档则初始化）
	EnterGame(ctx context.Context, playerID int64, req *pb.EnterGameReq) (*pb.EnterGameResp, error)
	// LeaveGame 离开游戏：该玩家的待写数据同步落库，并逐出内存缓存
	LeaveGame(ctx context.Context, playerID int64, req *pb.LeaveGameReq) (*pb.LeaveGameResp, error)
}

// PlayerLogic 玩家模块业务接口。
type PlayerLogic interface {
	ModuleLogic
	// PlayerGetActor 获取玩家actor
	PlayerGetActor(ctx context.Context, playerID int64) (*PlayerActor, error)
	// PlayerGetActors 玩家获取所有actor
	PlayerGetActors(ctx context.Context) ([]*PlayerActor, error)
	// PlayerCreateActor 玩家创建actor
	PlayerCreateActor(ctx context.Context, playerID int64) error
	// PlayerDestroyActor 玩家销毁actor
	PlayerDestroyActor(ctx context.Context, playerID int64) error
	// PlayerKick 玩家踢下线
	PlayerKick(ctx context.Context, playerID int64) error
}

// ItemLogic 物品容器业务接口：
type ItemLogic interface {
	ModuleLogic
	// ItemContainerList 物品容器列表
	ItemContainerList(ctx context.Context, playerID int64, req *pb.ItemContainerListReq) (*pb.ItemContainerListResp, error)
	// ItemMove 物品容器内移动物品
	ItemMove(ctx context.Context, playerID int64, req *pb.ItemMoveReq) (*pb.ItemMoveResp, error)
	// ItemRotate 物品容器内旋转物品
	ItemRotate(ctx context.Context, playerID int64, req *pb.ItemRotateReq) (*pb.ItemRotateResp, error)
	// ItemSort 物品容器一键整理
	ItemSort(ctx context.Context, playerID int64, req *pb.ItemSortReq) (*pb.ItemSortResp, error)
	// ItemTransfer 物品容器间移动物品
	ItemTransfer(ctx context.Context, playerID int64, req *pb.ItemTransferReq) (*pb.ItemTransferResp, error)
	// ItemAddEntry 向物品容器放入对象（GM，返回变更后的物品容器）
	ItemAddEntry(ctx context.Context, playerID int64, req *pb.GmItemAddEntryReq) (*pb.ItemContainerListResp, error)
}

// SkillLogic 技能业务接口：玩家技能持有数据生命周期。
type SkillLogic interface {
	ModuleLogic
	// SkillList 技能列表
	SkillList(ctx context.Context, playerID int64) (*pb.SkillListResp, error)
	// SkillAdd 新增技能
	SkillAdd(ctx context.Context, playerID int64, req *pb.SkillAddReq) (*pb.SkillAddResp, error)
	// SkillUpdate 更新技能
	SkillUpdate(ctx context.Context, playerID int64, req *pb.SkillUpdateReq) (*pb.SkillUpdateResp, error)
}

// WeaponLogic 武器业务接口：玩家武器持有数据生命周期。
type WeaponLogic interface {
	ModuleLogic
	// WeaponList 武器列表
	WeaponList(ctx context.Context, playerID int64) (*pb.WeaponListResp, error)
	// WeaponAdd 新增武器
	WeaponAdd(ctx context.Context, playerID int64, req *pb.WeaponAddReq) (*pb.WeaponAddResp, error)
	// WeaponUpdate 更新武器
	WeaponUpdate(ctx context.Context, playerID int64, req *pb.WeaponUpdateReq) (*pb.WeaponUpdateResp, error)
}

// HeroLogic 英雄业务接口：玩家英雄持有数据生命周期。
type HeroLogic interface {
	ModuleLogic
	// HeroList 英雄列表
	HeroList(ctx context.Context, playerID int64) (*pb.HeroListResp, error)
	// HeroAdd 新增英雄
	HeroAdd(ctx context.Context, playerID int64, req *pb.HeroAddReq) (*pb.HeroAddResp, error)
	// HeroUpdate 更新英雄
	HeroUpdate(ctx context.Context, playerID int64, req *pb.HeroUpdateReq) (*pb.HeroUpdateResp, error)
}

// EquipLogic 装备业务接口：玩家装备持有数据生命周期。
type EquipLogic interface {
	ModuleLogic
	// EquipList 装备列表
	EquipList(ctx context.Context, playerID int64) (*pb.EquipListResp, error)
	// EquipAdd 新增装备
	EquipAdd(ctx context.Context, playerID int64, req *pb.EquipAddReq) (*pb.EquipAddResp, error)
	// EquipUpdate 更新装备
	EquipUpdate(ctx context.Context, playerID int64, req *pb.EquipUpdateReq) (*pb.EquipUpdateResp, error)
}

// AssetLogic 资产业务统一入口：按 tpl.AssetType 路由到对应实现（item/equip/weapon/skill/hero 等），
// 供奖励发放、消耗校验等场景复用，不关心资产落在哪张表/哪个容器。
type AssetLogic interface {
	// AssetAdd 发放资产；Count<=0 按 1 处理
	AssetAdd(ctx context.Context, playerID int64, asset *tpl.Asset) error
	// AssetRemove 扣除资产；持有不足返回错误
	AssetRemove(ctx context.Context, playerID int64, asset *tpl.Asset) error
	// AssetUpdate 覆盖持有状态：Count<=0 表示移除，>0 表示确保持有（可叠加类型按数量覆盖）
	AssetUpdate(ctx context.Context, playerID int64, asset *tpl.Asset) error
	// AssetQuery 查询单个配置ID的持有数量；未持有返回 0
	AssetQuery(ctx context.Context, playerID int64, assetType tpl.AssetType, cfgID int32) (int32, error)
	// AssetList 列出某类资产的全部持有
	AssetList(ctx context.Context, playerID int64, assetType tpl.AssetType) ([]*tpl.Asset, error)
}

// AssetHandler 单类资产的增删改查实现：item / equip / weapon / skill / hero 各一个实现类。
type AssetHandler interface {
	// AssetType 该实现负责的资产类型
	AssetType() tpl.AssetType
	// Add 发放；Count<=0 按 1 处理
	Add(ctx context.Context, playerID int64, cfgID, count int32) error
	// Remove 扣除；持有不足返回错误
	Remove(ctx context.Context, playerID int64, cfgID, count int32) error
	// Update 覆盖持有：count<=0 移除，>0 确保持有
	Update(ctx context.Context, playerID int64, cfgID, count int32) error
	// Query 查询持有数量
	Query(ctx context.Context, playerID int64, cfgID int32) (int32, error)
	// List 列出该类资产全部持有
	List(ctx context.Context, playerID int64) ([]*tpl.Asset, error)
}
