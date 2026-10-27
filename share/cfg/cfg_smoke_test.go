package cfg

import "testing"

// TestLoadSmoke 冒烟测试：加载配置表并验证访问 + 新方法。
func TestLoadSmoke(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	item := T().Item
	t.Logf("items=%d", item.Len())

	if !item.Has(1) {
		t.Fatal("id=1 should exist")
	}
	if item.Has(99999) {
		t.Fatal("id=99999 should not exist")
	}

	it := item.Get(1)
	t.Logf("id=1: Type=%v Name=%q", it.Type, it.Name)
	it2 := item.Get(1008)
	t.Logf("id=1008: Type=%v Name=%q", it2.Type, it2.Name)
}

// TestLoadIdempotent 确认 Load 幂等。
func TestLoadIdempotent(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if err := Load(); err != nil {
		t.Fatalf("second Load should return nil, got %v", err)
	}
}
