// Package repo 数据访问层：DB / Redis 读写，向上（logic）只暴露接口。
package repo

import (
	"context"
	"errors"
	"server-cluster/share/singleton"

	"gorm.io/gorm"

	"server-cluster/internal/svc"
	"server-cluster/share/errs"
	"server-cluster/share/model"
)

// PlayerRepo 玩家数据访问接口。
type PlayerRepo interface {
	// Load 按 uid 加载玩家；不存在返回 errs.ErrPlayerNotFound。
	Load(ctx context.Context, uid int64) (*model.PlayerModel, error)
	// Save 新建或更新玩家。
	Save(ctx context.Context, p *model.PlayerModel) error
}

var playerRepo = singleton.NewContainer[PlayerRepo]("PlayerRepo")

// RegisterPlayer 装配期注册实现；仅 app/wire.go 调用。
func RegisterPlayer(r PlayerRepo) { playerRepo.Register(r) }

// Player 获取玩家数据访问单例。
func Player() PlayerRepo { return playerRepo.MustGet() }

type playerRepoImpl struct{}

// NewPlayerRepo 创建玩家数据访问实例。
func NewPlayerRepo() PlayerRepo { return &playerRepoImpl{} }

func (r *playerRepoImpl) Load(ctx context.Context, uid int64) (*model.PlayerModel, error) {
	gdb, err := svc.Ctx().DB()
	if err != nil {
		return nil, errs.ErrInternal
	}
	var m model.PlayerModel
	err = gdb.WithContext(ctx).First(&m, "uid = ?", uid).Error
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
