// Package logic 业务实现：实现 app 定义的接口。
package logic

type battleLogic struct{}

// NewBattleLogic 构造战斗业务实现。
func NewBattleLogic() *battleLogic { return &battleLogic{} }

//func (l *battleLogic) StartBattle(ctx context.Context, uid int64, req *pb.StartBattleReq) (*pb.StartBattleResp, error) {
//	// TODO: 实现战斗匹配/创建房间逻辑
//	return &pb.StartBattleResp{BattleId: 1, Status: "matched"}, nil
//}
