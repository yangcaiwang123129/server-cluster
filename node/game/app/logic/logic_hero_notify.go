package logic

import (
	"context"

	"github.com/dobyte/due/v2/log"

	"server-cluster/node/game/app/global"
)

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
	global.NotifyHero(ctx, playerID, entry)
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
