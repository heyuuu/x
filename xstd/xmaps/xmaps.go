package xmaps

import (
	"cmp"
	"slices"
)

// Keys 以 slice 形式返回所有键，不保证结果有序
// 功能等价于 slices.Collect(maps.Keys(m))，但不用 iter 性能更好
func Keys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}

// Values 以 slice 形式返回所有值，不保证结果有序
// 功能等价于 slices.Collect(maps.Values(m))，但不用 iter 性能更好
func Values[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}

// SortedKeys 以 slice 形式返回所有键，结果按默认规则排序
func SortedKeys[M ~map[K]V, K cmp.Ordered, V any](m M) []K {
	keys := Keys(m)
	slices.Sort(keys)
	return keys
}

// GetOrDefault 取 key 对应的值，不存在时返回 defaultValue。
// 与直接下标 m[key] 的区别：能区分"存的是零值"与"不存在"。
func GetOrDefault[M ~map[K]V, K comparable, V any](m M, key K, defaultValue V) V {
	if value, exists := m[key]; exists {
		return value
	}

	return defaultValue
}

// GetOrElse 同 GetOrDefault，但默认值惰性求值：key 不存在时才调用 defaultValue。
func GetOrElse[M ~map[K]V, K comparable, V any](m M, key K, defaultValue func() V) V {
	if value, exists := m[key]; exists {
		return value
	}

	return defaultValue()
}

// GetOrPut 取 key 对应的值，不存在时调用 defaultValue 计算并写入 m 后返回。
// m 为 nil 时按原生写语义 panic，不做静默兜底：写 nil map 属程序错误。
func GetOrPut[M ~map[K]V, K comparable, V any](m M, key K, defaultValue func() V) V {
	if value, exists := m[key]; exists {
		return value
	}

	value := defaultValue()
	// 此处若 m 为 nil，直接触发原生 nil map panic
	m[key] = value
	return value
}
