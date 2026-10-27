package cfg

import "testing"

// TestLoadSmoke 冒烟测试：加载全部配置表并抽查关键记录。
func TestLoadSmoke(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	t.Logf("items=%d", len(T().Item.GetDataList()))
	it := T().Item.Get(1)
	t.Logf("item id=1: %+v", it)
	it2 := T().Item.Get(1008)
	t.Logf("item id=1008: %+v", it2)
}
