// notify.go 集中收口 game 服主动下发的 pb 通知：所有 Push 只在这里实现，
// 业务层只负责组装数据并调用对应的 NotifyXxx。
package global

import (
	"context"

	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/errors"
	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/svc"
	"server-cluster/share/pb"
)

// NotifyHero 推送英雄完整持有结构；客户端收到后整体替换本地该英雄数据。
// 玩家离线或推送失败只记日志，不影响业务主流程。
func NotifyHero(ctx context.Context, playerID int64, entry *pb.HeroEntry) {
	if playerID <= 0 || entry == nil {
		return
	}
	pushToPlayer(ctx, playerID, &cluster.Message{
		Route: int32(pb.RouteCode_ROUTE_CODE_HERO_NOTIFY),
		Data:  &pb.HeroNotify{HeroEntry: entry},
	}, "hero")
}

// NotifyTask 推送任务进度/状态变化；玩家离线只记日志。
func NotifyTask(ctx context.Context, playerID int64, entries []*pb.TaskEntry) {
	if playerID <= 0 || len(entries) == 0 {
		return
	}
	pushToPlayer(ctx, playerID, &cluster.Message{
		Route: int32(pb.RouteCode_ROUTE_CODE_TASK_NOTIFY),
		Data:  &pb.TaskNotify{Entries: entries},
	}, "task")
}

// NotifyAttr 推送玩家属性变化（当前用于货币快照，键为 Attr 枚举值）；玩家离线只记日志。
func NotifyAttr(ctx context.Context, playerID int64, values *pb.AttrMap) {
	if playerID <= 0 || values == nil || len(values.Values) == 0 {
		return
	}
	pushToPlayer(ctx, playerID, &cluster.Message{
		Route: int32(pb.RouteCode_ROUTE_CODE_ATTR_NOTIFY),
		Data:  &pb.AttrNotify{Values: values},
	}, "attr")
}

// pushToPlayer 向指定玩家推送消息。
// Ack=true：gate 上找不到该用户会话时，due 会清除本地定位缓存并重新定位后重试一次，
// 避免玩家换网关/跨节点重连后仍向旧网关推送导致静默丢失。
func pushToPlayer(ctx context.Context, playerID int64, message *cluster.Message, tag string) {
	if err := svc.NodeProxy().Push(ctx, &cluster.PushArgs{
		Kind:    cluster.User,
		Target:  playerID,
		Ack:     true,
		Message: message,
	}); err != nil {
		// 玩家离线不是故障：定位不到 gate 或 gate 上无会话时降级为 Debug，避免线上告警噪声。
		if errors.Is(err, errors.ErrNotFoundUserLocation) || errors.Is(err, errors.ErrNotFoundSession) {
			log.Debugf("game: %s notify skipped, player offline, player=%d", tag, playerID)
			return
		}
		log.Warnf("game: push %s notify failed, player=%d route=%d: %v", tag, playerID, message.Route, err)
	}
}
