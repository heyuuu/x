package xos

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// makeFile 在 dir 下创建一个普通文件并返回其路径。
func makeFile(t *testing.T, dir string, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPathExists(t *testing.T) {
	dir := t.TempDir()
	file := makeFile(t, dir, "file.txt")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"存在的文件", file, true},
		{"存在的目录", dir, true},
		{"不存在的路径", filepath.Join(dir, "no-such"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PathExists(tt.path); got != tt.want {
				t.Errorf("PathExists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsFile(t *testing.T) {
	dir := t.TempDir()
	file := makeFile(t, dir, "file.txt")

	// 符号链接跟随目标（os.Stat 语义）
	linkToFile := filepath.Join(dir, "link-to-file")
	if err := os.Symlink(file, linkToFile); err != nil {
		t.Fatal(err)
	}
	linkToDir := filepath.Join(dir, "link-to-dir")
	if err := os.Symlink(dir, linkToDir); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"普通文件", file, true},
		{"指向文件的符号链接", linkToFile, true},
		{"目录", dir, false},
		{"指向目录的符号链接", linkToDir, false},
		{"不存在的路径", filepath.Join(dir, "no-such"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsFile(tt.path); got != tt.want {
				t.Errorf("IsFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}

	// 设备文件等特殊文件不是普通文件——这正是用 IsRegular 而非 !IsDir 的原因
	if runtime.GOOS != "windows" {
		if IsFile("/dev/null") {
			t.Error("IsFile(/dev/null) = true, want false（设备文件不是普通文件）")
		}
	}
}

func TestIsDir(t *testing.T) {
	dir := t.TempDir()
	file := makeFile(t, dir, "file.txt")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"目录", dir, true},
		{"文件", file, false},
		{"不存在的路径", filepath.Join(dir, "no-such"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDir(tt.path); got != tt.want {
				t.Errorf("IsDir(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("父目录已存在", func(t *testing.T) {
		path := filepath.Join(dir, "a.txt")
		if err := WriteFile(path, []byte("hello"), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "hello" {
			t.Errorf("内容 = %q, want %q", got, "hello")
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0600 {
			t.Errorf("文件权限 = %o, want 600", perm)
		}
	})

	t.Run("多级父目录自动创建", func(t *testing.T) {
		path := filepath.Join(dir, "x", "y", "z", "a.txt")
		if err := WriteFile(path, []byte("nested"), 0600); err != nil {
			t.Fatal(err)
		}
		if !IsFile(path) {
			t.Errorf("%s 未写入成功", path)
		}
		if !IsDir(filepath.Join(dir, "x", "y", "z")) {
			t.Error("父目录未创建")
		}
	})

	t.Run("覆盖已有文件", func(t *testing.T) {
		path := filepath.Join(dir, "b.txt")
		if err := WriteFile(path, []byte("old"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := WriteFile(path, []byte("new"), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "new" {
			t.Errorf("内容 = %q, want %q", got, "new")
		}
	})

	t.Run("父路径是已有文件时报错", func(t *testing.T) {
		file := makeFile(t, dir, "c.txt")
		if err := WriteFile(filepath.Join(file, "child.txt"), nil, 0600); err == nil {
			t.Error("期望报错，得到 nil")
		}
	})

	t.Run("无父目录组件的相对路径", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if err := WriteFile("plain.txt", []byte("rel"), 0600); err != nil {
			t.Fatal(err)
		}
		if !IsFile("plain.txt") {
			t.Error("plain.txt 未写入成功")
		}
	})
}
