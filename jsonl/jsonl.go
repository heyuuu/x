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

// Write 将 values 逐行序列化后整体覆盖写入 path（不存在则创建，父目录自动递归创建）。
// tmp 文件 + rename 原子替换：中途失败不触碰旧文件，读者看到的要么是旧内容要么是完整新内容；
// values 为空时写入空文件。先全部序列化成功再落盘，序列化失败不产生副作用。
func Write(path string, values ...any) error {
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

	// 原子写入：tmp 落同目录以保证与目标同文件系统（rename 原子性前提），
	// 中途失败清理 tmp。
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp*")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		if tmpPath != "" {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(buf); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := tmp.Chmod(0644); err != nil {
		return fmt.Errorf("设置临时文件权限失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("重命名临时文件失败: %w", err)
	}
	tmpPath = "" // rename 已接管 tmp，跳过 defer 清理
	return nil
}
