package xslices

import (
	"cmp"
	"slices"
)

// LastIndex 类似 slices.Index, 不过是反向查找的
func LastIndex[S ~[]E, E comparable](s S, v E) int {
	for i := len(s) - 1; i >= 0; i-- {
		if v == s[i] {
			return i
		}
	}
	return -1
}

// LastIndexFunc 类似 slices.IndexFunc, 不过是反向查找的
func LastIndexFunc[S ~[]E, E any](s S, f func(E) bool) int {
	for i := len(s) - 1; i >= 0; i-- {
		if f(s[i]) {
			return i
		}
	}
	return -1
}

// IndexBy 类似 slices.Index, 不过以 keyOf 计算的 key 判等, 元素本身不可比较时也可用。
func IndexBy[S ~[]E, E any, K comparable](s S, v E, keyOf func(E) K) int {
	key := keyOf(v)
	return slices.IndexFunc(s, func(item E) bool {
		return keyOf(item) == key
	})
}

// LastIndexBy 类似 IndexBy, 不过是反向查找的。
func LastIndexBy[S ~[]E, E any, K comparable](s S, v E, keyOf func(E) K) int {
	key := keyOf(v)
	return LastIndexFunc(s, func(item E) bool {
		return keyOf(item) == key
	})
}

// Map 逐元素映射为新切片。
func Map[S ~[]E1, E1 any, E2 any](s S, mapper func(E1) E2) []E2 {
	if len(s) == 0 {
		return nil
	}

	result := make([]E2, len(s))
	for i, item := range s {
		result[i] = mapper(item)
	}
	return result
}

// Filter 按谓词过滤
func Filter[S ~[]E, E any](s S, filter func(E) bool) []E {
	if len(s) == 0 {
		return nil
	}

	result := make([]E, 0, len(s))
	for _, item := range s {
		if filter(item) {
			result = append(result, item)
		}
	}
	return result
}

// FilterMap 过滤并映射：仅 f 返回 ok 的元素计入结果
func FilterMap[S ~[]E1, E1 any, E2 any](s S, f func(E1) (E2, bool)) []E2 {
	if len(s) == 0 {
		return nil
	}

	result := make([]E2, 0, len(s))
	for _, item := range s {
		if r, ok := f(item); ok {
			result = append(result, r)
		}
	}
	return result
}

// FlatMap 映射并展平：将 f 返回的 slice 依次拼接为结果
func FlatMap[S ~[]E1, E1 any, E2 any](s S, f func(E1) []E2) []E2 {
	if len(s) == 0 {
		return nil
	}

	result := make([][]E2, 0, len(s))
	for _, item := range s {
		result = append(result, f(item))
	}
	return slices.Concat(result...)
}

// Unique 去重，保留元素首次出现的顺序，返回新 slice
func Unique[S ~[]E, E comparable](s S) []E {
	if len(s) == 0 {
		return nil
	}

	seen := make(map[E]struct{}, len(s))
	result := make([]E, 0, len(s))
	for _, item := range s {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// UniqueBy 以 f 计算的 key 判等去重，保留元素首次出现的顺序，返回新 slice
func UniqueBy[S ~[]E, E any, K comparable](s S, f func(E) K) []E {
	if len(s) == 0 {
		return nil
	}

	seen := make(map[K]struct{}, len(s))
	result := make([]E, 0, len(s))
	for _, item := range s {
		key := f(item)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// SortedUnique 排序去重，返回排序且去重的 slice
func SortedUnique[S ~[]E, E cmp.Ordered](s S) []E {
	if len(s) == 0 {
		return nil
	}

	res := slices.Clone(s)
	slices.Sort(res)
	res = slices.Compact(res)
	return res
}

// Get 取下标 idx 处的元素，越界时返回零值而不 panic。
func Get[S ~[]E, E any](s S, idx int) E {
	if 0 <= idx && idx < len(s) {
		return s[idx]
	}

	var zero E
	return zero
}

// GetOr 取下标 idx 处的元素，越界时返回 orValue。
func GetOr[S ~[]E, E any](s S, idx int, orValue E) E {
	if 0 <= idx && idx < len(s) {
		return s[idx]
	}

	return orValue
}

// GetOrElse 同 GetOr，但兜底值惰性求值：越界时才调用 elseValue。
func GetOrElse[S ~[]E, E any](s S, idx int, elseValue func() E) E {
	if 0 <= idx && idx < len(s) {
		return s[idx]
	}

	return elseValue()
}
