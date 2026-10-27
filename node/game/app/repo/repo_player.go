// Package repo 数据访问层：DB / Redis 读写，向上（logic）只暴露接口。
// 单例容器在本包，init 自注册（替代 app/wire）。
package repo

import (
	"context"
	"errors"
	"server-cluster/node/game/app/global"

	"gorm.io/gorm"

	"server-cluster/internal/svc"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/singleton"
)

var playerRepo = singleton.NewContainer[global.PlayerRepo]("PlayerRepo")

// init 装配期自注册实现。
func init() {
	playerRepo.Register(NewPlayerRepo())
}

// Player 获取玩家数据访问单例。
func Player() global.PlayerRepo { return playerRepo.MustGet() }

type playerRepoImpl struct{}

// NewPlayerRepo 创建玩家数据访问实例。
func NewPlayerRepo() global.PlayerRepo { return &playerRepoImpl{} }

func (r *playerRepoImpl) Load(ctx context.Context, playerID int64) (*model.PlayerModel, error) {
	gdb, err := svc.Ctx().DB()
	if err != nil {
		return nil, errs.ErrInternal
	}
	var m model.PlayerModel
	err = gdb.WithContext(ctx).First(&m, "player_id = ?", playerID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.ErrPlayerNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *playerRepoImpl) Save(ctx context.Context, p *model.PlayerModel) error {
	gdb, err := svc.Ctx().DB()
	if err != nil {
		return errs.ErrInternal
	}
	return gdb.WithContext(ctx).Save(p).Error
}
