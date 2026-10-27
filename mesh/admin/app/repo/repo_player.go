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

// ListPlayers 玩家分页列表（直查 player_base）。
func (r *Repo) ListPlayers(q PlayerQuery, pageIndex, pageSize int) ([]dto.PlayerResp, int64, error) {
	d, err := r.db()
	if err != nil {
		return nil, 0, err
	}
	tx := d.Model(&model.PlayerBase{})
	if q.Username != "" {
		tx = tx.Where("username LIKE ?", util.Like(q.Username))
	}
	rows, total, err := util.PageQuery[model.PlayerBase](tx, pageIndex, pageSize, q.Order, "player_id desc")
	if err != nil {
		return nil, 0, err
	}
	list := make([]dto.PlayerResp, 0, len(rows))
	for i := range rows {
		m := &rows[i]
		list = append(list, dto.PlayerResp{
			PlayerID:    m.PlayerID,
			Username:    m.Username,
			Zone:        m.Zone,
			PhoneNumber: m.PhoneNumber,
			CreatedAt:   m.CreatedAt,
		})
	}
	return list, total, nil
}
