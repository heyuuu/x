package xslices

import (
	"fmt"
	"slices"
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
