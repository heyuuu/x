package gitignore

import "unicode/utf8"

// 本文件实现段级通配匹配，语义遵循 fnmatch(3)（RULES.md 引用的基准）：
// * 匹配任意字符序列；? 匹配单个字符；[...] 字符类支持 ! 或 ^ 取反、
// 首 ] 作为字面成员、a-z 范围、\ 转义；\ 可转义任意字符（所有平台一致，
// 不依赖 filepath.Match——其在 Windows 上禁用转义、且不支持 ! 取反）。
// 未闭合字符类与尾部 \ 视为非法，永不匹配（与 git 实测行为一致）。

// matchSegment 判定单个路径段 name 是否匹配段 pattern。
func matchSegment(pat, name string) bool {
	if pat == "" {
		return name == ""
	}
	switch pat[0] {
	case '*':
		// 连续 * 等价单个 *（RULES.md「其他连续 * 按普通 * 处理」）
		for pat != "" && pat[0] == '*' {
			pat = pat[1:]
		}
		if pat == "" {
			return true
		}
		for i := 0; i <= len(name); {
			if matchSegment(pat, name[i:]) {
				return true
			}
			if i == len(name) {
				break
			}
			_, w := utf8.DecodeRuneInString(name[i:])
			i += w
		}
		return false
	case '?':
		if name == "" {
			return false
		}
		_, w := utf8.DecodeRuneInString(name)
		return matchSegment(pat[1:], name[w:])
	case '[':
		if name == "" {
			return false
		}
		rest, hit, ok := matchClass(pat, name)
		if !ok || !hit {
			return false
		}
		_, w := utf8.DecodeRuneInString(name)
		return matchSegment(rest, name[w:])
	case '\\':
		if len(pat) == 1 || name == "" || pat[1] != name[0] {
			return false
		}
		return matchSegment(pat[2:], name[1:])
	default:
		if name == "" || pat[0] != name[0] {
			return false
		}
		return matchSegment(pat[1:], name[1:])
	}
}

// matchClass 匹配 pat 开头的字符类，返回类之后的剩余 pattern 与是否命中；
// 未闭合的类返回 ok=false。
func matchClass(pat, name string) (rest string, hit bool, ok bool) {
	r, _ := utf8.DecodeRuneInString(name)
	i := 1
	negated := false
	if i < len(pat) && (pat[i] == '!' || pat[i] == '^') {
		negated = true
		i++
	}
	matched := false
	first := true
	for i < len(pat) {
		if pat[i] == ']' && !first {
			if negated {
				return pat[i+1:], !matched, true
			}
			return pat[i+1:], matched, true
		}
		first = false
		lo, n := classRune(pat, i)
		if n == 0 {
			return "", false, false
		}
		i += n
		hi := lo
		// 仅当 - 两侧都有成员且右侧不是 ] 时构成范围；首尾 - 为字面成员
		if i < len(pat) && pat[i] == '-' && i+1 < len(pat) && pat[i+1] != ']' {
			h, n2 := classRune(pat, i+1)
			if n2 == 0 {
				return "", false, false
			}
			hi = h
			i += 1 + n2
		}
		if lo <= r && r <= hi {
			matched = true
		}
	}
	return "", false, false
}

// classRune 读取类内一个成员字符；\ 转义下一字符，尾部 \ 非法返回 n=0。
func classRune(pat string, i int) (r rune, n int) {
	if i >= len(pat) {
		return 0, 0
	}
	if pat[i] == '\\' {
		if i+1 >= len(pat) {
			return 0, 0
		}
		r, w := utf8.DecodeRuneInString(pat[i+1:])
		return r, 1 + w
	}
	r, w := utf8.DecodeRuneInString(pat[i:])
	return r, w
}
