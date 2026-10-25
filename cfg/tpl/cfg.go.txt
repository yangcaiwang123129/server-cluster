package cfg

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed data
var dataFS embed.FS

var tables *Tables

// Load 加载全部配置表；失败应终止服务启动。
func Load() error {
	t, err := NewTables(jsonLoader)
	if err != nil {
		return fmt.Errorf("cfg: load tables: %w", err)
	}
	tables = t
	return nil
}

// T 获取全局配置表；须先 Load 成功，否则返回 nil。
func T() *Tables { return tables }

// jsonLoader 实现生成代码要求的 JsonLoader：读取内嵌 data 目录下的 <file>.json。
func jsonLoader(file string) ([]map[string]interface{}, error) {
	raw, err := dataFS.ReadFile("data/" + file + ".json")
	if err != nil {
		return nil, fmt.Errorf("cfg: read %s.json: %w", file, err)
	}
	rows := make([]map[string]interface{}, 0)
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("cfg: parse %s.json: %w", file, err)
	}
	return rows, nil
}
