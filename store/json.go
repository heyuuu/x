package store

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"os"
)

// saveOpts 用 json/v2 的 Deterministic 保证同样的数据每次写出字节稳定
// （map key 排序），且 nil slice/map 默认输出 [] / {} 而非 null。
var saveOpts = []json.Options{json.Deterministic(true), jsontext.Multiline(true), jsontext.WithIndent("  ")}

// SaveJSON 序列化 v 并原子写入 path。
// 用缩进 + 尾换行——JSON 存储文件常被人工翻看，可读性优先于体积。
func SaveJSON(path string, v any) error {
	data, err := json.Marshal(v, saveOpts...)
	if err != nil {
		return jsonMarshalFailed(path, err)
	}
	return WriteFileAtomic(path, append(data, '\n'), 0644)
}

// LoadJSON 读取并反序列化 path 的 JSON 内容。
// 失败形态经哨兵错误上抛，由调用方决定降级策略：
// 文件缺失 → ErrStoreMissing；读取失败 → ErrStoreLoadFailed；解析失败 → ErrUnmarshalFailed。
func LoadJSON[T any](path string) (T, error) {
	var zero T
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return zero, jsonStoreMissing(path)
		}
		return zero, jsonStoreLoadFailed(path)
	}

	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return zero, jsonUnmarshalFailed(path, err)
	}
	return v, nil
}
