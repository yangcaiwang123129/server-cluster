// Package repo 数据访问层：接口 + gorm 实现 + 单例。
package repo

import (
	"context"
	"server-cluster/share/singleton"
)

// SocialRepo 社交数据访问接口。
type SocialRepo interface {
	// SaveMessage 保存聊天消息（预留）
	SaveMessage(ctx context.Context, msg interface{}) error
}

var socialRepo = singleton.NewContainer[SocialRepo]("SocialRepo")

// init 装配期自注册实现。
func init() {
	socialRepo.Register(NewSocialRepo())
}

// Social 获取社交数据访问单例。
func Social() SocialRepo { return socialRepo.MustGet() }

type socialRepoImpl struct{}

// NewSocialRepo 构造 gorm 实现。
func NewSocialRepo() SocialRepo { return &socialRepoImpl{} }

func (r *socialRepoImpl) SaveMessage(ctx context.Context, msg interface{}) error {
	return nil
}
