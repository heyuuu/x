package xslices

import (
	"maps"
	"slices"
	"testing"
)

func TestToSet(t *testing.T) {
	byLen := func(s string) int { return len(s) }
	tests := []struct {
		name string
		s    []string
		f    func(string) int
		want []int // 期望的集合内容（以排序后的 key 表达）
	}{
		{"重复 key 自然去重", []string{"a", "bb", "cc", "d"}, byLen, []int{1, 2}},
		{"key 各不相同", []string{"a", "bb", "ccc"}, byLen, []int{1, 2, 3}},
		{"全部同 key", []string{"a", "b", "c"}, byLen, []int{1}},
		{"空切片", []string{}, byLen, nil},
		{"nil 切片", nil, byLen, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := ToSet(tt.s, tt.f)
			if got := slices.Sorted(maps.Keys(set)); !slices.Equal(got, tt.want) {
				t.Errorf("ToSet(%v) 的 key = %v, want %v", tt.s, got, tt.want)
			}
		})
	}

	if got := ToSet(ints{3, 1, 3}, func(i int) int { return i }); !slices.Equal(slices.Sorted(maps.Keys(got)), []int{1, 3}) {
		t.Errorf("ToSet() 自定义切片类型 = %v, want [1 3]", got)
	}
}

func TestToBoolSet(t *testing.T) {
	byLen := func(s string) int { return len(s) }
	t.Run("存在的 key 为 true", func(t *testing.T) {
		set := ToBoolSet([]string{"a", "bb", "cc"}, byLen)
		if !set[1] || !set[2] || len(set) != 2 {
			t.Errorf("ToBoolSet() = %v, want {1:true 2:true}", set)
		}
	})
	t.Run("不存在的 key 为 false", func(t *testing.T) {
		set := ToBoolSet([]string{"a", "bb"}, byLen)
		if set[3] {
			t.Errorf("ToBoolSet() 中不存在的 key 应为 false, got %v", set)
		}
	})
	t.Run("重复 key 去重", func(t *testing.T) {
		set := ToBoolSet([]string{"a", "b", "c"}, byLen)
		if len(set) != 1 || !set[1] {
			t.Errorf("ToBoolSet() = %v, want {1:true}", set)
		}
	})
	t.Run("空与 nil 切片", func(t *testing.T) {
		if got := ToBoolSet([]string{}, byLen); len(got) != 0 {
			t.Errorf("ToBoolSet() 空切片 = %v, want 空 map", got)
		}
		var nilSlice []string
		if got := ToBoolSet(nilSlice, byLen); len(got) != 0 {
			t.Errorf("ToBoolSet() nil 切片 = %v, want 空 map", got)
		}
	})

	if got := ToBoolSet(ints{2, 4}, func(i int) bool { return i%2 == 0 }); !got[true] {
		t.Errorf("ToBoolSet() 自定义切片类型 = %v, want {true:true}", got)
	}
}

func TestToMap(t *testing.T) {
	lenOf := func(s string) (int, string) { return len(s), s }
	tests := []struct {
		name string
		s    []string
		f    func(string) (int, string)
		want map[int]string
	}{
		{"切片转 map", []string{"a", "bb", "ccc"}, lenOf, map[int]string{1: "a", 2: "bb", 3: "ccc"}},
		// 关键语义：key 重复时后出现的覆盖先出现的
		{"key 重复后者覆盖", []string{"a", "x", "y"}, lenOf, map[int]string{1: "y"}},
		{"空切片", []string{}, lenOf, map[int]string{}},
		{"nil 切片", nil, lenOf, map[int]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToMap(tt.s, tt.f); !maps.Equal(got, tt.want) {
				t.Errorf("ToMap(%v) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}

	if got := ToMap(ints{1, 2}, func(i int) (int, int) { return i, i * 10 }); !maps.Equal(got, map[int]int{1: 10, 2: 20}) {
		t.Errorf("ToMap() 自定义切片类型 = %v, want map[1:10 2:20]", got)
	}
}

func TestKeyBy(t *testing.T) {
	byLen := func(s string) int { return len(s) }
	tests := []struct {
		name string
		s    []string
		f    func(string) int
		want map[int]string
	}{
		{"以 key 索引元素本身", []string{"a", "bb", "ccc"}, byLen, map[int]string{1: "a", 2: "bb", 3: "ccc"}},
		// 关键语义：key 重复时后出现的元素覆盖先出现的
		{"key 重复后者覆盖", []string{"a", "b", "c"}, byLen, map[int]string{1: "c"}},
		{"空切片", []string{}, byLen, map[int]string{}},
		{"nil 切片", nil, byLen, map[int]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KeyBy(tt.s, tt.f); !maps.Equal(got, tt.want) {
				t.Errorf("KeyBy(%v) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}

	if got := KeyBy(ints{3, 1}, func(i int) int { return i }); !maps.Equal(got, map[int]int{3: 3, 1: 1}) {
		t.Errorf("KeyBy() 自定义切片类型 = %v, want map[3:3 1:1]", got)
	}
}

func TestGroupBy(t *testing.T) {
	byLen := func(s string) int { return len(s) }
	tests := []struct {
		name string
		s    []string
		f    func(string) int
		want map[int][]string
	}{
		// 关键语义：同组元素保持出现顺序
		{"按 key 分组且组内保序", []string{"a", "bb", "c", "dd", "ccc"}, byLen, map[int][]string{1: {"a", "c"}, 2: {"bb", "dd"}, 3: {"ccc"}}},
		{"每组各一个元素", []string{"a", "bb", "ccc"}, byLen, map[int][]string{1: {"a"}, 2: {"bb"}, 3: {"ccc"}}},
		{"全部同组", []string{"a", "b", "c"}, byLen, map[int][]string{1: {"a", "b", "c"}}},
		{"空切片", []string{}, byLen, map[int][]string{}},
		{"nil 切片", nil, byLen, map[int][]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupBy(tt.s, tt.f)
			if !maps.EqualFunc(got, tt.want, slices.Equal) {
				t.Errorf("GroupBy(%v) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}

	if got := GroupBy(ints{1, 2, 3, 4}, func(i int) bool { return i%2 == 0 }); !maps.EqualFunc(got, map[bool][]int{true: {2, 4}, false: {1, 3}}, slices.Equal) {
		t.Errorf("GroupBy() 自定义切片类型 = %v, want map[true:[2 4] false:[1 3]]", got)
	}
}
