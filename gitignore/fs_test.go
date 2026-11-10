package gitignore

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// writeFile 在 root 下写入规范路径 rel 的文件，自动创建父目录。
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestLocalFS pin 住 LocalFS 的路径解析：相对 Root 读取，Root 为空时相对进程工作目录；
// 文件不存在时返回可用 errors.Is 判定的 fs.ErrNotExist。
func TestLocalFS(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "sub/.gitignore", "*.log\n")

	t.Run("resolves against root", func(t *testing.T) {
		content, err := LocalFS{Root: root}.OpenFile("sub/.gitignore")
		if err != nil {
			t.Fatal(err)
		}
		if content != "*.log\n" {
			t.Fatalf("got %q, want %q", content, "*.log\n")
		}
	})

	t.Run("missing file yields not exist", func(t *testing.T) {
		_, err := LocalFS{Root: root}.OpenFile(".gitignore")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("got err %v, want fs.ErrNotExist", err)
		}
	})

	t.Run("empty root resolves against working directory", func(t *testing.T) {
		t.Chdir(root)
		content, err := LocalFS{}.OpenFile("sub/.gitignore")
		if err != nil {
			t.Fatal(err)
		}
		if content != "*.log\n" {
			t.Fatalf("got %q, want %q", content, "*.log\n")
		}
	})
}

// TestAutoloadMatcherLocalFS pin 住 AutoloadMatcher 经 LocalFS 以 Root 为判定根
// 加载磁盘上各级目录的 .gitignore。
func TestAutoloadMatcherLocalFS(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".gitignore", "*.log\nbuild/\n")
	writeFile(t, root, "sub/.gitignore", "!keep.log\n")

	matcher := NewAutoloadMatcher(LocalFS{Root: root})

	if !matcher.Match("a.log", false) {
		t.Fatal("a.log 应被根 *.log 忽略")
	}
	if !matcher.Match("sub/x.log", false) {
		t.Fatal("sub/x.log 应被根 *.log 忽略")
	}
	if matcher.Match("sub/keep.log", false) {
		t.Fatal("sub/keep.log 应被 sub/.gitignore 的 !keep.log 重新包含")
	}
	if !matcher.Match("build/out/a.txt", false) {
		t.Fatal("build/out/a.txt 应随被排除的祖先目录 build 一并忽略")
	}
	if matcher.Match("sub/main.go", false) {
		t.Fatal("sub/main.go 不应被忽略")
	}
}
