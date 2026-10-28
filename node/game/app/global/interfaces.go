// Package global 全局契约与装配：业务接口定义 + 基础设施初始化。
// 依赖方向：handler -> logic -> global；logic 通过包级 init 自注册单例。
package global

import (
	"context"
	"server-cluster/share/pb"
)

// PlayerModuleLogic 玩家模块生命周期接口：进入游戏预热/建档该模块数据，离开游戏同步落库并清除缓存。
// 所有持有玩家数据的模块都实现它，由 PlayerLogic 的 EnterGame/LeaveGame 统一驱动。
type PlayerModuleLogic interface {
	// EnterGame 进入游戏：加载并预热该模块的玩家数据（未建档则初始化）
	EnterGame(ctx context.Context, playerID int64, req *pb.EnterGameReq) (*pb.EnterGameResp, error)
	// LeaveGame 离开游戏：该玩家的待写数据同步落库，并逐出内存缓存
	LeaveGame(ctx context.Context, playerID int64, req *pb.LeaveGameReq) (*pb.LeaveGameResp, error)
}

// PlayerLogic 玩家模块业务接口。
type PlayerLogic interface {
	PlayerModuleLogic
}

// PlayerContainerLogic 玩家容器业务接口。
type PlayerContainerLogic interface {
	PlayerModuleLogic
	// GetContainer 拉取容器快照
	GetContainer(ctx context.Context, playerID int64, req *pb.GetContainerReq) (*pb.GetContainerResp, error)
	// MoveItem 移动物品
	MoveItem(ctx context.Context, playerID int64, req *pb.MoveItemReq) (*pb.ContainerOpResp, error)
	// RotateItem 旋转物品
	RotateItem(ctx context.Context, playerID int64, req *pb.RotateItemReq) (*pb.ContainerOpResp, error)
	// RemoveItem 删除物品
	RemoveItem(ctx context.Context, playerID int64, req *pb.RemoveItemReq) (*pb.ContainerOpResp, error)
	// SortContainer 一键整理
	SortContainer(ctx context.Context, playerID int64, req *pb.SortContainerReq) (*pb.ContainerOpResp, error)
}

// SkillLogic 技能业务接口：配置读取 + 玩家技能持有数据生命周期。
type SkillLogic interface {
	PlayerModuleLogic
	// SkillList 技能列表
	SkillList(ctx context.Context) (*pb.SkillListResp, error)
}

// WeaponLogic 武器业务接口：配置读取 + 玩家武器持有数据生命周期。
type WeaponLogic interface {
	PlayerModuleLogic
	// WeaponList 武器列表
	WeaponList(ctx context.Context) (*pb.WeaponListResp, error)
}

// HeroLogic 英雄业务接口：配置读取 + 玩家英雄持有数据生命周期。
type HeroLogic interface {
	PlayerModuleLogic
	// HeroList 英雄列表
	HeroList(ctx context.Context) (*pb.HeroListResp, error)
}
