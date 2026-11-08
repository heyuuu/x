package xos

import (
	"os"
	"path/filepath"
)

// PathExists 判断路径存在（文件或目录）。
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsFile 判断存在且是普通文件，符号链接跟随目标（os.Stat 语义）。
func IsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// IsDir 判断存在且是目录。
func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// WriteFile 写文件，父目录不存在时自动创建（新建目录权限 0755）。
func WriteFile(path string, content []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, content, perm)
}
