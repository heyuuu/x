package xmaps

import (
	"fmt"
	"maps"
	"slices"
	"testing"
)

// 基准规模覆盖三个数量级以上。
var benchSizes = []int{10, 100, 1000, 10000}

func benchMap(size int) map[int]int {
	m := make(map[int]int, size)
	for i := range size {
		m[i] = i
	}
	return m
}

// benchSink 承接结果：go vet 要求标准库调用结果必须被使用，同时排除死代码消除的干扰。
var benchSink any

// BenchmarkKeysVsCollect 对比 Keys(m) 与 slices.Collect(maps.Keys(m))：
// 两者都返回物化切片，前者按 len(m) 预分配一次到位，后者从 nil 起 append 反复扩容。
func BenchmarkKeysVsCollect(b *testing.B) {
	for _, size := range benchSizes {
		m := benchMap(size)
		b.Run(fmt.Sprintf("n=%d/xmaps.Keys", size), func(b *testing.B) {
			for b.Loop() {
				benchSink = Keys(m)
			}
		})
		b.Run(fmt.Sprintf("n=%d/slices.Collect(maps.Keys)", size), func(b *testing.B) {
			for b.Loop() {
				benchSink = slices.Collect(maps.Keys(m))
			}
		})
	}
}

// BenchmarkValuesVsCollect 同上，对比 Values(m) 与 slices.Collect(maps.Values(m))。
func BenchmarkValuesVsCollect(b *testing.B) {
	for _, size := range benchSizes {
		m := benchMap(size)
		b.Run(fmt.Sprintf("n=%d/xmaps.Values", size), func(b *testing.B) {
			for b.Loop() {
				benchSink = Values(m)
			}
		})
		b.Run(fmt.Sprintf("n=%d/slices.Collect(maps.Values)", size), func(b *testing.B) {
			for b.Loop() {
				benchSink = slices.Collect(maps.Values(m))
			}
		})
	}
}
