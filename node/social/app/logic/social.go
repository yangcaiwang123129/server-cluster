// Package logic 业务实现：实现 global 定义的接口；单例容器在本包，init 自注册（替代 app/wire）。
package logic

import (
	"server-cluster/node/social/app/global"
	"server-cluster/share/singleton"
)

type socialLogic struct{}

var socialLogicInst = singleton.NewContainer[global.SocialLogic]("SocialLogic")

func init() {
	socialLogicInst.Register(&socialLogic{})
}

// Social 获取社交模块业务单例。
func Social() global.SocialLogic { return socialLogicInst.MustGet() }

//
//func (l *socialLogic) SendChat(ctx context.Context, uid int64, req *pb.SendChatReq) (*pb.SendChatResp, error) {
//	// TODO: 实现聊天广播/落库逻辑
//	return &pb.SendChatResp{MsgId: 1}, nil
//}
