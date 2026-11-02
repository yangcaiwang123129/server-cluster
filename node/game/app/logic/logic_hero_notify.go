package logic

import (
	"context"

	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/errors"
	"github.com/dobyte/due/v2/log"
	"github.com/dobyte/due/v2/session"

	"server-cluster/internal/svc"
	"server-cluster/share/pb"
)

// notifyHeroEntry 推送英雄完整持有结构；客户端收到 HeroNotify 后整体替换本地该英雄数据。
// 玩家离线或推送失败只记日志，不影响本次业务结果。
func notifyHeroEntry(ctx context.Context, playerID int64, entry *pb.HeroEntry) {
	if playerID <= 0 || entry == nil {
		return
	}
	// Ack=true：gate 上找不到该用户会话时，due 会清除本地定位缓存并重新定位后重试一次，
	// 避免玩家换网关/跨节点重连后仍向旧网关推送导致静默丢失。
	if err := svc.NodeProxy().Push(ctx, &cluster.PushArgs{
		Kind:   session.User,
		Target: playerID,
		Ack:    true,
		Message: &cluster.Message{
			Route: int32(pb.RouteCode_ROUTE_CODE_HERO_NOTIFY),
			Data:  &pb.HeroNotify{HeroEntry: entry},
		},
	}); err != nil {
		// 玩家离线不是故障：定位不到 gate 或 gate 上无会话时降级为 Debug，避免线上告警噪声。
		if errors.Is(err, errors.ErrNotFoundUserLocation) || errors.Is(err, errors.ErrNotFoundSession) {
			log.Debugf("game: hero notify skipped, player offline, player=%d hero=%d: %v", playerID, entry.GetCfgId(), err)
			return
		}
		log.Warnf("game: push hero notify failed, player=%d hero=%d route=%d: %v",
			playerID, entry.GetCfgId(), pb.RouteCode_ROUTE_CODE_HERO_NOTIFY, err)
	}
}

// notifyHero 按英雄ID组装完整持有结构后推送；英雄未拥有时不推送。
func notifyHero(ctx context.Context, playerID int64, heroID int32) {
	if heroID <= 0 {
		return
	}
	entry, err := heroEntry(ctx, playerID, heroID)
	if err != nil {
		log.Warnf("game: assemble hero entry failed, player=%d hero=%d: %v", playerID, heroID, err)
		return
	}
	notifyHeroEntry(ctx, playerID, entry)
}

// notifyHeroByWeapon 武器变化后按固定绑定反查所属英雄并推送。
func notifyHeroByWeapon(ctx context.Context, playerID int64, cfgID int32) {
	heroID, ok := heroIDByWeapon(cfgID)
	if !ok {
		return
	}
	notifyHero(ctx, playerID, heroID)
}

// notifyHeroBySkill 技能变化后按固定绑定反查所属英雄并推送。
func notifyHeroBySkill(ctx context.Context, playerID int64, cfgID int32) {
	heroID, ok := heroIDBySkill(cfgID)
	if !ok {
		return
	}
	notifyHero(ctx, playerID, heroID)
}
