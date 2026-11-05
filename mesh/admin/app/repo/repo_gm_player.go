package repo

import (
	"server-cluster/share/consts"
	"server-cluster/share/dto"
	"server-cluster/share/model"
	"server-cluster/share/util"
)

// PlayerQuery 玩家列表过滤条件。
type PlayerQuery struct {
	PlayerID       int64
	Username       string
	PhoneNumber    string // 手机号，字符串支持前缀/模糊
	Status         int32  // 0 全部 1 正常 2 封号
	CreatedAtStart int64  // 毫秒
	CreatedAtEnd   int64  // 毫秒
	Order          string
}

// ListPlayers 玩家分页列表，username/phone 从 player_auth 表批量补齐。
func (r *Repo) ListPlayers(q PlayerQuery, pageIndex, pageSize int) ([]dto.PlayerResp, int64, error) {
	d, err := r.db()
	if err != nil {
		return nil, 0, err
	}
	tx := d.Model(&model.Player{})

	// ──── 过滤条件 ────
	if q.PlayerID > 0 {
		tx = tx.Where("player_id = ?", q.PlayerID)
	}
	if q.Username != "" {
		// 子查询：先查 player_auth(type=1) 拿到匹配 player_id
		sub := d.Model(&model.PlayerAuth{}).
			Select("player_id").
			Where("auth_type = ? AND auth_identifier LIKE ?", consts.LoginTypeAccount, util.Like(q.Username))
		tx = tx.Where("player_id IN (?)", sub)
	}
	if q.PhoneNumber != "" {
		sub := d.Model(&model.PlayerAuth{}).
			Select("player_id").
			Where("auth_type = ? AND auth_identifier LIKE ?", consts.LoginTypePhoneSMS, util.Like(q.PhoneNumber))
		tx = tx.Where("player_id IN (?)", sub)
	}
	if q.Status > 0 {
		tx = tx.Where("status = ?", q.Status)
	}
	if q.CreatedAtStart > 0 {
		tx = tx.Where("created_at >= ?", q.CreatedAtStart)
	}
	if q.CreatedAtEnd > 0 {
		tx = tx.Where("created_at <= ?", q.CreatedAtEnd)
	}

	rows, total, err := util.PageQuery[model.Player](tx, pageIndex, pageSize, q.Order, "player_id desc")
	if err != nil {
		return nil, 0, err
	}

	// ──── 批量补齐 player_extra + player_auth ────
	ids := util.MapSlice(rows, func(m model.Player) int64 { return m.PlayerID })

	// player_extra (昵称/头像/等级/VIP)
	extras := map[int64]*model.PlayerExtra{}
	// player_auth (username / phone / wechat / apple)
	auths := map[int64]map[int32]*model.PlayerAuth{} // player_id → auth_type → row

	if len(ids) > 0 {
		var extraRows []model.PlayerExtra
		if err := d.Model(&model.PlayerExtra{}).Where("player_id IN ?", ids).Find(&extraRows).Error; err != nil {
			return nil, 0, err
		}
		for i := range extraRows {
			extras[extraRows[i].PlayerID] = &extraRows[i]
		}

		var authRows []model.PlayerAuth
		if err := d.Model(&model.PlayerAuth{}).Where("player_id IN ?", ids).Find(&authRows).Error; err != nil {
			return nil, 0, err
		}
		for i := range authRows {
			aid := authRows[i].PlayerID
			if _, ok := auths[aid]; !ok {
				auths[aid] = map[int32]*model.PlayerAuth{}
			}
			auths[aid][authRows[i].AuthType] = &authRows[i]
		}
	}

	// ──── 组装响应 ────
	list := make([]dto.PlayerResp, 0, len(rows))
	for i := range rows {
		m := &rows[i]
		row := dto.PlayerResp{
			PlayerID:   m.PlayerID,
			Zone:       m.Zone,
			Coupon:     m.Coupon,
			Gold:       m.Gold,
			Coin:       m.Coin,
			RegisterIP: m.RegisterIP,
			CreatedAt:  m.CreatedAt,
			Status:     m.Status,
			BanUntil:   m.BanUntil,
			BanReason:  m.BanReason,
		}
		// 从 player_auth 补 username / phone
		if aMap, ok := auths[m.PlayerID]; ok {
			if a, ok := aMap[consts.LoginTypeAccount]; ok {
				row.Username = a.AuthIdentifier
			}
			if a, ok := aMap[consts.LoginTypePhoneSMS]; ok {
				row.PhoneNumber = a.AuthIdentifier
			}
		}
		// 从 player_extra 补昵称/头像/等级/VIP
		if extra, ok := extras[m.PlayerID]; ok {
			row.Nickname = extra.Nickname
			row.AvatarID = extra.AvatarID
			row.Level = extra.Level
			row.VipLevel = extra.VipLevel
		}
		list = append(list, row)
	}
	return list, total, nil
}

// BanPlayer 封号。
func (r *Repo) BanPlayer(playerID int64, banUntil int64, reason string) error {
	d, err := r.db()
	if err != nil {
		return err
	}
	return d.Model(&model.Player{}).Where("player_id = ?", playerID).Updates(map[string]any{
		"status":     model.PlayerStatusBanned,
		"ban_until":  banUntil,
		"ban_reason": reason,
	}).Error
}

// UnbanPlayer 解封。
func (r *Repo) UnbanPlayer(playerID int64) error {
	d, err := r.db()
	if err != nil {
		return err
	}
	return d.Model(&model.Player{}).Where("player_id = ?", playerID).Updates(map[string]any{
		"status":     model.PlayerStatusNormal,
		"ban_until":  0,
		"ban_reason": "",
	}).Error
}

// ResetPlayerPassword 重置玩家密码：找 player_auth(type=1) 更新 auth_secret。
func (r *Repo) ResetPlayerPassword(playerID int64, passwordHash string) error {
	d, err := r.db()
	if err != nil {
		return err
	}
	return d.Model(&model.PlayerAuth{}).
		Where("player_id = ? AND auth_type = ?", playerID, consts.LoginTypeAccount).
		Update("auth_secret", passwordHash).Error
}
