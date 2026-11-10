package finder

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// absDir 返回测试数据目录的绝对路径。
func absDir(t *testing.T, dir string) string {
	t.Helper()
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func collectPaths(t *testing.T, root string, opts []Option) ([]string, error) {
	t.Helper()
	var paths []string
	var firstErr error
	for entry, err := range Find(root, opts...) {
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		paths = append(paths, entry.RelPath())
	}
	return paths, firstErr
}

func TestFind(t *testing.T) {
	tests := []struct {
		name    string
		root    string
		opts    []Option
		want    []string
		wantErr bool
	}{
		{
			name: "default includes files and directories",
			root: absDir(t, "testdata/basic"),
			want: []string{"a.txt", "dir", "dir/b.txt", "dir/sub", "dir/sub/c.txt"},
		},
		{
			name: "files",
			root: absDir(t, "testdata/basic"),
			opts: []Option{OptionFiles()},
			want: []string{"a.txt", "dir/b.txt", "dir/sub/c.txt"},
		},
		{
			name: "directories",
			root: absDir(t, "testdata/basic"),
			opts: []Option{OptionDirectories()},
			want: []string{"dir", "dir/sub"},
		},
		{
			name: "filter",
			root: absDir(t, "testdata/basic"),
			opts: []Option{
				OptionFilter(func(entry FileEntry) bool { return !strings.HasPrefix(entry.RelPath(), "dir") }),
			},
			want: []string{"a.txt"},
		},
		{
			// pins：目录被 filter 拒绝时整棵子树剪掉。
			// filter 只拒绝名为 dir 的目录本身，其子条目（dir/b.txt 等并不匹配该条件）
			// 也不得出现在结果中。
			name: "filtered dir prunes subtree",
			root: absDir(t, "testdata/basic"),
			opts: []Option{
				OptionFilter(func(entry FileEntry) bool {
					return !entry.IsDir() || entry.Name() != "dir"
				}),
			},
			want: []string{"a.txt"},
		},
		{
			// 多个 filter 串联执行，任一不满足即跳过（AND 语义）。
			name: "multiple filters",
			root: absDir(t, "testdata/basic"),
			opts: []Option{
				OptionFilter(func(entry FileEntry) bool { return strings.HasSuffix(entry.Name(), ".txt") }),
				OptionFilter(func(entry FileEntry) bool { return strings.HasPrefix(entry.Name(), "a") }),
				OptionFilter(func(entry FileEntry) bool { return !entry.IsDir() }),
			},
			want: []string{"a.txt"},
		},
		{
			name:    "nonexistent root yields error",
			root:    absDir(t, "testdata/nonexistent"),
			wantErr: true,
		},
		{
			name:    "relative root yields error",
			root:    "testdata/basic",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths, firstErr := collectPaths(t, tt.root, tt.opts)

			if !slices.Equal(paths, tt.want) {
				t.Fatalf("got %v, want %v", paths, tt.want)
			}
			if tt.wantErr {
				if firstErr == nil || (!errors.Is(firstErr, fs.ErrNotExist) && !errors.Is(firstErr, ErrRelativePath)) {
					t.Fatalf("got err %v, want fs.ErrNotExist or ErrRelativePath", firstErr)
				}
			} else if firstErr != nil {
				t.Fatalf("unexpected err %v", firstErr)
			}
		})
	}
}

// TestFindDirsFiles pins 住快捷函数只产出对应类型的条目。
func TestFindDirsFiles(t *testing.T) {
	root := absDir(t, "testdata/basic")

	var dirs, files []string
	for entry, err := range FindDirs(root) {
		if err != nil {
			t.Fatal(err)
		}
		dirs = append(dirs, entry.RelPath())
	}
	for entry, err := range FindFiles(root) {
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, entry.RelPath())
	}

	if want := []string{"dir", "dir/sub"}; !slices.Equal(dirs, want) {
		t.Fatalf("FindDirs: got %v, want %v", dirs, want)
	}
	if want := []string{"a.txt", "dir/b.txt", "dir/sub/c.txt"}; !slices.Equal(files, want) {
		t.Fatalf("FindFiles: got %v, want %v", files, want)
	}
}

// TestIgnoreVCS pins 住 VCS 目录过滤：.git/.hg/.svn 默认连同子树整棵跳过，
// OptionIgnoreVCS(false) 时正常产出。
// 用 t.TempDir 动态构造：git 无法追踪路径中含 .git 组件的文件，testdata 方案在全新 clone 后会静默失效。
func TestIgnoreVCS(t *testing.T) {
	newFixture := func(t *testing.T) string {
		t.Helper()
		tmp := t.TempDir()
		for _, vcs := range []string{".git", ".hg", ".svn"} {
			writeFile(t, tmp, vcs+"/config", "")
		}
		writeFile(t, tmp, "a.txt", "")
		return tmp
	}

	t.Run("default ignores vcs dirs", func(t *testing.T) {
		tmp := newFixture(t)
		paths, firstErr := collectPaths(t, tmp, nil)
		if firstErr != nil {
			t.Fatal(firstErr)
		}
		if want := []string{"a.txt"}; !slices.Equal(paths, want) {
			t.Fatalf("got %v, want %v", paths, want)
		}
	})

	t.Run("OptionIgnoreVCS(false) keeps vcs dirs", func(t *testing.T) {
		tmp := newFixture(t)
		paths, firstErr := collectPaths(t, tmp, []Option{OptionIgnoreVCS(false)})
		if firstErr != nil {
			t.Fatal(firstErr)
		}
		want := []string{".git", ".git/config", ".hg", ".hg/config", ".svn", ".svn/config", "a.txt"}
		if !slices.Equal(paths, want) {
			t.Fatalf("got %v, want %v", paths, want)
		}
	})
}

// TestReadErrorDoesNotStopSequence pins 住产出错误后序列继续：
// 子目录不可读产出一个错误，兄弟条目仍正常遍历到。
func TestReadErrorDoesNotStopSequence(t *testing.T) {
	skipIfNoFilePerms(t)

	tmp := t.TempDir()
	writeFile(t, tmp, "a.txt", "")
	writeFile(t, tmp, "locked/secret.txt", "")
	writeFile(t, tmp, "z.txt", "")
	locked := filepath.Join(tmp, "locked")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	var paths []string
	var errCount int
	for entry, err := range Find(tmp) {
		if err != nil {
			errCount++
			continue
		}
		paths = append(paths, entry.RelPath())
	}

	if errCount != 1 {
		t.Fatalf("got %d errors, want 1", errCount)
	}
	if want := []string{"a.txt", "locked", "z.txt"}; !slices.Equal(paths, want) {
		t.Fatalf("got %v, want %v", paths, want)
	}
}

// TestFileEntryFields pins 住 FileEntry 各字段的填充语义：
// Name 是条目自身名，AbsPath/RelPath 是逻辑路径，Depth 是相对根的层级（根下条目为 0）。
func TestFileEntryFields(t *testing.T) {
	root := absDir(t, "testdata/basic")

	var file, sub FileEntry
	for entry, err := range Find(root, OptionFiles()) {
		if err != nil {
			t.Fatal(err)
		}
		if entry.Name() == "a.txt" {
			file = entry
		}
	}
	for entry, err := range Find(root, OptionDirectories()) {
		if err != nil {
			t.Fatal(err)
		}
		if entry.Name() == "sub" {
			sub = entry
		}
	}

	if file == nil || file.AbsPath() != filepath.Join(root, "a.txt") || file.RelPath() != "a.txt" || file.IsDir() || file.Depth() != 0 {
		t.Fatalf("file entry filled wrong: %v", file)
	}
	if sub == nil || sub.AbsPath() != filepath.Join(root, "dir", "sub") || sub.RelPath() != "dir/sub" || !sub.IsDir() || sub.Depth() != 1 {
		t.Fatalf("dir entry filled wrong: %v", sub)
	}
}

// TestDepthSemantics pins 住 depth 语义：
// depth 为相对根目录的最大层级，根目录下的条目深度为 0，每进入一层子目录深度加 1。
func TestDepthSemantics(t *testing.T) {
	tests := []struct {
		depth int
		want  []string
	}{
		{0, []string{"a.txt", "dir"}},
		{1, []string{"a.txt", "dir", "dir/b.txt", "dir/sub"}},
		{2, []string{"a.txt", "dir", "dir/b.txt", "dir/sub", "dir/sub/c.txt"}},
	}
	for _, tt := range tests {
		t.Run(string(rune('0'+tt.depth)), func(t *testing.T) {
			paths, err := collectPaths(t,
				absDir(t, "testdata/basic"),
				[]Option{OptionDepth(tt.depth)},
			)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(paths, tt.want) {
				t.Fatalf("depth %d: got %v, want %v", tt.depth, paths, tt.want)
			}
		})
	}
}

func TestFindEarlyBreak(t *testing.T) {
	var count int
	for _, err := range Find(absDir(t, "testdata/basic")) {
		if err != nil {
			t.Fatal(err)
		}
		count++
		break
	}
	if count != 1 {
		t.Fatalf("got %d, want 1", count)
	}
}

// recordingMatcher 是测试用 IgnoreMatcher：按 ignored 判定，并记录每次被询问的入参
// （路径，目录带尾随 "/"，与 git check-ignore 的展示习惯一致）。
type recordingMatcher struct {
	ignored func(path string, isDir bool) bool
	calls   []string
}

func (m *recordingMatcher) Match(path string, isDir bool) bool {
	call := path
	if isDir {
		call += "/"
	}
	m.calls = append(m.calls, call)
	return m.ignored(path, isDir)
}

// TestOptionIgnore pins 住忽略匹配器的接入语义：匹配器收到相对遍历根的 "/" 分隔路径
// 与条目的 IsDir；命中的条目被跳过；目录命中时整棵子树剪掉且不再就其子孙询问匹配器；
// 与 OptionFiles 组合时剪枝同样生效。
func TestOptionIgnore(t *testing.T) {
	root := absDir(t, "testdata/basic")

	t.Run("matcher receives RelPath and IsDir", func(t *testing.T) {
		matcher := &recordingMatcher{ignored: func(string, bool) bool { return false }}
		paths, firstErr := collectPaths(t, root, []Option{OptionIgnore(matcher)})
		if firstErr != nil {
			t.Fatal(firstErr)
		}
		if want := []string{"a.txt", "dir", "dir/b.txt", "dir/sub", "dir/sub/c.txt"}; !slices.Equal(paths, want) {
			t.Fatalf("got %v, want %v", paths, want)
		}
		if want := []string{"a.txt", "dir/", "dir/b.txt", "dir/sub/", "dir/sub/c.txt"}; !slices.Equal(matcher.calls, want) {
			t.Fatalf("calls: got %v, want %v", matcher.calls, want)
		}
	})

	t.Run("ignored file is skipped", func(t *testing.T) {
		matcher := &recordingMatcher{ignored: func(path string, _ bool) bool { return path == "dir/b.txt" }}
		paths, firstErr := collectPaths(t, root, []Option{OptionIgnore(matcher)})
		if firstErr != nil {
			t.Fatal(firstErr)
		}
		if want := []string{"a.txt", "dir", "dir/sub", "dir/sub/c.txt"}; !slices.Equal(paths, want) {
			t.Fatalf("got %v, want %v", paths, want)
		}
	})

	t.Run("ignored dir prunes subtree", func(t *testing.T) {
		// pins：目录命中后其子孙不再产出，也不再询问匹配器（calls 中不出现 dir/sub 之下）。
		matcher := &recordingMatcher{ignored: func(path string, isDir bool) bool { return path == "dir/sub" && isDir }}
		paths, firstErr := collectPaths(t, root, []Option{OptionIgnore(matcher)})
		if firstErr != nil {
			t.Fatal(firstErr)
		}
		if want := []string{"a.txt", "dir", "dir/b.txt"}; !slices.Equal(paths, want) {
			t.Fatalf("got %v, want %v", paths, want)
		}
		if want := []string{"a.txt", "dir/", "dir/b.txt", "dir/sub/"}; !slices.Equal(matcher.calls, want) {
			t.Fatalf("calls: got %v, want %v", matcher.calls, want)
		}
	})

	t.Run("prunes even when only files are yielded", func(t *testing.T) {
		matcher := &recordingMatcher{ignored: func(path string, isDir bool) bool { return path == "dir" && isDir }}
		paths, firstErr := collectPaths(t, root, []Option{OptionFiles(), OptionIgnore(matcher)})
		if firstErr != nil {
			t.Fatal(firstErr)
		}
		if want := []string{"a.txt"}; !slices.Equal(paths, want) {
			t.Fatalf("got %v, want %v", paths, want)
		}
	})

	t.Run("multiple matchers any match ignores", func(t *testing.T) {
		fileMatcher := &recordingMatcher{ignored: func(path string, _ bool) bool { return path == "a.txt" }}
		dirMatcher := &recordingMatcher{ignored: func(path string, isDir bool) bool { return path == "dir/sub" && isDir }}
		paths, firstErr := collectPaths(t, root, []Option{OptionIgnore(fileMatcher), OptionIgnore(dirMatcher)})
		if firstErr != nil {
			t.Fatal(firstErr)
		}
		if want := []string{"dir", "dir/b.txt"}; !slices.Equal(paths, want) {
			t.Fatalf("got %v, want %v", paths, want)
		}
	})
}

// TestFollowLinks pins 住符号链接行为：
// 跟随时目录链接下钻且路径按链接路径（逻辑路径）产出、文件链接按目标类型产出、
// 断链按链接自身产出、成环不无限递归、指向根外的链接 RelPath 不出现 ".."；
// 默认不跟随（链接不下钻）。
func TestFollowLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows 上创建符号链接需要特权")
	}

	newFixture := func(t *testing.T) string {
		t.Helper()
		tmp := t.TempDir()
		writeFile(t, tmp, "real/a.txt", "")
		if err := os.Symlink(filepath.Join(tmp, "real"), filepath.Join(tmp, "dirlink")); err != nil {
			t.Fatal(err)
		}
		// 链接名与目标名不同，用于 pin 住 Name 按链接名而非目标名产出。
		if err := os.Symlink(filepath.Join(tmp, "real"), filepath.Join(tmp, "alias")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(tmp, "real", "a.txt"), filepath.Join(tmp, "filelink")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(tmp, "nowhere"), filepath.Join(tmp, "brokenlink")); err != nil {
			t.Fatal(err)
		}
		return tmp
	}

	t.Run("default does not follow", func(t *testing.T) {
		tmp := newFixture(t)
		paths, err := collectPaths(t, tmp, nil)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"alias", "brokenlink", "dirlink", "filelink", "real", "real/a.txt"}
		if !slices.Equal(paths, want) {
			t.Fatalf("got %v, want %v", paths, want)
		}
	})

	t.Run("follow links", func(t *testing.T) {
		tmp := newFixture(t)
		var paths []string
		var dirlink FileEntry
		for entry, err := range Find(tmp, OptionFollowLinks()) {
			if err != nil {
				t.Fatal(err)
			}
			paths = append(paths, entry.RelPath())
			if entry.Name() == "dirlink" {
				dirlink = entry
			}
		}

		// 目录链接下钻；filelink 按目标（文件）产出；brokenlink 解析失败按链接自身产出；
		// alias/dirlink 并行到达同一目标目录，各自产出（不判重）。
		want := []string{
			"alias", "alias/a.txt",
			"brokenlink",
			"dirlink", "dirlink/a.txt",
			"filelink",
			"real", "real/a.txt",
		}
		if !slices.Equal(paths, want) {
			t.Fatalf("got %v, want %v", paths, want)
		}

		// 跟随只影响分类与下钻：IsDir 按目标，Name/AbsPath 按链接自身路径。
		if dirlink == nil || !dirlink.IsDir() {
			t.Fatalf("dirlink 应按目标类型视为目录: %v", dirlink)
		}
		if want := filepath.Join(tmp, "dirlink"); dirlink.AbsPath() != want {
			t.Fatalf("dirlink AbsPath got %q, want %q", dirlink.AbsPath(), want)
		}
	})

	t.Run("link outside root keeps relPath namespace", func(t *testing.T) {
		parent := t.TempDir()
		root := filepath.Join(parent, "root")
		writeFile(t, root, "in.txt", "")
		external := filepath.Join(parent, "external")
		writeFile(t, external, "out.txt", "")
		if err := os.Symlink(external, filepath.Join(root, "extlink")); err != nil {
			t.Fatal(err)
		}

		paths, firstErr := collectPaths(t, root, []Option{OptionFollowLinks()})
		if firstErr != nil {
			t.Fatal(firstErr)
		}
		// 链接指向根外也按链接路径产出，RelPath 不会出现 ".."。
		want := []string{"extlink", "extlink/out.txt", "in.txt"}
		if !slices.Equal(paths, want) {
			t.Fatalf("got %v, want %v", paths, want)
		}
	})

	t.Run("cycle does not recurse forever", func(t *testing.T) {
		tmp := t.TempDir()
		writeFile(t, tmp, "a/x.txt", "")
		if err := os.Symlink(tmp, filepath.Join(tmp, "a", "up")); err != nil {
			t.Fatal(err)
		}

		paths, err := collectPaths(t, tmp, []Option{OptionFollowLinks()})
		if err != nil {
			t.Fatal(err)
		}
		// a/up 指向遍历根自身，判重后不再进入。
		want := []string{"a", "a/up", "a/x.txt"}
		if !slices.Equal(paths, want) {
			t.Fatalf("got %v, want %v", paths, want)
		}
	})
}
