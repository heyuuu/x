package di

import (
	"fmt"
	"iter"
	"reflect"
)

// requireFrozen 读操作的前置校验：未冻结就读属于装配期 bug，直接 panic。
func requireFrozen(c *Container) {
	if !c.frozen {
		panic("di: 容器尚未冻结, 先调用 Freeze()")
	}
}

// Get 按注册类型精确解析：命中返回 (实例, true)，未注册返回 (零值, false)；
// 容器未冻结则 panic（代码 bug）。
func Get[T any](c *Container) (T, bool) {
	requireFrozen(c)

	typ := reflect.TypeFor[T]()
	instance, ok := c.instances[typ]
	if !ok {
		var zero T
		return zero, false
	}

	return instance.(T), true
}

// MustGet 从容器中获取 T 类型的服务实例，未注册时 panic。
func MustGet[T any](c *Container) T {
	instance, ok := Get[T](c)
	if !ok {
		panic(fmt.Errorf("di: 服务 %s 未注册", reflect.TypeFor[T]().Name()))
	}
	return instance
}

// All 遍历容器内全部服务，产出 (注册类型, 实例) 对，按类型名稳定排序；容器未冻结则 panic。
func All(c *Container) iter.Seq2[reflect.Type, any] {
	requireFrozen(c)

	return func(yield func(reflect.Type, any) bool) {
		for _, typ := range c.sortedTypes() {
			if !yield(typ, c.instances[typ]) {
				return
			}
		}
	}
}

// AllOf 遍历注册类型可赋值给 T 的全部服务，按类型名稳定排序；容器未冻结则 panic。
func AllOf[T any](c *Container) iter.Seq[T] {
	requireFrozen(c)

	targetType := reflect.TypeFor[T]()
	return func(yield func(T) bool) {
		for _, typ := range c.sortedTypes() {
			// Implements 只接受接口类型，T 为具体类型时会 panic，故用 AssignableTo
			if typ.AssignableTo(targetType) && !yield(c.instances[typ].(T)) {
				return
			}
		}
	}
}
