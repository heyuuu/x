package store

import (
	"fmt"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONStore(t *testing.T) {
	// 父目录不存在，验证 Save 自动创建
	path := filepath.Join(t.TempDir(), "config", "state.json")
	s := JSONStore[map[string]int](path)

	_, err := s.Load()
	assert.ErrorIs(t, err, ErrStoreMissing, "缺失文件应返回 ErrStoreMissing")

	want := map[string]int{"k": 1}
	require.NoError(t, s.Save(want), "Save 失败")
	got, err := s.Load()
	require.NoError(t, err, "Load 失败")
	assert.Equal(t, want, got, "写读不一致")
}

func TestSectionStore(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.json")
	s := SectionStore[[]string](file, "items")

	_, err := s.Load()
	assert.ErrorIs(t, err, ErrStoreMissing, "缺失文件/节缺失应返回 ErrStoreMissing")

	require.NoError(t, s.Save([]string{"a"}), "Save 失败")
	got, err := s.Load()
	require.NoError(t, err, "Load 失败")
	assert.Equal(t, []string{"a"}, got, "写读不一致")
}

func TestWrapperStore(t *testing.T) {
	inner := JSONStore[int](filepath.Join(t.TempDir(), "state.json"))

	// wrap：int 转出 string，负数视为非法，用于验证 wrap 失败路径
	wrap := func(n int) (string, error) {
		if n < 0 {
			return "", fmt.Errorf("非法值: %d", n)
		}
		return strconv.Itoa(n), nil
	}
	unwrap := func(s string) (int, error) { return strconv.Atoi(s) }
	s := WrapperStore(inner, wrap, unwrap)

	// 内部缺失时透传哨兵错误，LoadOrZero 返回零值
	_, err := s.Load()
	assert.ErrorIs(t, err, ErrStoreMissing, "内部缺失应透传 ErrStoreMissing")
	assert.Empty(t, s.LoadOrZero(), "内部缺失时 LoadOrZero 应返回零值")

	// Save 经 unwrap 落盘，Load 经 wrap 还原
	require.NoError(t, s.Save("42"), "Save 失败")
	got, err := s.Load()
	require.NoError(t, err, "Load 失败")
	assert.Equal(t, "42", got, "写读不一致")

	// unwrap 失败时直接上抛，不落盘
	err = s.Save("not-a-number")
	assert.Error(t, err, "unwrap 失败应上抛")
	got, err = s.Load()
	require.NoError(t, err, "Load 失败")
	assert.Equal(t, "42", got, "unwrap 失败不应改写内部存储")

	// wrap 失败时 Load 上抛，LoadOrZero 返回零值
	require.NoError(t, inner.Save(-1), "写内部非法值失败")
	_, err = s.Load()
	assert.Error(t, err, "wrap 失败应上抛")
	assert.Empty(t, s.LoadOrZero(), "wrap 失败时 LoadOrZero 应返回零值")
}

func TestWrapperStoreNilPanic(t *testing.T) {
	wrap := func(n int) (string, error) { return strconv.Itoa(n), nil }
	unwrap := func(s string) (int, error) { return strconv.Atoi(s) }

	assert.PanicsWithValue(t, "store: wrap cannot be nil", func() {
		WrapperStore[int, string](nil, nil, unwrap)
	}, "wrap 为 nil 应 panic")
	assert.PanicsWithValue(t, "store: unwrap cannot be nil", func() {
		WrapperStore[int, string](nil, wrap, nil)
	}, "unwrap 为 nil 应 panic")
}
