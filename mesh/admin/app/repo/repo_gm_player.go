package repo

import (
	"server-cluster/share/dto"
	"server-cluster/share/model"
	"server-cluster/share/util"
)

// PlayerQuery 玩家列表过滤条件。
type PlayerQuery struct {
	Username string
	Order    string
}

// ListPlayers 玩家分页列表（直查 player）。
func (r *Repo) ListPlayers(q PlayerQuery, pageIndex, pageSize int) ([]dto.PlayerResp, int64, error) {
	d, err := r.db()
	if err != nil {
		return nil, 0, err
	}
	tx := d.Model(&model.Player{})
	if q.Username != "" {
		tx = tx.Where("username LIKE ?", util.Like(q.Username))
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
			CreatedAt:   m.CreatedAt,
		}
		if extra, ok := extras[m.PlayerID]; ok {
			row.Nickname = extra.Nickname
		}
		list = append(list, row)
	}
	return list, total, nil
}
