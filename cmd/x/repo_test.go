package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackages(t *testing.T) {
	root := t.TempDir()
	mkdir := func(p ...string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(append([]string{root}, p...)...), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	touch := func(p ...string) {
		t.Helper()
		mkdir(p[:len(p)-1]...)
		f, err := os.Create(filepath.Join(append([]string{root}, p...)...))
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}

	// 布局：di 是包；foo 有 go.mod 是包；bar 没有 go.mod 不是包；
	// cmd 和 .hidden 即使有 go.mod 也应被排除。
	mkdir("bar")
	touch("di", "go.mod")
	touch("foo", "go.mod")
	touch("cmd", "x", "go.mod")
	touch(".hidden", "go.mod")

	got, err := packages(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"di", "foo"}
	if len(got) != len(want) {
		t.Fatalf("packages() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("packages() = %v, want %v", got, want)
		}
	}
}

func TestRootDir(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "di", "internal")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(nested)
	got, err := rootDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Errorf("rootDir() = %q, want %q", got, root)
	}
}

func TestRootDirNotFound(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := rootDir(); err == nil {
		t.Fatal("rootDir() 在没有 .git 的目录下应报错")
	}
}
