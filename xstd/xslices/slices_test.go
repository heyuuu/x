package xslices

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// ints 用于验证 ~[] 约束对自定义切片类型同样生效。
type ints []int

func TestLastIndex(t *testing.T) {
	tests := []struct {
		name string
		s    []string
		v    string
		want int
	}{
		// 与 slices.Index 的区别所在：重复元素取最后一次出现
		{"重复元素取最后出现", []string{"a", "b", "a", "c"}, "a", 2},
		{"命中末尾元素", []string{"a", "b", "c"}, "c", 2},
		{"未命中", []string{"a", "b"}, "x", -1},
		{"空切片", []string{}, "a", -1},
		{"nil 切片", nil, "a", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LastIndex(tt.s, tt.v); got != tt.want {
				t.Errorf("LastIndex(%v, %q) = %d, want %d", tt.s, tt.v, got, tt.want)
			}
		})
	}

	if got := LastIndex(ints{3, 1, 3}, 3); got != 2 {
		t.Errorf("LastIndex() 自定义切片类型 = %d, want 2", got)
	}
}

func TestLastIndexFunc(t *testing.T) {
	even := func(i int) bool { return i%2 == 0 }
	tests := []struct {
		name string
		s    []int
		f    func(int) bool
		want int
	}{
		{"返回最后一个满足者的下标", []int{2, 1, 4, 5, 6}, even, 4},
		{"全部不满足", []int{1, 3, 5}, even, -1},
		{"空切片", []int{}, even, -1},
		{"nil 切片", nil, even, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LastIndexFunc(tt.s, tt.f); got != tt.want {
				t.Errorf("LastIndexFunc() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMap(t *testing.T) {
	if got := Map([]int{1, 2, 3}, func(i int) int { return i * 2 }); !slices.Equal(got, []int{2, 4, 6}) {
		t.Errorf("Map() = %v, want [2 4 6]", got)
	}
	// 映射可以改变元素类型
	if got := Map([]int{1, 2}, func(i int) string { return fmt.Sprintf("#%d", i) }); !slices.Equal(got, []string{"#1", "#2"}) {
		t.Errorf("Map() 类型转换 = %v, want [#1 #2]", got)
	}
	// 空输入返回 nil
	if got := Map([]int{}, func(i int) int { return i }); got != nil {
		t.Errorf("Map() 空切片 = %v, want nil", got)
	}
	var nilSlice []int
	if got := Map(nilSlice, func(i int) int { return i }); got != nil {
		t.Errorf("Map() nil 切片 = %v, want nil", got)
	}
	if got := Map(ints{1, 2}, func(i int) int { return i + 1 }); !slices.Equal(got, []int{2, 3}) {
		t.Errorf("Map() 自定义切片类型 = %v, want [2 3]", got)
	}
}

func TestFilter(t *testing.T) {
	even := func(i int) bool { return i%2 == 0 }
	tests := []struct {
		name string
		s    []int
		f    func(int) bool
		want []int
	}{
		{"保留满足者且保持顺序", []int{1, 2, 3, 4}, even, []int{2, 4}},
		{"全部保留", []int{2, 4}, even, []int{2, 4}},
		{"全部滤掉", []int{1, 3}, even, nil},
		{"空切片", []int{}, even, nil},
		{"nil 切片", nil, even, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Filter(tt.s, tt.f)
			if !slices.Equal(got, tt.want) {
				t.Errorf("Filter() = %v, want %v", got, tt.want)
			}
		})
	}
	// 空输入返回 nil
	if got := Filter([]int{}, even); got != nil {
		t.Errorf("Filter() 空切片 = %v, want nil", got)
	}

	if got := Filter(ints{1, 2, 3, 4}, even); !slices.Equal(got, []int{2, 4}) {
		t.Errorf("Filter() 自定义切片类型 = %v, want [2 4]", got)
	}
}

func TestFilterMap(t *testing.T) {
	// 偶数保留并转为字符串，兼顾过滤与类型变换
	evenStr := func(i int) (string, bool) { return fmt.Sprintf("#%d", i), i%2 == 0 }
	tests := []struct {
		name string
		s    []int
		f    func(int) (string, bool)
		want []string
	}{
		{"过滤并映射", []int{1, 2, 3, 4}, evenStr, []string{"#2", "#4"}},
		{"全部保留", []int{2, 4}, evenStr, []string{"#2", "#4"}},
		{"全部滤掉", []int{1, 3}, evenStr, nil},
		{"空切片", []int{}, evenStr, nil},
		{"nil 切片", nil, evenStr, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FilterMap(tt.s, tt.f); !slices.Equal(got, tt.want) {
				t.Errorf("FilterMap(%v) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}

	if got := FilterMap(ints{1, 2}, func(i int) (int, bool) { return i * 10, i > 1 }); !slices.Equal(got, []int{20}) {
		t.Errorf("FilterMap() 自定义切片类型 = %v, want [20]", got)
	}
}

func TestFlatMap(t *testing.T) {
	tests := []struct {
		name string
		s    []string
		f    func(string) []string
		want []string
	}{
		{"映射并展平", []string{"a b", "c"}, strings.Fields, []string{"a", "b", "c"}},
		// strings.Fields("") 返回 nil，内层空 slice 应被跳过
		{"内层空 slice 跳过", []string{"a", "", "b"}, strings.Fields, []string{"a", "b"}},
		{"全部内层为空", []string{"", ""}, strings.Fields, nil},
		{"空切片", []string{}, strings.Fields, nil},
		{"nil 切片", nil, strings.Fields, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FlatMap(tt.s, tt.f); !slices.Equal(got, tt.want) {
				t.Errorf("FlatMap(%v) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
	// 展平二维切片，内层可为空
	flatten := func(is []int) []int { return is }
	if got := FlatMap([][]int{{1, 2}, {}, {3}}, flatten); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("FlatMap() 展平二维切片 = %v, want [1 2 3]", got)
	}

	if got := FlatMap(ints{1, 2}, func(i int) []int { return []int{i, -i} }); !slices.Equal(got, []int{1, -1, 2, -2}) {
		t.Errorf("FlatMap() 自定义切片类型 = %v, want [1 -1 2 -2]", got)
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name string
		s    []int
		want []int
	}{
		// 与排序去重的区别所在：保留首次出现的顺序
		{"重复元素保留首次出现", []int{3, 1, 3, 2, 1}, []int{3, 1, 2}},
		{"无重复", []int{1, 2, 3}, []int{1, 2, 3}},
		{"全部重复", []int{7, 7, 7}, []int{7}},
		{"空切片", []int{}, nil},
		{"nil 切片", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Unique(tt.s); !slices.Equal(got, tt.want) {
				t.Errorf("Unique(%v) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
	// 空输入返回 nil
	if got := Unique([]int{}); got != nil {
		t.Errorf("Unique() 空切片 = %v, want nil", got)
	}

	if got := Unique(ints{3, 1, 3}); !slices.Equal(got, []int{3, 1}) {
		t.Errorf("Unique() 自定义切片类型 = %v, want [3 1]", got)
	}
}

func TestUniqueBy(t *testing.T) {
	byLen := func(s string) int { return len(s) }
	tests := []struct {
		name string
		s    []string
		f    func(string) int
		want []string
	}{
		{"key 相同保留首次出现", []string{"a", "b", "cc", "d", "ee"}, byLen, []string{"a", "cc"}},
		{"key 各不相同", []string{"a", "bb", "ccc"}, byLen, []string{"a", "bb", "ccc"}},
		{"全部同 key", []string{"a", "b", "c"}, byLen, []string{"a"}},
		{"空切片", []string{}, byLen, nil},
		{"nil 切片", nil, byLen, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UniqueBy(tt.s, tt.f); !slices.Equal(got, tt.want) {
				t.Errorf("UniqueBy(%v) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
	// 元素本身不可比较时也能按 key 去重
	byFirst := func(s []int) int { return s[0] }
	got := UniqueBy([][]int{{1, 2}, {3}, {1, 5}, {4}}, byFirst)
	if len(got) != 3 || !slices.Equal(got[0], []int{1, 2}) || !slices.Equal(got[1], []int{3}) || !slices.Equal(got[2], []int{4}) {
		t.Errorf("UniqueBy() 不可比较元素 = %v, want [[1 2] [3] [4]]", got)
	}

	if got := UniqueBy(ints{3, 1, 2, 4}, func(i int) bool { return i%2 == 0 }); !slices.Equal(got, []int{3, 2}) {
		t.Errorf("UniqueBy() 自定义切片类型 = %v, want [3 2]", got)
	}
}

func TestSortedUnique(t *testing.T) {
	tests := []struct {
		name string
		s    []int
		want []int
	}{
		{"排序并去重", []int{3, 1, 3, 2, 1}, []int{1, 2, 3}},
		{"已排序无重复", []int{1, 2, 3}, []int{1, 2, 3}},
		{"全部重复", []int{7, 7, 7}, []int{7}},
		{"空切片", []int{}, nil},
		{"nil 切片", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SortedUnique(tt.s); !slices.Equal(got, tt.want) {
				t.Errorf("SortedUnique(%v) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
	// 不修改原 slice
	origin := []int{3, 1, 2}
	SortedUnique(origin)
	if !slices.Equal(origin, []int{3, 1, 2}) {
		t.Errorf("SortedUnique() 修改了原 slice: %v", origin)
	}

	if got := SortedUnique(ints{2, 1, 2}); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("SortedUnique() 自定义切片类型 = %v, want [1 2]", got)
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name string
		s    []int
		idx  int
		want int
	}{
		{"下标 0", []int{10, 20, 30}, 0, 10},
		{"中间下标", []int{10, 20, 30}, 1, 20},
		{"末尾下标", []int{10, 20, 30}, 2, 30},
		{"负下标", []int{10, 20, 30}, -1, 0},
		{"下标等于 len", []int{10, 20, 30}, 3, 0},
		{"下标大幅越界", []int{10, 20, 30}, 100, 0},
		{"空切片", []int{}, 0, 0},
		{"nil 切片", nil, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Get(tt.s, tt.idx); got != tt.want {
				t.Errorf("Get(%v, %d) = %v, want %v", tt.s, tt.idx, got, tt.want)
			}
		})
	}

	if got := Get(ints{7}, 0); got != 7 {
		t.Errorf("Get() 自定义切片类型 = %v, want 7", got)
	}
}

func TestGetOr(t *testing.T) {
	tests := []struct {
		name    string
		s       []int
		idx     int
		orValue int
		want    int
	}{
		{"命中", []int{10, 20, 30}, 1, 99, 20},
		// 关键区别于 Get 的零值兜底：命中的是零值也返回存量值，而非 orValue
		{"命中且值为零值", []int{0, 20}, 0, 99, 0},
		{"负下标取兜底值", []int{10, 20, 30}, -1, 99, 99},
		{"下标等于 len 取兜底值", []int{10, 20, 30}, 3, 99, 99},
		{"空切片取兜底值", []int{}, 0, 99, 99},
		{"nil 切片取兜底值", nil, 0, 99, 99},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOr(tt.s, tt.idx, tt.orValue); got != tt.want {
				t.Errorf("GetOr(%v, %d) = %v, want %v", tt.s, tt.idx, got, tt.want)
			}
		})
	}
}

func TestGetOrElse(t *testing.T) {
	t.Run("命中时不调用 fn", func(t *testing.T) {
		called := false
		fn := func() int { called = true; return 99 }
		if got := GetOrElse([]int{10, 20}, 1, fn); got != 20 {
			t.Errorf("GetOrElse() = %v, want 20", got)
		}
		if called {
			t.Error("下标命中时不应调用 elseValue")
		}
	})
	t.Run("越界时调用 fn", func(t *testing.T) {
		if got := GetOrElse([]int{10, 20}, 5, func() int { return 99 }); got != 99 {
			t.Errorf("GetOrElse() = %v, want 99", got)
		}
	})
	t.Run("nil 切片时调用 fn", func(t *testing.T) {
		var s []int
		if got := GetOrElse(s, 0, func() int { return 99 }); got != 99 {
			t.Errorf("GetOrElse() = %v, want 99", got)
		}
	})
}
