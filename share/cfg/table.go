package cfg

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed data
var dataFS embed.FS

// Table 是配置表的泛型容器：按 id 建 map，同时保留原始顺序的 list。
// 构造后只读，支持并发读。
type Table[T any] struct {
	dataMap  map[int32]*T
	dataList []*T
}

// Get 按 id 查询，不存在返回 nil。
func (t *Table[T]) Get(id int32) *T { return t.dataMap[id] }

// Has 判断 id 是否存在。
func (t *Table[T]) Has(id int32) bool {
	_, ok := t.dataMap[id]
	return ok
}

// Len 返回记录数。
func (t *Table[T]) Len() int { return len(t.dataMap) }

// GetDataMap 返回全部记录的 id→记录 map。
func (t *Table[T]) GetDataMap() map[int32]*T { return t.dataMap }

// GetDataList 返回按文件顺序排列的全部记录。
func (t *Table[T]) GetDataList() []*T { return t.dataList }

// LoadTable 从 data/<file>.json 加载一张表，keyFn 从记录中提取主键。
// 若出现重复 key，返回错误而非静默覆盖。
func LoadTable[T any](file string, keyFn func(*T) int32) (*Table[T], error) {
	raw, err := dataFS.ReadFile("data/" + file + ".json")
	if err != nil {
		return nil, fmt.Errorf("cfg: read %s.json: %w", file, err)
	}
	var rows []*T
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("cfg: parse %s.json: %w", file, err)
	}
	tb := &Table[T]{
		dataList: rows,
		dataMap:  make(map[int32]*T, len(rows)),
	}
	for _, r := range rows {
		k := keyFn(r)
		if _, dup := tb.dataMap[k]; dup {
			return nil, fmt.Errorf("cfg: duplicate id %d in %s.json", k, file)
		}
		tb.dataMap[k] = r
	}
	return tb, nil
}
