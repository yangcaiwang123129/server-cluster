// Package logic 业务实现：实现 app 定义的接口。
package logic

import (
	"context"

	"server-cluster/share/pb"
)

type socialLogic struct{}

// NewSocialLogic 构造社交业务实现。
func NewSocialLogic() *socialLogic { return &socialLogic{} }

func (l *socialLogic) SendChat(ctx context.Context, uid int64, req *pb.SendChatReq) (*pb.SendChatResp, error) {
	// TODO: 实现聊天广播/落库逻辑
	return &pb.SendChatResp{MsgId: 1}, nil
}
