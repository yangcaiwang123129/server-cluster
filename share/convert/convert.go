// Package convert 实体与协议（pb）之间的统一转换层。
// 所有业务模型 / 配置到 wire 结构的映射都集中在这里，业务层只负责取数与流程，不再各自实现转换。
package convert

import (
	"server-cluster/share/consts"
	"server-cluster/share/dto"
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

// PlayerAttr 玩家实体 → 下发客户端的属性表：昵称/等级/经验/vip 取自 player_extra，币/金取自 player，键为 Attr 枚举值。
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

// ========== game：模型 / DTO ↔ 协议 ==========

// Activities 活动快照 → wire 活动列表（键为实例ID，已取消的不下发）；快照未就绪返回 nil。
func Activities(snap *dto.ActivitySnapshot) map[int64]*pb.ActivityEntry {
	if snap == nil {
		return nil
	}
	out := make(map[int64]*pb.ActivityEntry, len(snap.Activities))
	for _, a := range snap.Activities {
		if a.Status == consts.ActivityStatusCancelled {
			continue
		}
		entry := &pb.ActivityEntry{
			InstanceId: a.ID,
			CfgId:      a.CfgID,
			Status:     a.Status,
			StartTime:  a.StartTime,
			EndTime:    a.EndTime,
			Round:      a.Round,
		}
		if cfg := tpl.T().Activity.Get(a.CfgID); cfg != nil {
			entry.Name = cfg.Name
			entry.Icon = cfg.Icon
			entry.Desc = cfg.Desc
		}
		out[a.ID] = entry
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// MailAttachmentEntry 邮件附件实体 → wire 资产条目；Count<=0 与发放逻辑一致按 1 处理。
func MailAttachmentEntry(a model.MailAttachment) *pb.AssetEntry {
	count := a.Count
	if count <= 0 {
		count = 1
	}
	return &pb.AssetEntry{Type: a.Type, CfgId: a.CfgID, Count: count}
}

// MailAttachmentsFromPb wire 资产条目 → 邮件附件实体。
func MailAttachmentsFromPb(entries []*pb.AssetEntry) []model.MailAttachment {
	out := make([]model.MailAttachment, 0, len(entries))
	for _, a := range entries {
		out = append(out, model.MailAttachment{Type: a.GetType(), CfgID: a.GetCfgId(), Count: a.GetCount()})
	}
	return out
}

// WeaponInfoFromPb 武器更新请求 → 武器实体（等级/品质下限 1，耐久/突破下限 0）。
func WeaponInfoFromPb(req *pb.WeaponUpdateReq) model.WeaponInfo {
	return model.WeaponInfo{
		WeaponID:     req.GetCfgId(),
		Level:        clampDim(req.GetLevel(), 1),
		Durability:   clampDim(req.GetDurability(), 0),
		Quality:      clampDim(req.GetQuality(), 1),
		Breakthrough: clampDim(req.GetBreakthrough(), 0),
	}
}

// SkillInfoFromPb 技能更新请求 → 技能实体（等级/品质下限 1，突破下限 0）。
func SkillInfoFromPb(req *pb.SkillUpdateReq) model.SkillInfo {
	return model.SkillInfo{
		SkillID:      req.GetCfgId(),
		Level:        clampDim(req.GetLevel(), 1),
		Quality:      clampDim(req.GetQuality(), 1),
		Breakthrough: clampDim(req.GetBreakthrough(), 0),
	}
}

// EquipInfoFromPb 装备更新请求 → 装备实体（保留原实例ID/配置ID；等级/品质下限 1，突破下限 0）。
func EquipInfoFromPb(old model.EquipInfo, req *pb.EquipUpdateReq, heroID int32) model.EquipInfo {
	return model.EquipInfo{
		EquipID:      old.EquipID,
		CfgID:        old.CfgID,
		HeroID:       heroID,
		Level:        clampDim(req.GetLevel(), 1),
		Equipped:     req.GetEquipped() && heroID > 0,
		Quality:      clampDim(req.GetQuality(), 1),
		Breakthrough: clampDim(req.GetBreakthrough(), 0),
	}
}

// OnlinePlayer 在线玩家运行时信息 → wire 在线玩家条目。
func OnlinePlayer(nodeID string, playerID, enterTime int64, loginIP string) *pb.OnlinePlayer {
	return &pb.OnlinePlayer{PlayerId: playerID, EnterTime: enterTime, NodeId: nodeID, LoginIp: loginIP}
}

// clampDim 维度下限收敛：等级/品质至少 1，突破至少 0。
func clampDim(v, min int32) int32 {
	if v < min {
		return min
	}
	return v
}

// ========== admin：DTO ↔ 协议 ==========

// OnlinePlayerResp wire 在线玩家条目 → 后台响应行。
func OnlinePlayerResp(p *pb.OnlinePlayer) dto.OnlinePlayerResp {
	return dto.OnlinePlayerResp{
		PlayerID:    p.GetPlayerId(),
		EnterTime:   p.GetEnterTime(),
		LobbyNodeID: p.GetNodeId(),
		LoginIP:     p.GetLoginIp(),
	}
}

// OnlinePlayersResp wire 在线玩家列表 → 后台响应行列表。
func OnlinePlayersResp(players []*pb.OnlinePlayer) []dto.OnlinePlayerResp {
	return util.MapSlice(players, OnlinePlayerResp)
}

// GmMailAttachment wire 资产条目 → 后台邮件附件。
func GmMailAttachment(a *pb.AssetEntry) dto.GmMailAttachment {
	return dto.GmMailAttachment{Type: a.GetType(), CfgID: a.GetCfgId(), Count: a.GetCount()}
}

// GmMailEntry wire 邮件条目 → 后台响应行。
func GmMailEntry(e *pb.MailEntry) dto.GmMailEntry {
	return dto.GmMailEntry{
		ID:          e.GetId(),
		Title:       e.GetTitle(),
		Content:     e.GetContent(),
		Attachments: util.MapSlice(e.GetAttachments(), GmMailAttachment),
		Status:      int32(e.GetStatus()),
		CreatedAt:   e.GetCreatedAt(),
	}
}

// GmMailEntries wire 邮件列表 → 后台响应行列表。
func GmMailEntries(entries []*pb.MailEntry) []dto.GmMailEntry {
	return util.MapSlice(entries, GmMailEntry)
}

// GmWeaponEntry wire 武器条目 → 后台响应行（补齐名称/图标，heroID 为反查到的绑定英雄）。
func GmWeaponEntry(e *pb.WeaponEntry, heroID int32) dto.GmWeaponEntry {
	row := dto.GmWeaponEntry{
		WeaponID:     e.GetCfgId(),
		Level:        e.GetLevel(),
		Quality:      e.GetQuality(),
		Breakthrough: e.GetBreakthrough(),
		Durability:   e.GetDurability(),
		HeroID:       heroID,
	}
	if cfg := tpl.T().Weapon.Get(e.GetCfgId()); cfg != nil {
		row.Name = cfg.WeaponName
		row.IconKey = cfg.Icon
	}
	return row
}

// GmSkillEntry wire 技能条目 → 后台响应行（补齐名称/描述，heroID 为反查到的绑定英雄）。
func GmSkillEntry(e *pb.SkillEntry, heroID int32) dto.GmSkillEntry {
	row := dto.GmSkillEntry{
		SkillID:      e.GetCfgId(),
		Level:        e.GetLevel(),
		Quality:      e.GetQuality(),
		Breakthrough: e.GetBreakthrough(),
		HeroID:       heroID,
	}
	if cfg := tpl.T().Skill.Get(e.GetCfgId()); cfg != nil {
		row.Name = cfg.SkillName
		row.Desc = cfg.Desc
	}
	return row
}

// GmEquipEntry wire 装备条目 → 后台响应行（补齐名称/图标/描述）。
func GmEquipEntry(e *pb.EquipEntry) dto.GmEquipEntry {
	row := dto.GmEquipEntry{
		ID:           e.GetId(),
		EquipID:      e.GetCfgId(),
		HeroID:       e.GetHeroId(),
		Level:        e.GetLevel(),
		Quality:      e.GetQuality(),
		Breakthrough: e.GetBreakthrough(),
		Equipped:     e.GetEquipped(),
	}
	if cfg := tpl.T().Equip.Get(e.GetCfgId()); cfg != nil {
		row.Name = cfg.EquiqName
		row.IconKey = cfg.Icon
		row.Desc = cfg.Desc
	}
	return row
}

// GmEquipEntries wire 装备列表 → 后台响应行列表。
func GmEquipEntries(entries []*pb.EquipEntry) []dto.GmEquipEntry {
	return util.MapSlice(entries, GmEquipEntry)
}

// GmHeroEntry wire 英雄条目 → 后台响应行（补齐名称/稀有度，并带上绑定的武器/技能/装备）。
func GmHeroEntry(e *pb.HeroEntry) dto.GmHeroEntry {
	row := dto.GmHeroEntry{
		HeroID:   e.GetCfgId(),
		Level:    e.GetLevel(),
		Exp:      e.GetExp(),
		Star:     e.GetStar(),
		SkinID:   e.GetSkinId(),
		Equipped: e.GetEquipped(),
	}
	if cfg := tpl.T().RoleInfo.Get(e.GetCfgId()); cfg != nil {
		row.Name = cfg.Name
		row.Rarity = cfg.Rarity
	}
	if w := e.GetWeapon(); w != nil {
		weapon := GmWeaponEntry(w, e.GetCfgId())
		row.Weapon = &weapon
	}
	for _, s := range e.GetSkills() {
		row.Skills = append(row.Skills, GmSkillEntry(s, e.GetCfgId()))
	}
	for _, g := range e.GetEquips() {
		row.Equips = append(row.Equips, GmEquipEntry(g))
	}
	return row
}

// GmHeroEntries wire 英雄列表 → 后台响应行列表。
func GmHeroEntries(entries []*pb.HeroEntry) []dto.GmHeroEntry {
	return util.MapSlice(entries, GmHeroEntry)
}

// GmHeroAttributeResp wire 英雄属性响应 → 后台响应行。
func GmHeroAttributeResp(resp *pb.HeroAttributeResp) *dto.GmHeroAttribute {
	entry := resp.GetHeroEntry()
	attr := resp.GetAttribute()
	return &dto.GmHeroAttribute{
		HeroID: entry.GetCfgId(),
		Level:  entry.GetLevel(),
		Star:   entry.GetStar(),
		Power:  attr.GetPower(),
		Hp:     attr.GetHp(),
		Armor:  attr.GetArmor(),
		Damage: attr.GetDamage(),
	}
}

// GmUpgradeResp 升级结果 → 后台响应行。
func GmUpgradeResp(cfgID, level int32, attr *pb.HeroAttribute) *dto.GmUpgradeResp {
	return &dto.GmUpgradeResp{
		CfgID:  cfgID,
		Level:  level,
		Power:  attr.GetPower(),
		Hp:     attr.GetHp(),
		Armor:  attr.GetArmor(),
		Damage: attr.GetDamage(),
	}
}

// GmEquipDimResp 装备维度操作结果（等级/品质/突破 + 英雄累计属性）→ 后台响应行。
func GmEquipDimResp(e *pb.EquipEntry, attr *pb.HeroAttribute) *dto.GmUpgradeResp {
	r := GmUpgradeResp(e.GetCfgId(), e.GetLevel(), attr)
	r.Quality = e.GetQuality()
	r.Breakthrough = e.GetBreakthrough()
	return r
}

// GmItemResp wire 物品容器 → 后台响应行（补齐物品名称/占格等展示信息）。
func GmItemResp(playerID int64, e *pb.ItemContainer) dto.GmItemResp {
	items := make([]dto.GmItemEntry, 0, len(e.GetItemEntries()))
	for _, it := range e.GetItemEntries() {
		w, h := model.GetWHByItemCfgId(it.GetCfgId())
		if it.GetRotated() {
			w, h = h, w
		}
		row := dto.GmItemEntry{
			ItemID:    it.GetId(),
			ItemCfgID: it.GetCfgId(),
			Count:     it.GetCount(),
			GridX:     it.GetGridX(),
			GridY:     it.GetGridY(),
			Rotated:   it.GetRotated(),
			Width:     w,
			Height:    h,
		}
		if cfg := tpl.T().Item.Get(it.GetCfgId()); cfg != nil {
			row.Name = cfg.Name
			row.IconKey = cfg.IconKey
			row.Rarity = cfg.Rarity
			row.MaxCount = cfg.MaxCount
		}
		items = append(items, row)
	}
	return dto.GmItemResp{
		PlayerID:      playerID,
		ContainerType: int32(e.GetType()),
		GridW:         e.GetGridW(),
		GridH:         e.GetGridH(),
		Version:       e.GetVersion(),
		Items:         items,
	}
}

// GmItemResps wire 物品容器列表 → 后台响应行列表。
func GmItemResps(playerID int64, containers []*pb.ItemContainer) []dto.GmItemResp {
	return util.MapSlice(containers, func(e *pb.ItemContainer) dto.GmItemResp {
		return GmItemResp(playerID, e)
	})
}

// PbKickPlayerReq 玩家ID → 踢下线 wire 请求。
func PbKickPlayerReq(playerID int64) *pb.KickPlayerReq {
	return &pb.KickPlayerReq{PlayerIds: []int64{playerID}}
}

// PbGmMailListReq 后台邮件列表请求 → wire 请求。
func PbGmMailListReq(req dto.GmPlayerReq) *pb.GmMailListReq {
	return &pb.GmMailListReq{PlayerId: req.PlayerID}
}

// PbGmMailSendReq 后台发送邮件请求 → wire 请求。
func PbGmMailSendReq(req dto.GmMailSendReq) *pb.GmMailSendReq {
	return &pb.GmMailSendReq{
		PlayerId:    req.PlayerID,
		Title:       req.Title,
		Content:     req.Content,
		Attachments: util.MapSlice(req.Attachments, gmMailAttachmentEntry),
	}
}

// gmMailAttachmentEntry 后台邮件附件 → wire 资产条目。
func gmMailAttachmentEntry(a dto.GmMailAttachment) *pb.AssetEntry {
	return &pb.AssetEntry{Type: a.Type, CfgId: a.CfgID, Count: a.Count}
}

// PbGmHeroListReq 后台英雄列表请求 → wire 请求。
func PbGmHeroListReq(req dto.GmPlayerReq) *pb.GmHeroListReq {
	return &pb.GmHeroListReq{PlayerId: req.PlayerID}
}

// PbGmHeroAddReq 后台新增英雄请求 → wire 请求。
func PbGmHeroAddReq(req dto.GmHeroAddReq) *pb.GmHeroAddReq {
	return &pb.GmHeroAddReq{PlayerId: req.PlayerID, Req: &pb.HeroAddReq{CfgId: req.HeroID}}
}

// PbGmHeroUpdateReq 后台更新英雄请求 → wire 请求。
func PbGmHeroUpdateReq(req dto.GmHeroUpdateReq) *pb.GmHeroUpdateReq {
	return &pb.GmHeroUpdateReq{
		PlayerId: req.PlayerID,
		Req: &pb.HeroUpdateReq{
			CfgId:    req.HeroID,
			Level:    req.Level,
			Exp:      req.Exp,
			Star:     req.Star,
			SkinId:   req.SkinID,
			Equipped: req.Equipped,
		},
	}
}

// PbGmHeroAttributeReq 后台查看英雄属性请求 → wire 请求。
func PbGmHeroAttributeReq(req dto.GmHeroAttributeReq) *pb.GmHeroAttributeReq {
	return &pb.GmHeroAttributeReq{PlayerId: req.PlayerID, HeroId: req.HeroID}
}

// PbGmHeroUpgradeReq 后台英雄升级请求 → wire 请求。
func PbGmHeroUpgradeReq(req dto.GmUpgradeReq) *pb.GmHeroUpgradeReq {
	return &pb.GmHeroUpgradeReq{PlayerId: req.PlayerID, HeroId: req.CfgID}
}

// PbGmEquipListReq 后台装备列表请求 → wire 请求。
func PbGmEquipListReq(req dto.GmPlayerReq) *pb.GmEquipListReq {
	return &pb.GmEquipListReq{PlayerId: req.PlayerID}
}

// PbGmEquipAddReq 后台新增装备请求 → wire 请求。
func PbGmEquipAddReq(req dto.GmEquipAddReq) *pb.GmEquipAddReq {
	return &pb.GmEquipAddReq{
		PlayerId: req.PlayerID,
		Req:      &pb.EquipAddReq{CfgId: req.EquipID, HeroId: req.HeroID},
	}
}

// PbGmEquipUpdateReq 后台更新装备请求 → wire 请求。
func PbGmEquipUpdateReq(req dto.GmEquipUpdateReq) *pb.GmEquipUpdateReq {
	return &pb.GmEquipUpdateReq{
		PlayerId: req.PlayerID,
		Req: &pb.EquipUpdateReq{
			Id:           req.ID,
			HeroId:       req.HeroID,
			Level:        req.Level,
			Quality:      req.Quality,
			Breakthrough: req.Breakthrough,
			Equipped:     req.Equipped,
		},
	}
}

// PbGmEquipWearReq 后台装备穿戴请求 → wire 请求。
func PbGmEquipWearReq(req dto.GmEquipWearReq) *pb.GmEquipWearReq {
	return &pb.GmEquipWearReq{PlayerId: req.PlayerID, EquipId: req.ID, HeroId: req.HeroID}
}

// PbGmEquipUnWearReq 后台装备卸载请求 → wire 请求。
func PbGmEquipUnWearReq(req dto.GmEquipUnWearReq) *pb.GmEquipUnWearReq {
	return &pb.GmEquipUnWearReq{PlayerId: req.PlayerID, EquipId: req.ID}
}

// PbGmEquipUpgradeReq 后台装备升级请求 → wire 请求。
func PbGmEquipUpgradeReq(req dto.GmUpgradeReq) *pb.GmEquipUpgradeReq {
	return &pb.GmEquipUpgradeReq{PlayerId: req.PlayerID, EquipId: req.EquipID}
}

// PbGmEquipUpQualityReq 后台装备升品请求 → wire 请求。
func PbGmEquipUpQualityReq(req dto.GmUpgradeReq) *pb.GmEquipUpQualityReq {
	return &pb.GmEquipUpQualityReq{PlayerId: req.PlayerID, EquipId: req.EquipID}
}

// PbGmEquipBreakthroughReq 后台装备突破请求 → wire 请求。
func PbGmEquipBreakthroughReq(req dto.GmUpgradeReq) *pb.GmEquipBreakthroughReq {
	return &pb.GmEquipBreakthroughReq{PlayerId: req.PlayerID, EquipId: req.EquipID}
}

// PbGmItemListReq 后台物品容器列表请求 → wire 请求。
func PbGmItemListReq(req dto.GmItemListReq) *pb.GmItemListReq {
	return &pb.GmItemListReq{
		PlayerId: req.PlayerID,
		Req:      &pb.ItemContainerListReq{ContainerType: pb.ContainerType(req.ContainerType)},
	}
}

// PbGmItemMoveReq 后台物品移动请求 → wire 请求。
func PbGmItemMoveReq(req dto.GmItemMoveReq) *pb.GmItemMoveReq {
	return &pb.GmItemMoveReq{
		PlayerId: req.PlayerID,
		Req: &pb.ItemMoveReq{
			ContainerType: pb.ContainerType(req.ContainerType),
			Id:            req.ItemID,
			ToX:           req.ToX,
			ToY:           req.ToY,
			Rotated:       req.Rotated,
		},
	}
}

// PbGmItemRotateReq 后台物品旋转请求 → wire 请求。
func PbGmItemRotateReq(req dto.GmItemRotateReq) *pb.GmItemRotateReq {
	return &pb.GmItemRotateReq{
		PlayerId: req.PlayerID,
		Req: &pb.ItemRotateReq{
			ContainerType: pb.ContainerType(req.ContainerType),
			Id:            req.ItemID,
		},
	}
}

// PbGmItemSortReq 后台物品整理请求 → wire 请求。
func PbGmItemSortReq(req dto.GmItemSortReq) *pb.GmItemSortReq {
	return &pb.GmItemSortReq{
		PlayerId: req.PlayerID,
		Req:      &pb.ItemSortReq{ContainerType: pb.ContainerType(req.ContainerType)},
	}
}

// PbGmItemTransferReq 后台物品容器间移动请求 → wire 请求。
func PbGmItemTransferReq(req dto.GmItemTransferReq) *pb.GmItemTransferReq {
	return &pb.GmItemTransferReq{
		PlayerId: req.PlayerID,
		Req: &pb.ItemTransferReq{
			FromType: pb.ContainerType(req.FromType),
			ToType:   pb.ContainerType(req.ToType),
			Id:       req.ItemID,
			ToX:      req.ToX,
			ToY:      req.ToY,
			Rotated:  req.Rotated,
		},
	}
}

// PbGmItemAddEntryReq 后台放入物品请求 → wire 请求。
func PbGmItemAddEntryReq(req dto.GmItemAddEntryReq) *pb.GmItemAddEntryReq {
	return &pb.GmItemAddEntryReq{
		PlayerId:      req.PlayerID,
		ContainerType: req.ContainerType,
		CfgId:         req.ItemCfgID,
		Count:         req.Count,
	}
}

// PbGmItemClearReq 后台清除物品请求 → wire 请求。
func PbGmItemClearReq(req dto.GmItemClearReq) *pb.GmItemClearReq {
	return &pb.GmItemClearReq{
		PlayerId:      req.PlayerID,
		ContainerType: req.ContainerType,
		ItemId:        req.ItemID,
	}
}
