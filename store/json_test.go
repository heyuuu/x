package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoadJson(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")

	type conf struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}

	// 缺失文件返回哨兵错误
	_, err := LoadJSON[conf](path)
	require.ErrorIs(t, err, ErrFileMissing, "缺失文件应返回 ErrFileMissing")

	want := conf{Name: "xdiff", N: 3}
	require.NoError(t, SaveJSON(path, want), "SaveJSON 失败")
	got, err := LoadJSON[conf](path)
	require.NoError(t, err, "LoadJSON 失败")
	assert.Equal(t, want, got, "roundtrip 不一致")

	// 坏 JSON 返回解析错误（非哨兵）
	require.NoError(t, os.WriteFile(path, []byte("{oops"), 0644))
	_, err = LoadJSON[conf](path)
	require.Error(t, err, "坏 JSON 应返回解析错误")
	assert.NotErrorIs(t, err, ErrFileMissing, "坏 JSON 不应返回哨兵错误")
}

// 稳定性契约：同样的数据（无论 map 写入顺序、nil slice/map）每次写出字节完全一致。
func TestSaveJSONStable(t *testing.T) {
	dir := t.TempDir()

	type inner struct {
		Tags   []string          `json:"tags"`
		Meta   map[string]string `json:"meta"`
		PtrN   *int              `json:"ptr_n"`
		BigNum int64             `json:"big_num"`
	}
	type doc struct {
		Items []int            `json:"items"`
		KV    map[string]int   `json:"kv"`
		Subs  []inner          `json:"subs"`
		By    map[string]inner `json:"by"`
	}

	n := 3
	build := func() doc {
		return doc{
			Items: []int{1, 2, 3},
			KV:    map[string]int{"zebra": 1, "apple": 2, "mango": 3},
			Subs: []inner{
				{Tags: []string{"a"}, Meta: map[string]string{"k": "v"}, PtrN: &n, BigNum: 9007199254740993},
			},
			By: map[string]inner{"z": {Tags: []string{"z"}}},
		}
	}

	// 同一份数据写两次：字节完全一致
	p1 := filepath.Join(dir, "d1", "a.json")
	p2 := filepath.Join(dir, "d2", "b.json")
	require.NoError(t, SaveJSON(p1, build()))
	require.NoError(t, SaveJSON(p2, build()))
	b1, err := os.ReadFile(p1)
	require.NoError(t, err)
	b2, err := os.ReadFile(p2)
	require.NoError(t, err)
	assert.Equal(t, string(b1), string(b2), "同样的数据两次写出的字节不一致")

	// map key 按字典序排列
	assert.Less(t, strings.Index(string(b1), `"apple"`), strings.Index(string(b1), `"mango"`), "map key 未按字典序输出")
	assert.Less(t, strings.Index(string(b1), `"mango"`), strings.Index(string(b1), `"zebra"`), "map key 未按字典序输出")

	// 大整数经归一化后不丢精度
	assert.Contains(t, string(b1), "9007199254740993", "大整数精度丢失")
}

// 空 slice / 空 map 落盘为 [] / {} 而非 null，且 roundtrip 不因归一化破坏数据。
func TestSaveJSONEmptyCollections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")

	type conf struct {
		List []string       `json:"list"`
		KV   map[string]int `json:"kv"`
		Ptr  *conf          `json:"ptr"`
	}
	require.NoError(t, SaveJSON(path, conf{}), "SaveJSON 空集合失败")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "\"list\": []", "nil slice 应输出 []")
	assert.Contains(t, string(data), "\"kv\": {}", "nil map 应输出 {}")
	assert.Contains(t, string(data), "\"ptr\": null", "nil 指针保持 null")

	// 非空嵌套容器内的 nil slice/map 同样归一化
	nested := filepath.Join(dir, "nested.json")
	require.NoError(t, SaveJSON(nested, map[string]any{"inner": []any{map[string]any{"x": []int(nil)}}}))
	data, err = os.ReadFile(nested)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"x": []`, "嵌套 nil slice 应输出 []")

	// roundtrip：空集合读回为空集合，数据不丢
	got, err := LoadJSON[conf](path)
	require.NoError(t, err)
	assert.Empty(t, got.List)
	assert.Empty(t, got.KV)
	assert.Nil(t, got.Ptr)

	// 归一化不篡改调用方原值
	orig := conf{}
	require.NoError(t, SaveJSON(filepath.Join(dir, "orig.json"), orig))
	assert.Nil(t, orig.List, "归一化不应修改调用方的 nil slice")
	assert.Nil(t, orig.KV, "归一化不应修改调用方的 nil map")
}
