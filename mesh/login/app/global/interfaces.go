// Package global 全局契约与装配：业务接口定义 + 基础设施初始化。
// 依赖方向：handler -> logic -> global；global 不反向依赖 logic/repo，
// logic/repo 通过包级 init 自注册单例，替代 app/wire 显式装配。
package global

import (
	"context"

	"server-cluster/share/dto"
)

// AccountLogic 账号模块业务接口。
type AccountLogic interface {
	// Register 注册账号
	Register(ctx context.Context, req *dto.RegisterReq) (*dto.RegisterResp, error)
	// Login 登录并签发 token
	Login(ctx context.Context, req *dto.LoginReq) (*dto.LoginResp, error)
}
