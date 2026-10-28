package global

import "github.com/dobyte/due/v2/cluster/node"

type PlayerActor struct {
	*node.Actor
	PlayerID  int64 // 玩家ID
	EnterTime int64 // 上线 Unix 秒
}
