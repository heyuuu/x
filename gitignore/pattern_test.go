package gitignore

import "testing"

// TestPatternMatch 以表驱动 pin 住单条 pattern 的匹配语义（锚定、dirOnly、
// 子树覆盖、**、转义、domain 剥离），用例取自 RULES.md（整理自 gitignore(5)）。
// path 传入 match 前即为规范路径（matcher 层入口归一化，这里直接给定）。
func TestPatternMatch(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		domain  string
		path    string
		isDir   bool
		want    matchResult
	}{
		// 锚定 vs 任意层级（RULES.md「尾部 / 的匹配示例」）
		{name: "anchored dir matches itself", pattern: "doc/frotz/", path: "doc/frotz", isDir: true, want: exclude},
		{name: "anchored dir rejects deeper level", pattern: "doc/frotz/", path: "a/doc/frotz", isDir: true, want: noMatch},
		{name: "unanchored dir matches root", pattern: "frotz/", path: "frotz", isDir: true, want: exclude},
		{name: "unanchored dir matches any level", pattern: "frotz/", path: "a/frotz", isDir: true, want: exclude},

		// dirOnly：只匹配目录，但被忽略目录的子孙随子树命中
		{name: "dir only rejects file", pattern: "frotz/", path: "frotz", isDir: false, want: noMatch},
		{name: "dir only rejects file at any level", pattern: "logs/", path: "a/logs", isDir: false, want: noMatch},
		{name: "dir only covers subtree", pattern: "frotz/", path: "a/frotz/x", isDir: false, want: exclude},

		// 不含 /：按名匹配任意层级，文件目录皆可
		{name: "plain name matches file at any level", pattern: "frotz", path: "a/frotz", isDir: false, want: exclude},

		// 前导 / 锚定根目录
		{name: "leading slash matches root", pattern: "/foo", path: "foo", isDir: false, want: exclude},
		{name: "leading slash rejects deeper level", pattern: "/foo", path: "a/foo", isDir: false, want: noMatch},

		// 含中间 /：锚定，命中祖先段时整棵子树命中
		{name: "anchored path covers subtree", pattern: "a/b", path: "a/b/c", isDir: false, want: exclude},
		{name: "anchored path rejects misplaced", pattern: "a/b", path: "x/a/b", isDir: false, want: noMatch},

		// **（RULES.md「双星号规则」）
		{name: "leading double star matches root", pattern: "**/foo", path: "foo", isDir: false, want: exclude},
		{name: "leading double star matches any level", pattern: "**/foo", path: "a/b/foo", isDir: false, want: exclude},
		{name: "leading double star requires direct parent", pattern: "**/foo/bar", path: "a/foo/bar", isDir: false, want: exclude},
		{name: "leading double star rejects indirect parent", pattern: "**/foo/bar", path: "foo/x/bar", isDir: false, want: noMatch},
		{name: "trailing double star rejects dir itself", pattern: "abc/**", path: "abc", isDir: true, want: noMatch},
		{name: "trailing double star matches inside", pattern: "abc/**", path: "abc/x", isDir: false, want: exclude},
		{name: "middle double star matches zero dirs", pattern: "a/**/b", path: "a/b", isDir: true, want: exclude},
		{name: "middle double star matches many dirs", pattern: "a/**/b", path: "a/x/y/b", isDir: true, want: exclude},
		{name: "middle double star backtracks", pattern: "a/**/b/c", path: "a/b/x/b/c", isDir: false, want: exclude},
		{name: "embedded double star acts as single star", pattern: "a/**b", path: "a/xxb", isDir: false, want: exclude},
		{name: "triple star acts as single star", pattern: "***", path: "anything", isDir: false, want: exclude},

		// 转义在匹配中的表现（段级语义见 segment_test.go，解析结构见 parse_test.go）
		{name: "escaped hash matches literal", pattern: `\#foo`, path: "#foo", isDir: false, want: exclude},
		{name: "escaped bang matches literal", pattern: `\!important!.txt`, path: "!important!.txt", isDir: false, want: exclude},
		{name: "negation prefix includes", pattern: "!foo", path: "foo", isDir: false, want: include},
		{name: "bang negated class excludes listed chars", pattern: "[!b]at", path: "cat", isDir: false, want: exclude},
		{name: "bang negated class rejects listed chars", pattern: "[!b]at", path: "bat", isDir: false, want: noMatch},
		{name: "escaped trailing space is kept", pattern: `keep\ `, path: "keep ", isDir: false, want: exclude},
		{name: "escaped trailing space rejects bare name", pattern: `keep\ `, path: "keep", isDir: false, want: noMatch},
		{name: "trailing backslash never matches", pattern: `foo\`, path: "foo", isDir: false, want: noMatch},

		// domain：pattern 相对所属 .gitignore 的目录匹配
		{name: "domain prefix is stripped", pattern: "foo", domain: "a", path: "a/foo", isDir: false, want: exclude},
		{name: "domain mismatch rejects", pattern: "foo", domain: "a", path: "b/foo", isDir: false, want: noMatch},
		{name: "path no deeper than domain rejects", pattern: "foo", domain: "a", path: "a", isDir: true, want: noMatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePattern(tt.pattern, tt.domain).match(tt.path, tt.isDir)
			if got != tt.want {
				t.Fatalf("parsePattern(%q, %q).match(%q, %v) = %v, want %v",
					tt.pattern, tt.domain, tt.path, tt.isDir, got, tt.want)
			}
		})
	}
}

// TestPatternMatchExact pin 住 matchExact 只对路径自身求值：pattern 必须恰好落在
// 整条路径上，命中祖先段不连带子孙（对照 TestPatternMatch 中的 covers subtree 用例）。
func TestPatternMatchExact(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		domain  string
		path    string
		isDir   bool
		want    matchResult
	}{
		// 不含 /：只按末段（条目名）匹配
		{name: "plain name matches entry at any level", pattern: "frotz", path: "a/frotz", isDir: false, want: exclude},
		{name: "plain name rejects descendant", pattern: "frotz", path: "a/frotz/x", isDir: false, want: noMatch},
		{name: "dir only matches dir", pattern: "frotz/", path: "a/frotz", isDir: true, want: exclude},
		{name: "dir only rejects file", pattern: "frotz/", path: "a/frotz", isDir: false, want: noMatch},
		{name: "dir only rejects descendant", pattern: "frotz/", path: "a/frotz/x", isDir: false, want: noMatch},

		// 含 /：锚定，须耗尽整条路径
		{name: "anchored path matches itself", pattern: "a/b", path: "a/b", isDir: false, want: exclude},
		{name: "anchored path rejects descendant", pattern: "a/b", path: "a/b/c", isDir: false, want: noMatch},
		{name: "anchored dir only rejects file", pattern: "a/b/", path: "a/b", isDir: false, want: noMatch},
		{name: "trailing double star matches any depth inside", pattern: "abc/**", path: "abc/x/y", isDir: false, want: exclude},
		{name: "trailing double star rejects dir itself", pattern: "abc/**", path: "abc", isDir: true, want: noMatch},
		{name: "middle double star rejects descendant", pattern: "a/**/b", path: "a/x/b/c", isDir: false, want: noMatch},

		// 否定模式同样只命中路径自身
		{name: "negation includes matched path", pattern: "!sub/", path: "sub", isDir: true, want: include},
		{name: "negation rejects descendant", pattern: "!sub/", path: "sub/x", isDir: false, want: noMatch},

		// domain：剥离后再按上述规则匹配
		{name: "domain prefix is stripped", pattern: "foo", domain: "a", path: "a/b/foo", isDir: false, want: exclude},
		{name: "domain rejects descendant", pattern: "foo", domain: "a", path: "a/foo/b", isDir: false, want: noMatch},
		{name: "domain mismatch rejects", pattern: "foo", domain: "a", path: "b/foo", isDir: false, want: noMatch},
		{name: "path no deeper than domain rejects", pattern: "*", domain: "a", path: "a", isDir: true, want: noMatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePattern(tt.pattern, tt.domain).matchExact(tt.path, tt.isDir)
			if got != tt.want {
				t.Fatalf("parsePattern(%q, %q).matchExact(%q, %v) = %v, want %v",
					tt.pattern, tt.domain, tt.path, tt.isDir, got, tt.want)
			}
		})
	}
}
