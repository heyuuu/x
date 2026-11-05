package jsonl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "log.jsonl") // 父目录不存在，验证自动创建

	require.NoError(t, Append(path, map[string]int{"x": 1}, map[string]int{"x": 2}))
	require.NoError(t, Append(path, map[string]int{"x": 3}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, `{"x":1}
{"x":2}
{"x":3}
`, string(data), "应逐行紧凑写入，无多余空行")
}

func TestAppendBadLineRepair(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.jsonl")

	// 模拟上次写中断留下的残缺行：末尾无换行
	require.NoError(t, os.WriteFile(path, []byte(`{"x":1}`), 0644))

	require.NoError(t, Append(path, map[string]int{"x": 2}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, `{"x":1}
{"x":2}
`, string(data), "残缺行应被隔断，新记录独占一行")
}
