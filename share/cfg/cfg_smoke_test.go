package cfg

import "testing"

// TestLoadSmoke 冒烟测试：加载全部配置表并抽查关键记录。
func TestLoadSmoke(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	w := T().TbWeapon.Get(2)
	t.Logf("weapon id=2: %+v", w)
	e := T().TbEnemy.Get(2003)
	t.Logf("enemy id=2003: %+v", e)
	t.Logf("items=%d bullets=%d", len(T().TbItem.GetDataList()), len(T().TbBullet.GetDataList()))
}
