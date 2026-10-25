// Package model 大厅业务模型：可序列化的领域结构（如背包），与存储/传输解耦。
package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"server-cluster/share/enum"
)

// BackpackItem 背包格子中的物品。
type BackpackItem struct {
	ItemID int64 `json:"item_id"`
	Count  int64 `json:"count"`
}

// Backpack 背包实体（对应 player_backpack 表，整包 JSON 序列化后落库/缓存）。
// 数组格子方式：Slots 下标即格子ID（bag slot id），nil 表示空格子；
// index 为 itemID -> 格子下标 的内存映射表，反序列化后自动重建，不随 JSON 落盘。
type Backpack struct {
	PlayerID int64           `json:"player_id"`
	Slots    []*BackpackItem `json:"slots"`
	Version  uint64          `json:"version"`

	index map[int64]int // itemID -> 格子下标（内存态）
}

// Marshal 序列化背包（JSON，仅持久化字段）。
func (b *Backpack) Marshal() ([]byte, error) { return json.Marshal(b) }

// Unmarshal 反序列化背包并重建 itemID -> 格子下标 映射表。
func (b *Backpack) Unmarshal(data []byte) error {
	if err := json.Unmarshal(data, b); err != nil {
		return err
	}
	b.RebuildIndex()
	return nil
}

// Scan sql.Scanner：DB 列值（[]byte/string）JSON 解码并重建映射表。
func (b *Backpack) Scan(src any) error {
	if err := ScanJSON(src, b); err != nil {
		return err
	}
	b.RebuildIndex()
	return nil
}

// Value driver.Valuer：整包 JSON 编码（供 BackpackRow.data 列落库）。
func (b Backpack) Value() (driver.Value, error) { return ValueJSON(b) }

// BackpackRow player_backpack 表行（gorm 自动建表对象）：data 列存整包 JSON。
type BackpackRow struct {
	PlayerID int64    `gorm:"primaryKey;column:player_id" json:"player_id"`
	Data     Backpack `gorm:"column:data;type:json"       json:"data"`
	Version  uint64   `gorm:"column:version"              json:"version"`
}

// TableName 指定表名。
func (BackpackRow) TableName() string { return enum.TablePlayerBackpack.String() }

// RebuildIndex 按 Slots 重建 itemID -> 格子下标 映射表（加载后或外部直接改 Slots 后调用）。
func (b *Backpack) RebuildIndex() {
	b.index = make(map[int64]int, len(b.Slots))
	for slot, item := range b.Slots {
		if item != nil {
			b.index[item.ItemID] = slot
		}
	}
}

// SlotOf 按 itemID 查格子下标；不存在返回 ok=false。
func (b *Backpack) SlotOf(itemID int64) (slot int, ok bool) {
	if b.index == nil {
		b.RebuildIndex()
	}
	slot, ok = b.index[itemID]
	return slot, ok
}

// Get 按下标取格子物品；越界或空格返回 nil。
func (b *Backpack) Get(slot int) *BackpackItem {
	if slot < 0 || slot >= len(b.Slots) {
		return nil
	}
	return b.Slots[slot]
}

// Set 在指定格子放置物品（nil 表示清空该格），同步维护映射表；越界返回错误。
func (b *Backpack) Set(slot int, item *BackpackItem) error {
	if slot < 0 || slot >= len(b.Slots) {
		return fmt.Errorf("model: backpack slot %d out of range %d", slot, len(b.Slots))
	}
	if b.index == nil {
		b.RebuildIndex()
	}
	if old := b.Slots[slot]; old != nil {
		delete(b.index, old.ItemID)
	}
	b.Slots[slot] = item
	if item != nil {
		b.index[item.ItemID] = slot
	}
	return nil
}

// Add 放入物品：同 itemID 已存在则堆叠数量，否则占第一个空格子；无空格则末尾扩容。
// 返回物品所在格子下标。
func (b *Backpack) Add(itemID, count int64) int {
	if b.index == nil {
		b.RebuildIndex()
	}
	if slot, ok := b.index[itemID]; ok {
		b.Slots[slot].Count += count
		return slot
	}
	for slot, item := range b.Slots {
		if item == nil {
			b.Slots[slot] = &BackpackItem{ItemID: itemID, Count: count}
			b.index[itemID] = slot
			return slot
		}
	}
	b.Slots = append(b.Slots, &BackpackItem{ItemID: itemID, Count: count})
	slot := len(b.Slots) - 1
	b.index[itemID] = slot
	return slot
}
