package gitignore

import "strings"

// matchResult 定义一次匹配判定的结果：未命中、排除或显式包含。
type matchResult int

const (
	noMatch matchResult = iota
	exclude
	include
)

const (
	// patternDirSep 是路径与 pattern 共用的分段分隔符。
	patternDirSep = "/"
	// zeroToManyDirs 独立成段时表示匹配零个或多个目录层。
	zeroToManyDirs = "**"
)

// pattern 是单条忽略规则，构造见 parse.go。所有路径均为规范路径（见 path.go）。
type pattern struct {
	domain    string // pattern 所属 .gitignore 的目录，匹配前先剥离
	segments  []string
	inclusion bool
	dirOnly   bool
	isGlob    bool
}

// match 判定 path 是否命中本规则，命中 path 的祖先段时视为整棵子树命中（子树覆盖）。
// 供不逐级判定祖先的调用方使用。path 必须是规范路径，由调用方归一化。
func (p *pattern) match(path string, isDir bool) matchResult {
	rel, ok := p.trimDomain(path)
	if !ok {
		return noMatch
	}

	if p.isGlob {
		return p.result(p.globMatch(rel, isDir))
	}
	return p.result(p.simpleNameMatch(rel, isDir))
}

// matchExact 判定 path 自身是否命中本规则：pattern 必须恰好落在整条路径上，
// 不因命中祖先段而连带命中其子孙（对照 match 的子树覆盖），等价 git 对单个路径的求值。
// 供逐级判定祖先的调用方使用：各层级各自 last-match-wins，子树覆盖会把祖先层级的
// 命中带到子孙层级，压过本该对子孙生效的 pattern——如 "*" + "!*/"，"!*/" 重新包含
// 目录 sub 后，sub/x 仍应由 "*" 排除。path 必须是规范路径，由调用方归一化。
func (p *pattern) matchExact(path string, isDir bool) matchResult {
	rel, ok := p.trimDomain(path)
	if !ok || (p.dirOnly && !isDir) {
		return noMatch
	}

	if p.isGlob {
		return p.result(matchSegments(p.segments, rel))
	}
	// 不含 "/" 的 pattern 只按末段（条目名）匹配。
	return p.result(matchSegment(p.segments[0], rel[strings.LastIndexByte(rel, '/')+1:]))
}

// trimDomain 剥离 path 的 domain 前缀，返回相对 pattern 所属目录的路径；
// path 不在 domain 之内（含恰为 domain 本身）时 ok 为 false。
func (p *pattern) trimDomain(path string) (rel string, ok bool) {
	if p.domain == "" {
		return path, true
	}
	if len(path) > len(p.domain) && path[len(p.domain)] == '/' && path[:len(p.domain)] == p.domain {
		return path[len(p.domain)+1:], true
	}
	return "", false
}

// result 将是否命中换算为判定结果：命中时按是否否定模式给出 include / exclude。
func (p *pattern) result(matched bool) matchResult {
	if !matched {
		return noMatch
	}
	if p.inclusion {
		return include
	}
	return exclude
}

// simpleNameMatch 处理不含 "/" 的 pattern：按名匹配任意层级的路径段。
// 命中中间段（即被忽略目录的祖先）时视为整棵子树命中。
func (p *pattern) simpleNameMatch(rel string, isDir bool) bool {
	for start := 0; start < len(rel); {
		idx := strings.IndexByte(rel[start:], '/')
		last := idx < 0
		end := len(rel)
		if !last {
			end = start + idx
		}
		if matchSegment(p.segments[0], rel[start:end]) {
			if p.dirOnly && !isDir && last {
				return false
			}
			return true
		}
		start = end + 1
	}
	return false
}

// globMatch 处理含 "/" 的 pattern（锚定于 domain 所在目录）：
// pattern 恰好耗尽 rel 的前 k 段即命中；不足整条时命中的是祖先目录，
// 其子孙随整棵子树忽略（祖先必为目录，dirOnly 亦满足）；
// 仅当耗尽整条 rel 且尾段是文件时，dirOnly pattern 不算命中。
func (p *pattern) globMatch(rel string, isDir bool) bool {
	for start := 0; ; {
		idx := strings.IndexByte(rel[start:], '/')
		last := idx < 0
		end := len(rel)
		if !last {
			end = start + idx
		}
		if matchSegments(p.segments, rel[:end]) {
			if !last || isDir || !p.dirOnly {
				return true
			}
			return false // 仅整条耗尽且尾段是文件的 dirOnly 不命中，且无更长前缀
		}
		if last {
			return false
		}
		start = end + 1
	}
}

// matchSegments 递归判定 pattern 段是否恰好耗尽 rel（不允许剩余）。
// 独立成段的 ** 在非末尾位置匹配零个或多个目录层（在所有段边界回溯）；
// 在末尾位置要求其后至少还有一个条目（"abc/**" 不匹配 "abc" 自身）。
// 其余段（含 a**b、*** 等内嵌连续星号）交给 matchSegment。
func matchSegments(pats []string, rel string) bool {
	if len(pats) == 0 {
		return rel == ""
	}
	if pats[0] == zeroToManyDirs {
		if len(pats) == 1 {
			return rel != ""
		}
		for cut := 0; ; {
			if matchSegments(pats[1:], rel[cut:]) {
				return true
			}
			idx := strings.IndexByte(rel[cut:], '/')
			if idx < 0 {
				return false
			}
			cut += idx + 1
		}
	}
	if rel == "" {
		return false
	}
	seg, rest := rel, ""
	if idx := strings.IndexByte(rel, '/'); idx >= 0 {
		seg, rest = rel[:idx], rel[idx+1:]
	}
	if !matchSegment(pats[0], seg) {
		return false
	}
	return matchSegments(pats[1:], rest)
}
