package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Serializable 实体序列化契约：JSON 编解码 + gorm Scan/Value（整实体 JSON 落库/缓存的统一出入口）。
type Serializable interface {
	// Marshal 序列化为 JSON。
	Marshal() ([]byte, error)
	// Unmarshal 从 JSON 反序列化。
	Unmarshal(data []byte) error
	// Scan sql.Scanner：从 DB 列值（[]byte/string）JSON 解码。
	Scan(src any) error
	// Value driver.Valuer：JSON 编码为 DB 列值。
	Value() (driver.Value, error)
}

// Model 表模型：序列化契约 + 表名（gorm 自动建表的对象）。
type Model interface {
	Serializable
	TableName() string
}

// ScanJSON sql.Scanner 通用实现：src 为 []byte/string 时按 JSON 解码到 dst。
func ScanJSON(src, dst any) error {
	if src == nil {
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("model: unsupported scan type %T", src)
	}
	return json.Unmarshal(data, dst)
}

// ValueJSON driver.Valuer 通用实现：JSON 编码。
func ValueJSON(v any) (driver.Value, error) { return json.Marshal(v) }
