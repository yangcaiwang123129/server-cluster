package repo

import (
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

// ListPlayers 玩家分页列表（直查 player）。
func (r *Repo) ListPlayers(q PlayerQuery, pageIndex, pageSize int) ([]dto.PlayerResp, int64, error) {
	d, err := r.db()
	if err != nil {
		return nil, 0, err
	}
	tx := d.Model(&model.Player{})
	if q.PlayerID > 0 {
		tx = tx.Where("player_id = ?", q.PlayerID)
	}
	if q.Username != "" {
		tx = tx.Where("username LIKE ?", util.Like(q.Username))
	}
	if q.PhoneNumber != "" {
		// phone_number 是 int，用 CAST 到字符串做 LIKE 前缀匹配
		tx = tx.Where("CAST(phone_number AS CHAR) LIKE ?", util.Like(q.PhoneNumber))
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

	// 本页玩家ID，批量补齐昵称（player_extra），避免逐行查询
	ids := make([]int64, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].PlayerID)
	}
	extras := map[int64]*model.PlayerExtra{}
	if len(ids) > 0 {
		var extraRows []model.PlayerExtra
		if err := d.Model(&model.PlayerExtra{}).Where("player_id IN ?", ids).Find(&extraRows).Error; err != nil {
			return nil, 0, err
		}
		for i := range extraRows {
			extras[extraRows[i].PlayerID] = &extraRows[i]
		}
	}

	list := make([]dto.PlayerResp, 0, len(rows))
	for i := range rows {
		m := &rows[i]
		row := dto.PlayerResp{
			PlayerID:    m.PlayerID,
			Username:    m.Username,
			Zone:        m.Zone,
			PhoneNumber: m.PhoneNumber,
			Coupon:      m.Coupon,
			Gold:        m.Gold,
			Coin:        m.Coin,
			RegisterIP:  m.RegisterIP,
			CreatedAt:   m.CreatedAt,
			Status:      m.Status,
			BanUntil:    m.BanUntil,
			BanReason:   m.BanReason,
		}
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

// BanPlayer 封号：更新状态+封号到期时间+原因。
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

// UnbanPlayer 解封：恢复状态，清空封号字段。
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

// ResetPlayerPassword 重置玩家密码：参数为明文哈希值，直接写表。
func (r *Repo) ResetPlayerPassword(playerID int64, passwordHash string) error {
	d, err := r.db()
	if err != nil {
		return err
	}
	return d.Model(&model.Player{}).Where("player_id = ?", playerID).Update("password", passwordHash).Error
}
