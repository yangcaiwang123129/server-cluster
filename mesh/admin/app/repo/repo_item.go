package repo

import (
	"errors"

	"gorm.io/gorm"

	"server-cluster/share/model"
)

// GetPlayerContainer 读取玩家容器集合；无记录返回 (nil, nil)。
func (r *Repo) GetPlayerContainer(playerID int64) (*model.PlayerContainer, error) {
	d, err := r.db()
	if err != nil {
		return nil, err
	}

	pc := &model.PlayerContainer{}
	if err := d.First(pc, "player_id = ?", playerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return pc, nil
}

// SavePlayerContainer 保存玩家容器集合（三列 JSON 整体覆盖，无记录时插入）。
func (r *Repo) SavePlayerContainer(pc *model.PlayerContainer) error {
	d, err := r.db()
	if err != nil {
		return err
	}
	return d.Save(pc).Error
}
