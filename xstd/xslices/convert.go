package xslices

// ToSet 将切片转为集合：以 f 计算的 key 作 map 的键，重复 key 自然去重。
func ToSet[S ~[]E, E any, K comparable](s S, f func(E) K) map[K]struct{} {
	set := make(map[K]struct{}, len(s))
	for _, item := range s {
		set[f(item)] = struct{}{}
	}
	return set
}

// ToBoolSet 类似 ToSet, 不过值恒为 true, 可直接以下标判断成员关系：m[key]。
func ToBoolSet[S ~[]E, E any, K comparable](s S, f func(E) K) map[K]bool {
	set := make(map[K]bool, len(s))
	for _, item := range s {
		set[f(item)] = true
	}
	return set
}

// ToMap 将切片转为 map：f 对每个元素返回键值对，key 重复时后出现的覆盖先出现的。
func ToMap[S ~[]E, E any, K comparable, V any](s S, f func(E) (K, V)) map[K]V {
	m := make(map[K]V, len(s))
	for _, item := range s {
		k, v := f(item)
		m[k] = v
	}
	return m
}

// KeyBy 以 f 计算的 key 索引元素本身，返回 map[K]E；key 重复时后出现的元素覆盖先出现的。
func KeyBy[S ~[]E, E any, K comparable](s S, f func(E) K) map[K]E {
	m := make(map[K]E, len(s))
	for _, item := range s {
		m[f(item)] = item
	}
	return m
}

// GroupBy 按 f 计算的 key 分组，返回 map[K][]E；同组元素保持出现顺序。
func GroupBy[S ~[]E, E any, K comparable](s S, f func(E) K) map[K][]E {
	m := make(map[K][]E, len(s))
	for _, item := range s {
		k := f(item)
		m[k] = append(m[k], item)
	}
	return m
}
