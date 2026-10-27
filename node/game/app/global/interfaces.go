// Package global 全局契约与装配：业务接口定义 + 基础设施初始化。
// 依赖方向：handler -> logic -> global；global 不反向依赖 logic/repo，
// logic/repo 通过包级 init 自注册单例，替代 app/wire 显式装配。
package global

import (
	"context"
	"server-cluster/share/model"
	"server-cluster/share/pb"
)

// PlayerLogic 玩家模块业务接口。
type PlayerLogic interface {
	// EnterGame 进入游戏
	EnterGame(ctx context.Context, playerID int64, req *pb.EnterGameReq) (*pb.EnterGameResp, error)
	// LeaveGame 离开游戏
	LeaveGame(ctx context.Context, playerID int64, req *pb.LeaveGameReq) (*pb.LeaveGameResp, error)
}

// PlayerRepo 玩家数据访问接口。
type PlayerRepo interface {
	// Load 按 playerID 加载玩家；不存在返回 errs.ErrPlayerNotFound。
	Load(ctx context.Context, playerID int64) (*model.PlayerModel, error)
	// Save 新建或更新玩家。
	Save(ctx context.Context, p *model.PlayerModel) error
}

// PlayerContainerLogic 玩家容器业务接口。
type PlayerContainerLogic interface {
	// GetContainerByType 通过类型拿到容器
	GetContainerByType(ctx context.Context, playerID int64, req *pb.EnterGameReq) (*pb.EnterGameResp, error)
}

// PlayerHeroLogic 玩家英雄列表业务接口。
type PlayerHeroLogic interface {
	// GetHeroById 通过id获取英雄
	GetHeroById(ctx context.Context, playerID int64, req *pb.EnterGameReq) (*pb.EnterGameResp, error)
}
