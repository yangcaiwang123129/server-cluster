package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
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

// TableModel 玩家表公共字段：乐观锁版本号 + 毫秒时间戳，以 gorm:"embedded" 展开为三列。
//
// 自动维护（无需手动设置）：
//   - BeforeCreate: Version=0, CreatedAt/UpdatedAt=now
//   - BeforeUpdate: Version+=1, UpdatedAt=now, CreatedAt 不变
//
// cache 层批量 upsert（OnConflict.Create）走 BeforeCreate：通过 Version 是否 > 0
// 区分新建行与冲突更新行，冲突行也会自动 Version+=1。
//
// 所有嵌入 TableModel 的玩家表自动获得这些 hook（Go 方法提升）。
type TableModel struct {
	Version   uint64 `gorm:"column:version;type:bigint unsigned;not null;default:0;comment:版本号(乐观锁)" json:"version"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间(毫秒时间戳)" json:"created_at"`
	UpdatedAt int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间(毫秒时间戳)" json:"updated_at"`
}

// BeforeCreate GORM Hook：新建 → Version=0, CreatedAt/UpdatedAt=now；
// cache upsert 冲突行（Version>0）→ Version+=1, UpdatedAt=now, CreatedAt 保持。
func (t *TableModel) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().UnixMilli()
	if t.Version == 0 {
		// 全新创建行 —— CreatedAt/UpdatedAt 都设 now，Version 保持 0
		t.CreatedAt = now
		t.UpdatedAt = now
	} else {
		// upsert 冲突更新行 —— Version+=1，UpdatedAt 刷新，CreatedAt 不动
		t.Version += 1
		t.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM Hook：常规 Update/Save → Version+=1, UpdatedAt=now, CreatedAt 不变。
func (t *TableModel) BeforeUpdate(tx *gorm.DB) error {
	t.Version += 1
	t.UpdatedAt = time.Now().UnixMilli()
	return nil
}

func (this *TableModel) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *TableModel) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *TableModel) Scan(src any) error          { return ScanJSON(src, this) }
func (this TableModel) Value() (driver.Value, error) { return ValueJSON(this) }
func (this TableModel) TableName() string {
	return ""
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
