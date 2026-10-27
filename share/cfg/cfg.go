package cfg

import "sync"

// Tables 持有全部配置表实例，由 Load 初始化。
type Tables struct {
	Item *Table[ItemCfg]
}

var (
	tables *Tables
	loadMu sync.Mutex
	loaded bool
)

// Load 加载全部配置表；幂等（重复调用直接返回 nil）。
// 调用顺序决定表间可引用关系（目前表无互相依赖）。
func Load() error {
	loadMu.Lock()
	defer loadMu.Unlock()
	if loaded {
		return nil
	}
	t := &Tables{}
	var err error
	if t.Item, err = LoadTable("itemconfig", func(r *ItemCfg) int32 { return r.Id }); err != nil {
		return err
	}
	tables = t
	loaded = true
	return nil
}

// MustLoad 同 Load，但失败直接 panic；适合 init() 阶段使用。
func MustLoad() {
	if err := Load(); err != nil {
		panic("cfg: MustLoad: " + err.Error())
	}
}

// T 获取全局配置表；须先 Load 成功，否则 panic。
func T() *Tables {
	if tables == nil {
		panic("cfg: T() called before Load()/MustLoad()")
	}
	return tables
}

// Loaded 是否已加载。
func Loaded() bool { return loaded }
