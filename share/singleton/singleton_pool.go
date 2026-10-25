package singleton

import (
	"fmt"
	"reflect"
	"sync"
)

// Container 泛型单例容器
type Container[T any] struct {
	mu   sync.RWMutex
	inst T
	name string
}

// NewContainer 创建一个容器，传入 name 方便定位未注册的模块
func NewContainer[T any](name string) *Container[T] {
	return &Container[T]{name: name}
}

// Register 注册实例
func (c *Container[T]) Register(inst T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inst = inst
}

// MustGet 获取实例，未注册或注入 nil 直接 panic
func (c *Container[T]) MustGet() T {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 使用 reflect 检查是否为空（兼容接口和指针）
	if reflect.ValueOf(c.inst).IsZero() || any(c.inst) == nil {
		panic(fmt.Errorf("[registry] dependency '%s' is not registered or is nil. Check your app.Init()", c.name))
	}
	return c.inst
}
