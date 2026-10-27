// Package global 全局契约与装配：业务接口定义 + 基础设施初始化。
// 依赖方向：handler -> logic -> global；logic 通过包级 init 自注册单例。
package global

import (
	"context"
	"server-cluster/share/pb"
)

// PlayerLogic 玩家模块业务接口。
type PlayerLogic interface {
	// EnterGame 进入游戏
	EnterGame(ctx context.Context, playerID int64, req *pb.EnterGameReq) (*pb.EnterGameResp, error)
	// LeaveGame 离开游戏
	LeaveGame(ctx context.Context, playerID int64, req *pb.LeaveGameReq) (*pb.LeaveGameResp, error)
}

// PlayerContainerLogic 玩家容器业务接口。
type PlayerContainerLogic interface {
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
