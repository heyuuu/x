package gitignore

import (
	"os"
	"path/filepath"
)

// FS 抽象忽略规则文件的读取。path 为相对判定根的规范路径
// （"/" 分隔、根为空串，见 path.go），实现负责翻译为所在平台的实际路径。
type FS interface {
	OpenFile(path string) (string, error)
}

// LocalFS 基于本地文件系统实现 FS：路径相对 Root 解析，Root 即忽略判定的根
// 所在目录（如工作树根）；Root 为空时相对进程工作目录解析。
type LocalFS struct {
	Root string
}

var _ FS = LocalFS{}

// OpenFile 读取 Root 下的文件内容；Windows 上将 "/" 翻译为平台分隔符。
func (fs LocalFS) OpenFile(path string) (string, error) {
	b, err := os.ReadFile(filepath.Join(fs.Root, filepath.FromSlash(path)))
	if err != nil {
		return "", err
	}
	return string(b), nil
}
