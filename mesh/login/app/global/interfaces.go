// Package global 全局契约与装配：业务接口定义 + 基础设施初始化。
// 依赖方向：handler -> logic -> global；global 不反向依赖 logic/repo，
// logic/repo 通过包级 init 自注册单例，替代 app/wire 显式装配。
package global

import (
	"context"

	"server-cluster/share/dto"
)

// PlayerLogic 玩家模块业务接口。
type PlayerLogic interface {
	// PlayerRegister 玩家注册账号
	PlayerRegister(ctx context.Context, req *dto.PlayerRegisterReq) (*dto.PlayerRegisterResp, error)
	// PlayerLogin 登录登录并签发 token
	PlayerLogin(ctx context.Context, req *dto.PlayerLoginReq) (*dto.PlayerLoginResp, error)
}
