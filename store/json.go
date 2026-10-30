package store

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
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
		return fmt.Errorf("序列化 JSON 失败: %w", err)
	}
	return WriteFileAtomic(path, append(data, '\n'), 0644)
}

// LoadJSON 读取并反序列化 path 的 JSON 内容。
// 文件缺失返回 (零值, ErrFileMissing)；解析失败原样上抛，由调用方决定降级策略。
func LoadJSON[T any](path string) (T, error) {
	var zero T
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return zero, ErrFileMissing
		}
		return zero, fmt.Errorf("读取 JSON 文件失败: %w", err)
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return zero, fmt.Errorf("解析 JSON 文件失败: %w", err)
	}
	return v, nil
}
