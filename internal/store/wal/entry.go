// Package wal 写前日志封装：基于 tidwall/wal，职责收敛为顺序写、按 seq 回放、前缀截断。
package wal

import "encoding/json"

// WAL 操作类型。
const (
	OpSetCore    = "set_core"
	OpSetAssets  = "set_assets"
	OpSetProfile = "set_profile"
	OpDelete     = "delete"
)

// Entry WAL 条目。Seq 为实例本地单调恢复序号（非跨实例版本号）；
// Version 为跨实例版本号，用于 MySQL 乐观锁比较新旧。
type Entry struct {
	Seq       uint64 `json:"seq"`
	Op        string `json:"op"`
	PlayerID  int64  `json:"player_id"`
	Version   uint64 `json:"version"`
	Payload   []byte `json:"payload,omitempty"`
	CreatedAt int64  `json:"created_at"` // UnixMilli
}

// Encode 序列化条目（JSON）。
func Encode(e *Entry) ([]byte, error) { return json.Marshal(e) }

// Decode 反序列化条目。
func Decode(data []byte) (*Entry, error) {
	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}
