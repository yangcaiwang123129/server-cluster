// Package app 装配层：业务接口定义 + 单例注册 + 依赖装配。
package app

import (
	"server-cluster/share/singleton"

	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/internal/svc"
	"server-cluster/node/social/app/logic"
	"server-cluster/node/social/app/repo"
	"server-cluster/node/social/app/service"
)

// SocialLogic 社交模块业务接口。
type SocialLogic interface {
	// SendChat 发送聊天
	//SendChat(ctx context.Context, uid int64, req *pb.SendChatReq) (*pb.SendChatResp, error)
}

var socialLogic = singleton.NewContainer[SocialLogic]("SocialLogic")

// RegisterSocialLogic 装配期注册实现；仅 wire 调用。
func RegisterSocialLogic(l SocialLogic) { socialLogic.Register(l) }

// Social 获取社交模块业务单例。
func Social() SocialLogic { return socialLogic.MustGet() }

// Init 初始化应用：svc（DB/Redis）→ service（mesh 客户端）→ wire（注册实现）。
// 须在注册路由前调用。
func Init(proxy *node.Proxy) error {
	if err := svc.Init(); err != nil {
		return err
	}
	if err := service.Init(proxy); err != nil {
		return err
	}
	wire()
	return nil
}

// Destroy 进程退出后释放 DB / Redis 等资源。
func Destroy() {
	_ = svc.Close()
}

// wire 依赖装配：构造各模块实现并注册到单例容器。
// 注意依赖方向：app -> logic/repo，反向只允许依赖接口，避免循环引用。
func wire() {
	repo.RegisterSocial(repo.NewSocialRepo())
	RegisterSocialLogic(logic.NewSocialLogic())
}
