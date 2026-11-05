// notify.go 是 game 服给玩家主动下发 pb 通知的统一出口：
//   - 全服仅 pushToPlayer 一处调用 svc.NodeProxy().Push，业务层只组装 pb 数据并调用
//     对应的 NotifyXxx，禁止在别处直调 Push；
//   - 依赖 logic 私有数据组装的消息（如按英雄ID构建 HeroEntry），取数部分留在 logic，
//     构造好 pb 后再调本包 NotifyXxx；本文件不 import logic，保证 handler→logic→global 单向依赖。
//
// Ack 语义：所有推送 Ack=true，玩家换网关/跨节点重连后由框架重新定位并重试一次，避免切换瞬间消息静默丢失。
//
// 离线降级：玩家不在线（定位不到 gate 或 gate 上无会话）不算故障，降级为 Debug 日志；其余推送失败记 Warn。
package global

import (
	"context"

	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/errors"
	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/svc"
	"server-cluster/node/game/app/repo"
	"server-cluster/share/pb"
)

// NotifyCurrencySnapshot 直查玩家行推送货币（Coupon/Gold/Coin）快照；行缺失（未建档）静默跳过。
func NotifyCurrencySnapshot(ctx context.Context, playerID int64) {
	row, err := repo.QueryPlayer(ctx, playerID)
	if err != nil {
		return
	}
	NotifyAttr(ctx, playerID, &pb.AttrMap{Values: map[int32]*pb.ObjectValue{
		int32(pb.Attr_PLAYER_ATTR_COUPON): {Kind: &pb.ObjectValue_BigintValue{BigintValue: row.Coupon}},
		int32(pb.Attr_PLAYER_ATTR_GOLD):   {Kind: &pb.ObjectValue_BigintValue{BigintValue: row.Gold}},
		int32(pb.Attr_PLAYER_ATTR_COIN):   {Kind: &pb.ObjectValue_BigintValue{BigintValue: row.Coin}},
	}})
}

// NotifyAttr 推送玩家属性变化；玩家离线只记日志。
func NotifyAttr(ctx context.Context, playerID int64, values *pb.AttrMap) {
	if playerID <= 0 || values == nil || len(values.Values) == 0 {
		return
	}
	pushToPlayer(ctx, playerID, &cluster.Message{
		Route: int32(pb.RouteCode_ROUTE_CODE_ATTR_NOTIFY),
		Data:  &pb.AttrNotify{Values: values},
	}, "attr")
}

// NotifyHero 推送英雄完整持有结构（客户端整体替换本地该英雄数据）；离线或推送失败只记日志。
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

// pushToPlayer 向指定玩家推送消息；Ack=true 在换网关/跨节点重连后重新定位并重试一次，避免静默丢失。
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
