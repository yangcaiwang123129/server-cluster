package global

import "github.com/dobyte/due/v2/cluster/node"

type PlayerActor struct {
	*node.Actor
	PlayerID  int64  // 玩家ID
	EnterTime int64  // 上线 Unix 秒
	LoginIP   string // 登录 IP（网关从连接获取，内存维护，不落库）
}
