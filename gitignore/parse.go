package gitignore

import (
	"slices"
	"strings"
)

// inclusionPrefix 是否定模式（negate）的行首前缀，使命中路径被显式重新包含。
const inclusionPrefix = "!"

// parsePatterns 解析一份 .gitignore 内容为 pattern 列表；
// domain 为该文件所在目录的规范路径，文件内的 pattern 相对该目录匹配。
func parsePatterns(doc string, domain string) []*pattern {
	var patterns []*pattern
	for _, line := range strings.Split(doc, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if strings.HasPrefix(line, "#") || len(strings.TrimSpace(line)) == 0 {
			continue
		}
		patterns = append(patterns, parsePattern(line, domain))
	}
	return patterns
}

// parsePattern 解析单行 pattern，语法依据 gitignore(5)，详见包内 RULES.md。
func parsePattern(p string, domain string) *pattern {
	d, ok := normalizePath(domain)
	if !ok {
		// 越根的 domain 原样保留：规范路径永不可能以其为前缀，规则永不命中
		d = domain
	}
	res := pattern{domain: d}

	if strings.HasPrefix(p, inclusionPrefix) {
		res.inclusion = true
		p = p[1:]
	}

	if !strings.HasSuffix(p, "\\ ") {
		p = strings.TrimRight(p, " ")
	}

	if strings.HasSuffix(p, patternDirSep) {
		res.dirOnly = true
		p = p[:len(p)-1]
	}

	if strings.Contains(p, patternDirSep) {
		res.isGlob = true
	}

	if p == "" {
		// 仅剩 "!" 或 "/" 的行：空 pattern 除空名外永不匹配。
		res.segments = []string{""}
		return &res
	}

	res.segments = strings.Split(p, patternDirSep)
	// 清除空段（如 "/foo" 的前导空段），语义不变。
	res.segments = slices.DeleteFunc(res.segments, func(seg string) bool { return seg == "" })
	return &res
}
