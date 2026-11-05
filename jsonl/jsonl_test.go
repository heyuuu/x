package jsonl

import (
	"math"
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

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "log.jsonl") // 父目录不存在，验证自动创建

	require.NoError(t, Write(path, map[string]int{"x": 1}, map[string]int{"x": 2}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, `{"x":1}
{"x":2}
`, string(data), "应逐行紧凑写入，无多余空行")
}

// Write 是整文件覆盖写，旧内容不残留。
func TestWriteOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"x":1}
{"x":2}
{"x":3}
`), 0644))

	require.NoError(t, Write(path, map[string]int{"x": 9}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, `{"x":9}
`, string(data), "旧内容应被整体替换，不残留")
}

// 先全部序列化成功再落盘：序列化失败时旧文件应原样保留。
func TestWriteMarshalErrorKeepsOldFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.jsonl")
	const old = `{"x":1}` + "\n"
	require.NoError(t, os.WriteFile(path, []byte(old), 0644))

	assert.Error(t, Write(path, map[string]int{"x": 2}, math.NaN()))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, old, string(data), "失败时旧文件不应被触碰")
}

// values 为空时写入空文件。
func TestWriteEmptyValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"x":1}`+"\n"), 0644))

	require.NoError(t, Write(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Empty(t, data, "无值时应写入空文件")
}

// 原子写成功后目录中不留 tmp 残留。
func TestWriteNoTempLeftover(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.jsonl")

	require.NoError(t, Write(path, map[string]int{"x": 1}))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "目录中应只有目标文件")
	assert.Equal(t, "log.jsonl", entries[0].Name())
}

// Write 与 Each 互逆往返。
func TestWriteEachRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.jsonl")
	require.NoError(t, Write(path,
		map[string]int{"x": 1}, map[string]int{"x": 2}, map[string]int{"x": 3}))

	var got []map[string]int
	for value, err := range Each[map[string]int](path) {
		require.NoError(t, err)
		got = append(got, value)
	}
	assert.Equal(t, []map[string]int{{"x": 1}, {"x": 2}, {"x": 3}}, got)
}
