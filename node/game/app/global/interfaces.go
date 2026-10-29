// Package global 全局契约与装配：业务接口定义 + 基础设施初始化。
// 依赖方向：handler -> logic -> global；logic 通过包级 init 自注册单例。
package global

import (
	"context"

	"server-cluster/share/pb"
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

// ContainerLogic 容器业务接口：
type ContainerLogic interface {
	ModuleLogic
	// ContainerList 容器列表
	ContainerList(ctx context.Context, playerID int64, req *pb.ContainerListReq) (*pb.ContainerListResp, error)
	// ContainerMove 容器内移动物品
	ContainerMove(ctx context.Context, playerID int64, req *pb.ContainerMoveReq) (*pb.ContainerMoveResp, error)
	// ContainerRotate 容器内旋转物品
	ContainerRotate(ctx context.Context, playerID int64, req *pb.ContainerRotateReq) (*pb.ContainerRotateResp, error)
	// ContainerSort 容器一键整理
	ContainerSort(ctx context.Context, playerID int64, req *pb.SortContainerReq) (*pb.SortContainerResp, error)
	// ContainerTransfer 容器间移动物品
	ContainerTransfer(ctx context.Context, playerID int64, req *pb.ContainerTransferReq) (*pb.ContainerTransferResp, error)
	// ContainerAddItem 向容器放入物品（GM，返回变更后的容器）
	ContainerAddItem(ctx context.Context, playerID int64, req *pb.GmContainerAddItemReq) (*pb.ContainerListResp, error)
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
