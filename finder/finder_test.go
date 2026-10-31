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

func collectPaths(t *testing.T, dirs []string, opts []Option) ([]string, error) {
	t.Helper()
	var paths []string
	var firstErr error
	for entry, err := range Find(dirs, opts...) {
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
		dirs    []string
		opts    []Option
		want    []string
		wantErr bool
	}{
		{
			name:    "default includes files and directories",
			dirs:    []string{absDir(t, "testdata/basic")},
			want:    []string{"a.txt", "dir", "dir/b.txt", "dir/sub", "dir/sub/c.txt"},
			wantErr: false,
		},
		{
			name:    "files",
			dirs:    []string{absDir(t, "testdata/basic")},
			opts:    []Option{OptionFiles()},
			want:    []string{"a.txt", "dir/b.txt", "dir/sub/c.txt"},
			wantErr: false,
		},
		{
			name:    "directories",
			dirs:    []string{absDir(t, "testdata/basic")},
			opts:    []Option{OptionDirectories()},
			want:    []string{"dir", "dir/sub"},
			wantErr: false,
		},
		{
			name: "filter",
			dirs: []string{absDir(t, "testdata/basic")},
			opts: []Option{
				OptionFilter(func(entry FileEntry) bool { return !strings.HasPrefix(entry.RelPath(), "dir") }),
			},
			want:    []string{"a.txt"},
			wantErr: false,
		},
		{
			// pins：目录被 filter 拒绝时整棵子树剪掉。
			// filter 只拒绝名为 dir 的目录本身，其子条目（dir/b.txt 等并不匹配该条件）
			// 也不得出现在结果中。
			name: "filtered dir prunes subtree",
			dirs: []string{absDir(t, "testdata/basic")},
			opts: []Option{
				OptionFilter(func(entry FileEntry) bool {
					return !entry.IsDir() || entry.Name() != "dir"
				}),
			},
			want:    []string{"a.txt"},
			wantErr: false,
		},
		{
			// 多个 filter 串联执行，任一不满足即跳过（AND 语义）。
			name: "multiple filters",
			dirs: []string{absDir(t, "testdata/basic")},
			opts: []Option{
				OptionFilter(func(entry FileEntry) bool { return strings.HasSuffix(entry.Name(), ".txt") }),
				OptionFilter(func(entry FileEntry) bool { return strings.HasPrefix(entry.Name(), "a") }),
				OptionFilter(func(entry FileEntry) bool { return !entry.IsDir() }),
			},
			want:    []string{"a.txt"},
			wantErr: false,
		},
		{
			name:    "nonexistent dir yields error",
			dirs:    []string{absDir(t, "testdata/nonexistent")},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "error does not stop following dirs",
			dirs:    []string{absDir(t, "testdata/nonexistent"), absDir(t, "testdata/basic")},
			want:    []string{"a.txt", "dir", "dir/b.txt", "dir/sub", "dir/sub/c.txt"},
			wantErr: true,
		},
		{
			name:    "relative dir yields error",
			dirs:    []string{"testdata/basic"},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths, firstErr := collectPaths(t, tt.dirs, tt.opts)

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
		paths, firstErr := collectPaths(t, []string{tmp}, nil)
		if firstErr != nil {
			t.Fatal(firstErr)
		}
		if want := []string{"a.txt"}; !slices.Equal(paths, want) {
			t.Fatalf("got %v, want %v", paths, want)
		}
	})

	t.Run("OptionIgnoreVCS(false) keeps vcs dirs", func(t *testing.T) {
		tmp := newFixture(t)
		paths, firstErr := collectPaths(t, []string{tmp}, []Option{OptionIgnoreVCS(false)})
		if firstErr != nil {
			t.Fatal(firstErr)
		}
		want := []string{".git", ".git/config", ".hg", ".hg/config", ".svn", ".svn/config", "a.txt"}
		if !slices.Equal(paths, want) {
			t.Fatalf("got %v, want %v", paths, want)
		}
	})
}

// TestFileEntryFields pins 住 FileEntry 字段的填充，
// 以及组合的 fs.DirEntry 方法（Name/IsDir/Info）可直接使用。
func TestFileEntryFields(t *testing.T) {
	var file, dirEntry FileEntry
	for entry, err := range Find([]string{absDir(t, "testdata/basic")}, OptionFiles()) {
		if err != nil {
			t.Fatal(err)
		}
		if entry.Name() == "a.txt" {
			file = entry
		}
	}
	for entry, err := range Find([]string{absDir(t, "testdata/basic")}, OptionDirectories()) {
		if err != nil {
			t.Fatal(err)
		}
		if entry.Name() == "dir" {
			dirEntry = entry
		}
	}

	if file.AbsPath() == "" || file.RelPath() != "a.txt" || file.IsDir() {
		t.Fatalf("file entry filled wrong: %v", file)
	}
	if dirEntry.AbsPath() == "" || dirEntry.RelPath() != "dir" || !dirEntry.IsDir() {
		t.Fatalf("dir entry filled wrong: %v", dirEntry)
	}

	info, err := file.Info()
	if err != nil {
		t.Fatal(err)
	}
	if info.Name() != "a.txt" {
		t.Fatalf("got info name %q, want a.txt", info.Name())
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
				[]string{absDir(t, "testdata/basic")},
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
	for _, err := range Find([]string{absDir(t, "testdata/basic")}) {
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

func TestFindMultipleDirs(t *testing.T) {
	dirs := []string{
		absDir(t, "testdata/basic/dir/sub"),
		absDir(t, "testdata/basic/dir"),
	}
	var paths []string
	for entry, err := range Find(dirs) {
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, entry.RelPath())
	}
	if !slices.Equal(paths, []string{"c.txt", "b.txt", "sub", "sub/c.txt"}) {
		t.Fatalf("got %v", paths)
	}
}

// TestFollowLinks pins 住符号链接跟随行为：
// 跟随时目录链接下钻、文件链接按目标类型产出、断链按链接自身产出、成环不无限递归；
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
		paths, err := collectPaths(t, []string{tmp}, nil)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"brokenlink", "dirlink", "filelink", "real", "real/a.txt"}
		if !slices.Equal(paths, want) {
			t.Fatalf("got %v, want %v", paths, want)
		}
	})

	t.Run("follow links", func(t *testing.T) {
		tmp := newFixture(t)
		paths, err := collectPaths(t, []string{tmp}, []Option{OptionFollowLinks()})
		if err != nil {
			t.Fatal(err)
		}
		// dirlink 按目录下钻；filelink 按目标（文件）产出；brokenlink 解析失败按链接自身产出。
		want := []string{"brokenlink", "dirlink", "dirlink/a.txt", "filelink", "real", "real/a.txt"}
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

		paths, err := collectPaths(t, []string{tmp}, []Option{OptionFollowLinks()})
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
