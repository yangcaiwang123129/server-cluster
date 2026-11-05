package actor

import "github.com/dobyte/due/v2/cluster/node"

type PlayerActor struct {
	*node.Actor
	PlayerID    int64  // 玩家ID
	EnterTime   int64  // 上线 Unix 秒
	LoginIP     string // 登录 IP（网关从连接获取，内存维护，不落库）
	ReconnectAt int64  // 最近一次断线重连复用 actor 的 Unix 秒（EnterGame 幂等复用时更新，供断线延迟销毁豁免判断）
}
