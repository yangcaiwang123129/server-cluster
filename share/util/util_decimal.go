package util

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// Decimal 定点小数：内嵌 shopspring/decimal，补齐 JSON / SQL / 文本的序列化与反序列化。
// JSON 输出为字符串（"12.3400"）而非数字，避免 JS 的 float64 精度丢失。
// 零值即 0，可直接作结构体字段与 map 值使用。
type Decimal struct {
	decimal.Decimal
}

// Zero 数值 0。
var Zero = Decimal{decimal.Zero}

// NewDecimal 从字符串解析；空串视为 0。
func NewDecimal(s string) (Decimal, error) {
	return parseDecimal(s)
}

// MustDecimal 从字符串解析，失败 panic（用于常量、配置等初始化场景）。
func MustDecimal(s string) Decimal {
	d, err := NewDecimal(s)
	if err != nil {
		panic(err)
	}
	return d
}

// FromInt64 整数构造。
func FromInt64(v int64) Decimal { return Decimal{decimal.NewFromInt(v)} }

// FromFloat64 浮点构造（按浮点最短表示转换，0.1 得 0.1 而非 0.1000000000000000055511151231257827）。
func FromFloat64(v float64) Decimal { return Decimal{decimal.NewFromFloat(v)} }

// parseDecimal 解析字符串，兼容空串与首尾空白。
func parseDecimal(s string) (Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Decimal{}, nil
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Decimal{}, fmt.Errorf("util: parse decimal %q: %w", s, err)
	}
	return Decimal{d}, nil
}

// MarshalJSON 输出带引号的字符串；nil 指针输出 null。
func (d Decimal) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON 兼容 JSON 字符串（"12.34"）、JSON 数字（12.34）与 null。
func (d *Decimal) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" {
		*d = Decimal{}
		return nil
	}
	// 字符串形式去掉包裹引号：数值不需要转义处理。
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	parsed, err := parseDecimal(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// MarshalText 文本序列化（map 键、yaml、URL 参数等场景）。
func (d Decimal) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

// UnmarshalText 文本反序列化。
func (d *Decimal) UnmarshalText(b []byte) error {
	parsed, err := parseDecimal(string(b))
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// Value 实现 driver.Valuer：以字符串写入，MySQL 按列精度自动截断/舍入。
func (d Decimal) Value() (driver.Value, error) { return d.String(), nil }

// Scan 实现 sql.Scanner：兼容 MySQL 驱动的 []byte/string 以及整型、浮点型。
func (d *Decimal) Scan(v any) error {
	switch x := v.(type) {
	case nil:
		*d = Decimal{}
	case []byte:
		parsed, err := parseDecimal(string(x))
		if err != nil {
			return err
		}
		*d = parsed
	case string:
		parsed, err := parseDecimal(x)
		if err != nil {
			return err
		}
		*d = parsed
	case int64:
		*d = Decimal{decimal.NewFromInt(x)}
	case float64:
		*d = Decimal{decimal.NewFromFloat(x)}
	default:
		return fmt.Errorf("util: cannot scan %T into Decimal", v)
	}
	return nil
}

// GormDataType 声明 GORM 字段类型；列精度仍以 struct tag 的 `gorm:"type:decimal(20,6)"` 为准。
func (d Decimal) GormDataType() string { return "decimal" }
