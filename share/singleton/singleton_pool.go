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
func (this *Container[T]) Register(inst T) {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.inst = inst
}

// MustGet 获取实例，未注册或注入 nil 直接 panic
func (this *Container[T]) MustGet() T {
	this.mu.RLock()
	defer this.mu.RUnlock()

	// 使用 reflect 检查是否为空（兼容接口和指针）
	if reflect.ValueOf(this.inst).IsZero() || any(this.inst) == nil {
		panic(fmt.Errorf("[registry] dependency '%s' is not registered or is nil. Check your app.Init()", this.name))
	}
	return this.inst
}
