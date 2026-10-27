// Package global 全局契约与装配：业务接口定义 + 基础设施初始化。
// 依赖方向：handler -> logic -> global；global 不反向依赖 logic/repo，
// logic/repo 通过包级 init 自注册单例，替代 app/wire 显式装配。
package global

// BattleLogic 战斗模块业务接口。
type BattleLogic interface {
	// StartBattle 开始战斗
	//StartBattle(ctx context.Context, uid int64, req *pb.StartBattleReq) (*pb.StartBattleResp, error)
}
