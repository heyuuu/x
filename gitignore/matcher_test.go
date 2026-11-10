package gitignore

import (
	"fmt"
	"io/fs"
	"testing"
)

// memFS 是内存假文件系统：key 为规范路径（"/" 分隔，根为 ".gitignore"）。
type memFS map[string]string

var _ FS = memFS{}

func (m memFS) OpenFile(path string) (string, error) {
	content, ok := m[path]
	if !ok {
		return "", fmt.Errorf("%s: %w", path, fs.ErrNotExist)
	}
	return content, nil
}

// countingFS 包装 memFS，按文件统计 OpenFile 调用次数。
type countingFS struct {
	memFS
	opens map[string]int
}

func (c *countingFS) OpenFile(path string) (string, error) {
	c.opens[path]++
	return c.memFS.OpenFile(path)
}

// TestStaticMatcher pin 住静态判定：同一列表内 last-match-wins，否定模式生效，
// domain 锚定规则只匹配该目录内的路径。
func TestStaticMatcher(t *testing.T) {
	matcher := NewStaticMatcher("", "*.log\n!important.log\n")

	if !matcher.Match("a/debug.log", false) {
		t.Fatal("a/debug.log 应被 *.log 忽略")
	}
	if matcher.Match("a/important.log", false) {
		t.Fatal("a/important.log 应被 !important.log 重新包含")
	}
	if matcher.Match("a/main.go", false) {
		t.Fatal("a/main.go 不应被忽略")
	}

	sub := NewStaticMatcher("sub", "build/\n")
	if !sub.Match("sub/build/x", false) {
		t.Fatal("sub/build/x 应被 domain=sub 的 build/ 忽略")
	}
	if sub.Match("build/x", false) {
		t.Fatal("domain 外的 build/x 不应被忽略")
	}
}

func TestAutoloadMatcher(t *testing.T) {
	t.Run("deep gitignore overrides shallow", func(t *testing.T) {
		// pins：深层 .gitignore 的 pattern 覆盖浅层（RULES.md 优先级第 2 条）。
		files := memFS{
			".gitignore":   "*.log\n!keep.log\n",
			"a/.gitignore": "keep.log\n",
		}
		matcher := NewAutoloadMatcher(files)

		if !matcher.Match("a/debug.log", false) {
			t.Fatal("a/debug.log 应被根 *.log 忽略")
		}
		if !matcher.Match("a/keep.log", false) {
			t.Fatal("a/keep.log 应被 a/.gitignore 的 keep.log 覆盖回忽略")
		}
		if matcher.Match("b/keep.log", false) {
			t.Fatal("b/keep.log 只受根规则约束，应被 !keep.log 重新包含")
		}
	})

	t.Run("base patterns are lowest priority", func(t *testing.T) {
		// pins：base（树外来源，如 info/exclude）优先级低于树内所有 .gitignore。
		files := memFS{".gitignore": "!a.tmp\n"}
		matcher := NewAutoloadMatcher(files, "a.tmp\nb.tmp\n")

		if matcher.Match("a.tmp", false) {
			t.Fatal("a.tmp 应被根 !a.tmp 覆盖，不被忽略")
		}
		if !matcher.Match("b.tmp", false) {
			t.Fatal("b.tmp 应被 base 的 b.tmp 忽略")
		}
	})

	t.Run("excluded parent cannot be re-included", func(t *testing.T) {
		// pins：RULES.md 否定限制——父目录被排除后，! 无法重新包含其子孙。
		// AutoloadMatcher 逐级判定祖先，语义等价 git check-ignore。
		files := memFS{".gitignore": "build/\n!build/keep.txt\n"}
		matcher := NewAutoloadMatcher(files)

		if !matcher.Match("build", true) {
			t.Fatal("build 目录应被 build/ 忽略")
		}
		if !matcher.Match("build/keep.txt", false) {
			t.Fatal("父目录 build 已被排除，!build/keep.txt 不应重新包含")
		}

		// 对照：StaticMatcher 不判定祖先，同规则下按 last-match-wins 重新包含。
		static := NewStaticMatcher("", "build/\n!build/keep.txt\n")
		if static.Match("build/keep.txt", false) {
			t.Fatal("StaticMatcher 下 !build/keep.txt 应重新包含（不忽略）")
		}
	})

	t.Run("negation re-includes only the matched path", func(t *testing.T) {
		// pins：! 只重新包含命中的路径自身，不连带其子孙。白名单写法（* 排除一切，
		// !*/ 放行目录，!*.go 放行 Go 文件）下，目录被重新包含后，其中的文件仍各自
		// 由 * 排除，除非自身被 ! 命中。行为与 git check-ignore 实测一致。
		files := memFS{".gitignore": "*\n!*/\n!*.go\n"}
		matcher := NewAutoloadMatcher(files)

		if matcher.Match("cmd", true) || matcher.Match("cmd/x", true) {
			t.Fatal("目录应被 !*/ 重新包含")
		}
		if matcher.Match("cmd/x/main.go", false) {
			t.Fatal("cmd/x/main.go 应被 !*.go 重新包含")
		}
		if !matcher.Match("cmd/x/binary", false) {
			t.Fatal("cmd/x/binary 应被 * 排除：!*/ 只重新包含目录自身，不连带其中的文件")
		}
		if !matcher.Match("top.bin", false) {
			t.Fatal("top.bin 应被 * 排除")
		}
	})

	t.Run("deep negation does not leak into descendants", func(t *testing.T) {
		// pins：深层 .gitignore 的 ! 命中某目录时，不会连带压过浅层规则对该目录子孙的排除。
		files := memFS{
			".gitignore":   "docs\n",
			"a/.gitignore": "!?\n",
		}
		matcher := NewAutoloadMatcher(files)

		if matcher.Match("a/b", true) {
			t.Fatal("a/b 不应被忽略")
		}
		if !matcher.Match("a/b/docs", false) {
			t.Fatal("a/b/docs 应被根 docs 排除：!? 命中的是目录 b 自身，不连带其子孙")
		}
	})

	t.Run("re-included dir frees its descendants", func(t *testing.T) {
		// pins：目录被 ! 重新包含后，先前排除它的 pattern 不再牵连其子孙
		// （排除 pattern 同样只对路径自身求值）。
		files := memFS{".gitignore": "build\n!build\n"}
		matcher := NewAutoloadMatcher(files)

		if matcher.Match("build", true) {
			t.Fatal("build 应被 !build 重新包含")
		}
		if matcher.Match("build/x", false) {
			t.Fatal("build/x 不应被忽略：build 已被重新包含，x 自身不命中任何 pattern")
		}
	})

	t.Run("no gitignore means nothing ignored", func(t *testing.T) {
		matcher := NewAutoloadMatcher(memFS{})
		if matcher.Match("a/b.log", false) {
			t.Fatal("无任何 .gitignore 时不应忽略任何路径")
		}
	})

	t.Run("input paths are normalized", func(t *testing.T) {
		// pins：入口统一规范化，各种写法等价；根路径本身不会被忽略。
		files := memFS{".gitignore": "*.log\n"}
		matcher := NewAutoloadMatcher(files)

		for _, variant := range []string{"a/b.log", "./a/b.log", "a//b.log", "a/b.log/", "/a/b.log"} {
			if !matcher.Match(variant, false) {
				t.Fatalf("%q 应规范化后命中 *.log", variant)
			}
		}
		for _, variant := range []string{"", ".", "/"} {
			if matcher.Match(variant, false) {
				t.Fatalf("根路径 %q 不应被忽略", variant)
			}
		}
		// pins：越过判定根的路径在判定范围之外，即使字面上能命中规则也不忽略。
		for _, variant := range []string{"../b.log", "a/../../b.log"} {
			if matcher.Match(variant, false) {
				t.Fatalf("越根路径 %q 不应被忽略", variant)
			}
		}
	})
}

// TestAutoloadMatcherCache pin 住缓存行为：每个目录的 .gitignore 只经 fs 加载一次，
// 不存在与存在同样只读一次。
func TestAutoloadMatcherCache(t *testing.T) {
	fsys := &countingFS{
		memFS: memFS{
			".gitignore":        "x\n",
			"a/.gitignore":      "y\n",
			"a/b/.gitignore":    "z\n",
			"a/miss/.gitignore": "w\n",
		},
		opens: make(map[string]int),
	}
	matcher := NewAutoloadMatcher(fsys)

	matcher.Match("a/b/c", false)
	matcher.Match("a/b/d", false)
	matcher.Match("a/miss/e", false)
	matcher.Match("a/miss/f", false)

	for _, key := range []string{".gitignore", "a/.gitignore", "a/b/.gitignore", "a/miss/.gitignore"} {
		if got := fsys.opens[key]; got != 1 {
			t.Fatalf("%s 加载了 %d 次，应为 1", key, got)
		}
	}
}

// benchDoc 是一份典型规模的根 .gitignore（15 条规则，混合文件/目录/通配）。
const benchDoc = "*.log\n*.tmp\nbin/\nobj/\nnode_modules/\n*.test\ncoverage/\n.idea/\n*.swp\ndist/\nvendor/\n*.out\n__pycache__/\n.DS_Store\n*.bak\n"

// benchFiles 构造带层级 .gitignore 的典型仓库形态。
// 注意规则不能命中路径自身的祖先段（如 cmd/ 下写 "x" 会让 cmd/x 整棵提前剪枝），
// 否则测不到全深度判定的最坏路径。
func benchFiles() memFS {
	return memFS{
		".gitignore":          benchDoc,
		"cmd/.gitignore":      "local/\n",
		"cmd/x/.gitignore":    "testdata/\n",
		"internal/.gitignore": "gen/\n",
	}
}

// BenchmarkStaticMatcher 基线：无 autoload 机制，直接在 pattern 列表上判定。
func BenchmarkStaticMatcher(b *testing.B) {
	matcher := NewStaticMatcher("", benchDoc)

	b.Run("shallow file", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			matcher.Match("cmd/x/main.go", false)
		}
	})
}

// BenchmarkAutoloadMatcher 压测忽略判定的稳态开销（各级 .gitignore 已全部缓存），
// 场景取自典型遍历：根级文件、浅层源码文件、深层未命中文件（最坏路径）、
// 顶层即被排除的路径（遍历剪枝）。
func BenchmarkAutoloadMatcher(b *testing.B) {
	matcher := NewAutoloadMatcher(benchFiles())

	bench := func(name string, path string, isDir bool) {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				matcher.Match(path, isDir)
			}
		})
	}

	bench("root level file", "main.go", false)
	bench("shallow file", "cmd/x/main.go", false)
	bench("deep file worst case", "cmd/x/internal/a/b/c/d/e.go", false)
	bench("excluded top dir", "node_modules/left-pad/index.js", false)
}
