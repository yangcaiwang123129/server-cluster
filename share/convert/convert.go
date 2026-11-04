// Package convert 实体与协议（pb）之间的统一转换层。
// 所有业务模型 / 配置到 wire 结构的映射都集中在这里，业务层只负责取数与流程，不再各自实现转换。
package convert

import (
	"server-cluster/share/consts"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

// HeroEntry 英雄持有数据 → wire 条目（仅英雄状态；绑定的武器/技能/装备由业务层组装）。
func HeroEntry(h model.HeroInfo) *pb.HeroEntry {
	return &pb.HeroEntry{
		CfgId:    h.HeroID,
		Level:    h.Level,
		Exp:      h.Exp,
		Star:     h.Star,
		SkinId:   h.SkinID,
		Equipped: h.Equipped,
	}
}

// WeaponEntry 武器持有数据 → wire 条目（仅状态）。
func WeaponEntry(w model.WeaponInfo) *pb.WeaponEntry {
	return &pb.WeaponEntry{
		CfgId:        w.WeaponID,
		Level:        w.Level,
		Durability:   w.Durability,
		Quality:      w.Quality,
		Breakthrough: w.Breakthrough,
	}
}

// SkillEntry 技能持有数据 → wire 条目（仅状态）。
func SkillEntry(s model.SkillInfo) *pb.SkillEntry {
	return &pb.SkillEntry{
		CfgId:        s.SkillID,
		Level:        s.Level,
		Quality:      s.Quality,
		Breakthrough: s.Breakthrough,
	}
}

// EquipEntry 装备持有数据 → wire 条目（仅状态）。
func EquipEntry(g model.EquipInfo) *pb.EquipEntry {
	return &pb.EquipEntry{
		Id:           g.EquipID,
		CfgId:        g.CfgID,
		HeroId:       g.HeroID,
		Level:        g.Level,
		Equipped:     g.Equipped,
		Quality:      g.Quality,
		Breakthrough: g.Breakthrough,
	}
}

// MailEntry 邮件持有数据 → wire 条目。
func MailEntry(m model.MailInfo) *pb.MailEntry {
	return &pb.MailEntry{
		Id:      m.MailID,
		Title:   m.Title,
		Content: m.Content,
		Attachments: util.MapSlice(m.Attachments, func(a model.MailAttachment) *pb.AssetEntry {
			return &pb.AssetEntry{Type: a.Type, CfgId: a.CfgID, Count: a.Count}
		}),
		Status:    pb.MailStatus(m.Status),
		CreatedAt: m.CreatedAt,
		ReadAt:    m.ReadAt,
		ClaimedAt: m.ClaimedAt,
	}
}

// TaskEntry 任务配置 + 玩家进度 → wire 条目。
func TaskEntry(cfg *tpl.TaskCfg, info model.PlayerTaskInfo) *pb.TaskEntry {
	rewards := make([]*pb.AssetEntry, 0, len(cfg.AssetRewards))
	for _, a := range cfg.AssetRewards {
		if a == nil {
			continue
		}
		rewards = append(rewards, &pb.AssetEntry{Type: int32(a.Type), CfgId: a.Id, Count: a.Count})
	}
	return &pb.TaskEntry{
		TaskId:         cfg.Id,
		TaskType:       int32(cfg.TaskType),
		TargetTaskType: int32(cfg.TargetTaskType),
		PeriodKey:      info.PeriodKey,
		Progress:       info.Progress,
		Target:         info.Target,
		Status:         info.Status,
		CompletedAt:    info.CompletedAt,
		ClaimedAt:      info.ClaimedAt,
		Name:           cfg.Name,
		Desc:           cfg.Desc,
		Rewards:        rewards,
		Sort:           cfg.Sort,
		ActivityCfgId:  cfg.TaskGroupID,
		PreTaskId:      cfg.PreTaskId,
		AcceptType:     int32(cfg.AcceptType),
	}
}

// AssetEntry 资产配置 → wire 条目；Count<=0 与发放逻辑一致按 1 处理。
func AssetEntry(a *tpl.Asset) *pb.AssetEntry {
	count := a.Count
	if count <= 0 {
		count = 1
	}
	return &pb.AssetEntry{Type: int32(a.Type), CfgId: a.Id, Count: count}
}

// ItemContainer 物品容器实体 → wire 物品容器条目。
func ItemContainer(playerID int64, c *model.ItemContainer) *pb.ItemContainer {
	meta := c.Meta()
	items := c.Entries()
	entries := make([]*pb.ItemEntry, 0, len(items))
	for _, it := range items {
		entries = append(entries, &pb.ItemEntry{
			Id:      it.ItemID,
			CfgId:   it.ItemCfgID,
			Count:   it.Count,
			GridX:   it.GridX,
			GridY:   it.GridY,
			Rotated: it.Rotated,
		})
	}
	return &pb.ItemContainer{
		Id:          itemContainerID(playerID, meta.Type),
		Type:        pb.ContainerType(meta.Type),
		GridW:       meta.GridW,
		GridH:       meta.GridH,
		Version:     meta.Ver,
		ItemEntries: entries,
	}
}

// ContainerTypeToModel pb 物品容器类型 → 领域物品容器类型；UNSPECIFIED 等非法值返回 false。
func ContainerTypeToModel(t pb.ContainerType) (int32, bool) {
	switch t {
	case pb.ContainerType_CONTAINER_TYPE_BACKPACK:
		return int32(pb.ContainerType_CONTAINER_TYPE_BACKPACK), true
	case pb.ContainerType_CONTAINER_TYPE_STASH:
		return int32(pb.ContainerType_CONTAINER_TYPE_STASH), true
	case pb.ContainerType_CONTAINER_TYPE_SECURE:
		return int32(pb.ContainerType_CONTAINER_TYPE_SECURE), true
	default:
		return 0, false
	}
}

// PlayerAttr 玩家实体 → 下发客户端的属性表：昵称/等级/经验/vip 取自 player_extra，币/金取自 player。
// 键为 Attr 枚举值，值按实际类型放入 ObjectValue。
func PlayerAttr(playerExtra *model.PlayerExtra, player *model.Player) *pb.AttrMap {
	return &pb.AttrMap{Values: map[int32]*pb.ObjectValue{
		int32(pb.Attr_PLAYER_ATTR_NAME):          {Kind: &pb.ObjectValue_StringValue{StringValue: playerExtra.Nickname}},
		int32(pb.Attr_PLAYER_ATTR_LEVEL):         {Kind: &pb.ObjectValue_IntValue{IntValue: playerExtra.Level}},
		int32(pb.Attr_PLAYER_ATTR_EXP):           {Kind: &pb.ObjectValue_BigintValue{BigintValue: playerExtra.Exp}},
		int32(pb.Attr_PLAYER_ATTR_VIP_LEVEL):     {Kind: &pb.ObjectValue_IntValue{IntValue: playerExtra.VipLevel}},
		int32(pb.Attr_PLAYER_ATTR_VIP_EXP):       {Kind: &pb.ObjectValue_BigintValue{BigintValue: playerExtra.VipExp}},
		int32(pb.Attr_PLAYER_ATTR_COIN):          {Kind: &pb.ObjectValue_BigintValue{BigintValue: player.Coin}},
		int32(pb.Attr_PLAYER_ATTR_GOLD):          {Kind: &pb.ObjectValue_BigintValue{BigintValue: player.Gold}},
		int32(pb.Attr_PLAYER_ATTR_COUPON):        {Kind: &pb.ObjectValue_BigintValue{BigintValue: player.Coupon}},
		int32(pb.Attr_PLAYER_ATTR_SIGNATURE):     {Kind: &pb.ObjectValue_StringValue{StringValue: playerExtra.Signature}},
		int32(pb.Attr_PLAYER_ATTR_SETTINGS_JSON): {Kind: &pb.ObjectValue_StringValue{StringValue: playerExtra.SettingsJSON}},
	}}
}

// itemContainerID 物品容器实例 ID：playerID 与物品容器类型编码（playerID*ContainerTypeStride + type）。
func itemContainerID(playerID int64, typ int32) int64 {
	return playerID*consts.ContainerTypeStride + int64(typ)
}
