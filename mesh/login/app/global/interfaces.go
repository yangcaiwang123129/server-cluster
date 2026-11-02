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
	// PlayerLogin 统一登录入口，按 loginType 分发到各分支。
	PlayerLogin(ctx context.Context, req *dto.PlayerLoginReq) (*dto.PlayerLoginResp, error)
	// PlayerBind 绑定新凭据（游客升级 / 已登录玩家加绑）。
	PlayerBind(ctx context.Context, req *dto.PlayerBindReq) error
	// SendPhoneCode 发送短信验证码。
	SendPhoneCode(ctx context.Context, req *dto.PlayerPhoneSendCodeReq) error
}
