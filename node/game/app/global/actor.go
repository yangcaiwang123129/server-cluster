// actor.go 是玩家 actor（due actor）的基础设施：actor 类型常量与唯一标识、
// 在线登记表、actor 处理器及路由注册器注入。actor 生命周期与在线登记一一对应：
// processor.Start 时写入 PlayerActors 并绑定 uid、注册业务路由，
// processor.Destroy 时从 PlayerActors 移除登记。
package global

import (
	"strconv"

	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/log"

	"server-cluster/share/util"
)

// ActorKindPlayer 玩家 actor 类型。
const ActorKindPlayer = "player"

type PlayerActor struct {
	*node.Actor
	PlayerID    int64  // 玩家ID
	EnterTime   int64  // 上线 Unix 秒
	LoginIP     string // 登录 IP（网关从连接获取，内存维护，不落库）
	ReconnectAt int64  // 最近一次断线重连复用 actor 的 Unix 秒（EnterGame 幂等复用时更新，供断线延迟销毁豁免判断）
}

// PlayerActors 在线玩家 actor 表（playerID -> 玩家 actor），包级单例。
var PlayerActors = util.NewSafeMap[int64, *PlayerActor]()

// ActorID 玩家 actor 唯一标识。
func ActorID(playerID int64) string {
	return strconv.FormatInt(playerID, 10)
}

// actorRouteRegistrar 玩家 actor 路由注册器，由 handler 包装配期注入（避免 logic → handler 循环依赖）。
var actorRouteRegistrar func(act *node.Actor)

// RegisterActorRouteRegistrar 注入玩家 actor 路由注册器（handler 包在装配期调用）。
func RegisterActorRouteRegistrar(fn func(act *node.Actor)) {
	actorRouteRegistrar = fn
}

// playerProcessor 玩家 actor 处理器：生命周期与在线登记一一对应。
type playerProcessor struct {
	node.BaseProcessor
	actor    *node.Actor
	playerID int64
	loginIP  string
}

// NewPlayerProcessor due actor Creator。
func NewPlayerProcessor(act *node.Actor, args ...any) node.Processor {
	var playerID int64
	var loginIP string
	if len(args) > 0 {
		playerID, _ = args[0].(int64)
	}
	if len(args) > 1 {
		loginIP, _ = args[1].(string)
	}
	return &playerProcessor{actor: act, playerID: playerID, loginIP: loginIP}
}

func (p *playerProcessor) Start() {
	PlayerActors.Set(p.playerID, &PlayerActor{
		Actor:     p.actor,
		PlayerID:  p.playerID,
		EnterTime: util.Now().Unix(),
		LoginIP:   p.loginIP,
	})

	// uid 与 actor 绑定：stateful 消息由 scheduler 定位到本 actor
	if err := p.actor.Proxy().BindActor(p.playerID, ActorKindPlayer, p.actor.ID()); err != nil {
		log.Errorf("game: bind player actor failed, playerID=%d: %v", p.playerID, err)
	}
	// 注册业务路由：消息进入 actor 后串行处理
	if actorRouteRegistrar != nil {
		actorRouteRegistrar(p.actor)
	}
}

func (p *playerProcessor) Destroy() {
	PlayerActors.Delete(p.playerID)
}
