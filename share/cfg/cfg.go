package cfg

// Tables 持有全部配置表实例，由 Load 初始化。
type Tables struct {
	Item *Table[Item]
}

var tables *Tables

// Load 加载全部配置表；失败应终止服务启动。
func Load() error {
	var err error
	t := &Tables{}
	if t.Item, err = LoadTable("itemCfg", func(r *Item) int32 { return r.Id }); err != nil {
		return err
	}
	tables = t
	return nil
}

// T 获取全局配置表；须先 Load 成功，否则返回 nil。
func T() *Tables {
	if tables == nil {
		panic("cfg: T() called before Load()")
	}
	return tables
}
