package jsonl

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
)

// Append 序列化 v 为单行 JSON 追加写入 path（不存在则创建，父目录自动递归创建）。
// O_APPEND 小块写由 POSIX 保证原子，无需跨进程锁（单写者追加语义）。
func Append(path string, values ...any) error {
	// 序列化
	var buf []byte
	for _, value := range values {
		line, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("序列化 JSONL 行失败: %w", err)
		}
		buf = append(buf, line...)
		buf = append(buf, '\n')
	}

	// 打开文件
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("打开 JSONL 文件失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	// 坏行修复：上次写中断可能留下无换行结尾的残缺行，
	// 先补一个 '\n' 隔断，避免本次内容拼进残缺行（O_APPEND 保证写到文件尾）。
	if stat, err := f.Stat(); err != nil {
		return fmt.Errorf("读取 JSONL 文件信息失败: %w", err)
	} else if stat.Size() > 0 {
		var last [1]byte
		if _, err := f.ReadAt(last[:], stat.Size()-1); err != nil {
			return fmt.Errorf("检查 JSONL 文件末尾失败: %w", err)
		}
		if last[0] != '\n' {
			if _, err := f.Write([]byte{'\n'}); err != nil {
				return fmt.Errorf("修复 JSONL 坏行失败: %w", err)
			}
		}
	}

	// 写入
	if _, err := f.Write(buf); err != nil {
		return fmt.Errorf("追加 JSONL 行失败: %w", err)
	}

	return nil
}
