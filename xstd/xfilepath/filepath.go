package xfilepath

import (
	"path/filepath"
	"strings"
)

// HasPrefix 报告 path 是否为 prefix 本身或其下的路径（含相等）。
// 是 filepath.HasPrefix（已废弃）的安全替代：按路径段边界比较，/a/b 不会误判在
// /a/bb 之下，名为 ..foo 的目录不会被误判为父目录；纯词法比较，不解析符号链接、
// 不处理大小写不敏感文件系统；绝对/相对混合或跨卷（Windows）时返回 false。
func HasPrefix(path, prefix string) bool {
	rel, err := filepath.Rel(prefix, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, "../")
}
