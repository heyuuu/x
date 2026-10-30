package store

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "data.json")

	require.NoError(t, WriteFileAtomic(path, []byte("hello"), 0644), "首次原子写失败")
	// 深层目录不存在时自动递归创建
	deep := filepath.Join(dir, "a", "b", "c", "d.json")
	require.NoError(t, WriteFileAtomic(deep, []byte("x"), 0644), "深层目录自动创建失败")
	data, err := os.ReadFile(deep)
	require.NoError(t, err)
	assert.Equal(t, "x", string(data), "深层目录写入内容不对")

	require.NoError(t, WriteFileAtomic(path, []byte("world"), 0600), "覆盖原子写失败")
	data, err = os.ReadFile(path)
	require.NoError(t, err, "读回失败")
	assert.Equal(t, "world", string(data), "覆盖后内容不对")
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0600), info.Mode().Perm(), "权限未生效")

	// 目录里不应残留 tmp 文件
	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	var residue []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" || strings.HasPrefix(e.Name(), ".") {
			residue = append(residue, e.Name())
		}
	}
	assert.Empty(t, residue, "目录里不应残留 tmp 文件")
}

// 统一契约：所有写操作自动递归创建父目录
func TestWriteOpsCreateParentDir(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, AppendJsonl(filepath.Join(dir, "a/b/c/usage.jsonl"), map[string]int{"x": 1}), "AppendJsonl 应自动建父目录")
	require.NoError(t, SaveJSON(filepath.Join(dir, "d/e/config.json"), map[string]int{"x": 1}), "SaveJSON 应自动建父目录")
	require.NoError(t, WriteFileAtomic(filepath.Join(dir, "f/g/raw.bin"), []byte("x"), 0644), "WriteFileAtomic 应自动建父目录")
}
