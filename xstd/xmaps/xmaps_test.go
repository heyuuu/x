package xmaps

import (
	"slices"
	"testing"
)

// counts 用于验证 ~map 约束对自定义 map 类型同样生效。
type counts map[string]int

func TestKeys(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]int
		want []string
	}{
		{"多个键", map[string]int{"b": 2, "a": 1, "c": 3}, []string{"a", "b", "c"}},
		{"空 map", map[string]int{}, nil},
		{"nil map", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Keys(tt.m)
			slices.Sort(got) // Keys 不保证有序，排序后比对
			if !slices.Equal(got, tt.want) {
				t.Errorf("Keys() = %v, want %v", got, tt.want)
			}
		})
	}

	if got := Keys(counts{"one": 1, "two": 2}); len(got) != 2 {
		t.Errorf("Keys() 自定义 map 类型 = %v, want 2 个键", got)
	}
}

func TestValues(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]int
		want []int
	}{
		{"多个值", map[string]int{"a": 2, "b": 1, "c": 3}, []int{1, 2, 3}},
		{"值可重复", map[string]int{"a": 1, "b": 1, "c": 2}, []int{1, 1, 2}},
		{"空 map", map[string]int{}, nil},
		{"nil map", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Values(tt.m)
			slices.Sort(got) // Values 不保证有序，排序后比对
			if !slices.Equal(got, tt.want) {
				t.Errorf("Values() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortedKeys(t *testing.T) {
	if got := SortedKeys(map[int]string{3: "c", 1: "a", 2: "b"}); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("SortedKeys() = %v, want [1 2 3]", got)
	}
	if got := SortedKeys(map[string]int{}); len(got) != 0 {
		t.Errorf("SortedKeys() 空 map = %v, want 空切片", got)
	}
	var nilMap map[int]string
	if got := SortedKeys(nilMap); len(got) != 0 {
		t.Errorf("SortedKeys() nil map = %v, want 空切片", got)
	}
}

func TestGetOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		m            map[string]int
		key          string
		defaultValue int
		want         int
	}{
		{"命中", map[string]int{"a": 1}, "a", 99, 1},
		{"未命中", map[string]int{"a": 1}, "b", 99, 99},
		// 关键区别于直接下标 m[key]：存的是零值也返回存量值，而非默认值
		{"命中且值为零值", map[string]int{"a": 0}, "a", 99, 0},
		{"nil map", nil, "a", 99, 99},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOrDefault(tt.m, tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("GetOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOrElse(t *testing.T) {
	t.Run("命中时不调用 fn", func(t *testing.T) {
		called := false
		fn := func() int { called = true; return 99 }
		if got := GetOrElse(map[string]int{"a": 1}, "a", fn); got != 1 {
			t.Errorf("GetOrElse() = %v, want 1", got)
		}
		if called {
			t.Error("key 命中时不应调用 defaultValue")
		}
	})
	t.Run("未命中时调用 fn", func(t *testing.T) {
		if got := GetOrElse(map[string]int{"a": 1}, "b", func() int { return 99 }); got != 99 {
			t.Errorf("GetOrElse() = %v, want 99", got)
		}
	})
	t.Run("nil map 时调用 fn", func(t *testing.T) {
		var m map[string]int
		if got := GetOrElse(m, "a", func() int { return 99 }); got != 99 {
			t.Errorf("GetOrElse() = %v, want 99", got)
		}
	})
}

func TestGetOrPut(t *testing.T) {
	t.Run("命中时不调用 fn 也不写入", func(t *testing.T) {
		called := false
		fn := func() int { called = true; return 99 }
		m := map[string]int{"a": 1}
		if got := GetOrPut(m, "a", fn); got != 1 {
			t.Errorf("GetOrPut() = %v, want 1", got)
		}
		if called {
			t.Error("key 命中时不应调用 defaultValue")
		}
		if len(m) != 1 {
			t.Errorf("map = %v, want 保持 1 个键", m)
		}
	})
	t.Run("未命中时调用 fn 并写入", func(t *testing.T) {
		m := map[string]int{}
		if got := GetOrPut(m, "x", func() int { return 42 }); got != 42 {
			t.Errorf("GetOrPut() = %v, want 42", got)
		}
		if m["x"] != 42 {
			t.Errorf("m[%q] = %v, want 42", "x", m["x"])
		}
		// 二次调用命中已写入的值
		if got := GetOrPut(m, "x", func() int { return 0 }); got != 42 {
			t.Errorf("GetOrPut() 二次调用 = %v, want 42", got)
		}
	})
	t.Run("nil map 写入 panic", func(t *testing.T) {
		var m map[string]int
		defer func() {
			if recover() == nil {
				t.Error("m 为 nil 时写入应 panic（与原生 map 写语义一致）")
			}
		}()
		GetOrPut(m, "x", func() int { return 7 })
	})
}
