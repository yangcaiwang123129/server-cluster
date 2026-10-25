package store

import "errors"

// 存储层错误。实体定义见 share/model。
var (
	// ErrNotFound 玩家数据不存在（全层 miss）。
	ErrNotFound = errors.New("store: player data not found")
	// ErrInsufficientAsset 资产数量不足（增量扣减后小于 0）。
	ErrInsufficientAsset = errors.New("store: insufficient asset amount")
)
