package gitignore

import (
	"slices"
	"strings"
	"sync"
)

// Matcher 判定一条相对忽略判定根的路径是否被忽略。
// path 为 "/" 分隔的相对路径，入口统一规范化（见 path.go），根路径本身不会被忽略。
type Matcher interface {
	Match(path string, isDir bool) bool
}

// StaticMatcher 基于构造时给定的规则文本判定，不访问文件系统；
// 规则按传入顺序拼接，后出现的 pattern 覆盖先出现的（last-match-wins）。
// 只判定路径本身，祖先目录的忽略需调用方自行逐级判定。
type StaticMatcher struct {
	patterns []*pattern
}

var _ Matcher = (*StaticMatcher)(nil)

// NewStaticMatcher 构造静态 Matcher；doc 为一份规则文本（如某个 .gitignore
// 文件的内容），domain 为该文本所在目录的规范路径（根为 ""），规则相对该目录匹配。
func NewStaticMatcher(domain string, doc string) StaticMatcher {
	return StaticMatcher{
		patterns: parsePatterns(doc, domain)}
}

// Match 判定路径是否被忽略；只对路径本身求值，不判定祖先目录。
// 越过判定根的路径（如 "../x"）不在判定范围内，一律不忽略。
func (s StaticMatcher) Match(path string, isDir bool) bool {
	path, ok := normalizePath(path)
	if !ok {
		return false
	}
	return matchPatterns(s.patterns, path, isDir) == exclude
}

// AutoloadMatcher 在判定时经 fs 按需加载路径各级祖先目录的 .gitignore（含根）
// 并缓存；同时自根向叶逐级判定祖先目录，任一祖先被排除即整体忽略，
// 等价 git check-ignore 语义：父目录被排除后，! 否定模式无法重新包含其子孙。
type AutoloadMatcher struct {
	fs       FS
	base     []*pattern
	mutex    sync.Mutex
	patterns map[string][]*pattern
}

var _ Matcher = (*AutoloadMatcher)(nil)

// NewAutoloadMatcher 构造自动加载 Matcher。fs 的根即忽略判定的根（如工作树根）；
// base 为若干份树外来源的规则文本（如 info/exclude、core.excludesFile 文件的
// 内容），优先级低于树内所有 .gitignore。
func NewAutoloadMatcher(fs FS, base ...string) *AutoloadMatcher {
	return &AutoloadMatcher{
		fs:       fs,
		base:     parseDocs(base),
		patterns: make(map[string][]*pattern),
	}
}

// Match 判定路径是否被忽略：先逐级判定祖先目录（任一被排除即整体忽略），
// 再判定路径本身。越过判定根的路径（如 "../x"）不在判定范围内，一律不忽略。
func (a *AutoloadMatcher) Match(path string, isDir bool) bool {
	path, ok := normalizePath(path)
	if !ok || path == "" {
		return false
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	// 自根向叶单趟推进：逐级并入祖先目录（path[:cut] 为目录前缀子串，零分配）
	// 的 pattern，并判定当前前缀是否作为目录被排除。pattern 顺序为
	// base → 根 → 深层，matchPatterns 逆序取最后命中，即深层 .gitignore
	// 覆盖浅层、同一文件内 last-match-wins；.gitignore 不存在或不可读视为
	// 无 pattern，结果同样缓存。
	patterns := slices.Clone(a.base)
	cut := 0 // 当前目录前缀（含若干整段，不含尾分隔符）的末尾偏移
	for {
		dir := path[:cut]
		loaded, ok := a.patterns[dir]
		if !ok {
			loaded = loadIgnoreFile(a.fs, dir)
			a.patterns[dir] = loaded
		}
		patterns = append(patterns, loaded...)

		idx := strings.IndexByte(path[cut+1:], '/')
		if idx < 0 {
			return matchPatterns(patterns, path, isDir) == exclude
		}
		end := cut + 1 + idx // 判定中的祖先前缀（含当前段）的末尾
		if matchPatterns(patterns, path[:end], true) == exclude {
			return true
		}
		cut = end
	}
}

// matchPatterns 在 pattern 列表上判定，取最后一个命中的结果（last-match-wins）。
func matchPatterns(patterns []*pattern, path string, isDir bool) matchResult {
	for i := len(patterns) - 1; i >= 0; i-- {
		if match := patterns[i].match(path, isDir); match != noMatch {
			return match
		}
	}
	return noMatch
}

// parseDocs 解析多份根目录规则文本为按序拼接的 pattern 列表，
// 供 AutoloadMatcher 的树外 base 来源使用。
func parseDocs(docs []string) []*pattern {
	var patterns []*pattern
	for _, doc := range docs {
		patterns = append(patterns, parsePatterns(doc, "")...)
	}
	return patterns
}

// gitignoreFile 是各级目录中忽略规则文件的固定文件名。
const gitignoreFile = ".gitignore"

// loadIgnoreFile 读取 dir（规范路径，根为 ""）下 .gitignore 的 pattern 列表；
// 文件不存在或不可读时视为无 pattern（gitignore 文件本是可选的）。
func loadIgnoreFile(fs FS, dir string) []*pattern {
	name := gitignoreFile
	if dir != "" {
		name = dir + "/" + gitignoreFile
	}
	content, err := fs.OpenFile(name)
	if err != nil {
		return nil
	}
	return parsePatterns(content, dir)
}
