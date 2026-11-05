package store

import (
	"path/filepath"
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
