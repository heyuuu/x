package gitignore

import (
	"path"
	"strings"
)

// normalizePath 将输入路径规范化为包内规范形式：相对忽略判定根、"/" 分隔、
// 无前导 "./" 或 "/"、无尾随 "/"、无空段，根为空串；已规范的输入零拷贝返回。
// 路径越过判定根（如 "../x"、"a/../../x"）返回 ok=false：此类路径在忽略
// 判定范围之外，应直接视为不被忽略（git check-ignore 对仓库外路径同样拒绝）。
// 不转换反斜杠——gitignore 语法中 "\" 是转义符，静默替换会引入语义歧义。
func normalizePath(p string) (string, bool) {
	if p == "" {
		return "", true
	}

	p = path.Clean(p)
	if p == ".." || strings.HasPrefix(p, "../") {
		return "", false
	}
	if p == "." {
		return "", true
	}
	return strings.TrimPrefix(p, "/"), true
}
