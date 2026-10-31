package finder

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

// 本文件的测试针对代码评审发现的问题 pin 住“正确行为”：
// 在对应缺陷修复前这些测试是失败的，修复后转绿，作为回归测试保留。
// 场景全部用 t.TempDir() 动态构造，不依赖 testdata。

// skipIfNoFilePerms 跳过无法用 chmod 000 制造“读取失败”的环境（Windows、root 用户）。
func skipIfNoFilePerms(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("windows 上 chmod 000 无法拒绝属主读取")
	}
	if os.Geteuid() == 0 {
		t.Skip("root 用户不受 chmod 000 限制")
	}
}

// writeFile 在 tmp 下写入相对路径 rel 的文件，自动创建父目录。
func writeFile(t *testing.T, tmp, rel, content string) {
	t.Helper()
	path := filepath.Join(tmp, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestFindFilesystemRoot：以 "/" 作为遍历根不应产出错误。
// 现状：TrimSuffix 把 "/" 截成空串，之后 filepath.Rel("", path) 对每个条目都报
// "Rel: can't make ... relative to"，整个遍历失败（0 条目）。
func TestFindFilesystemRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(`windows 上 "/" 不是绝对路径，会走 ErrRelativePath 分支`)
	}

	// 只检查第一个产出即可暴露问题；不 break 会把整个文件系统走一遍。
	for _, err := range Find([]string{"/"}, OptionDepth(0)) {
		if err != nil {
			t.Fatalf("遍历文件系统根目录不应报错: %v", err)
		}
		break
	}
}

// TestFindSymlinkRootNotSilentlyEmpty：符号链接作为遍历根不应“既无条目也无错误”地空转。
// 现状：WalkDir 对根使用 Lstat，符号链接不被视为目录，只回调根自身一次又被跳过，
// 结果 0 条目 0 错误，调用方无法区分“目标是空目录”和“根本没遍历”。
// 跟随链接遍历目标、或显式产出错误，两种修复方式本测试都接受。
func TestFindSymlinkRootNotSilentlyEmpty(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows 上创建符号链接需要特权")
	}

	parent := t.TempDir()
	target := filepath.Join(parent, "data")
	writeFile(t, target, "a.txt", "x\n")
	link := filepath.Join(parent, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	paths, firstErr := collectPaths(t, []string{link}, nil)
	if len(paths) == 0 && firstErr == nil {
		t.Fatalf("符号链接作为根目录不应静默返回空（期望遍历目标 %s 或产出错误）", target)
	}
}

// TestDepthPrunesDeepSubtrees：超过 maxDepth 的子树不应被访问。
// 现状：超深条目只是不产出（return nil）而不是 SkipDir，WalkDir 仍会深入。
// 用一个 chmod 000 的深层目录暴露：其内容不可能被产出，
// 读取它时的 "permission denied" 就不该出现在结果里。
func TestDepthPrunesDeepSubtrees(t *testing.T) {
	skipIfNoFilePerms(t)

	tmp := t.TempDir()
	writeFile(t, tmp, "a.txt", "x\n")
	writeFile(t, tmp, "deep/locked/leaf.txt", "x\n") // locked 深度为 1，超过 maxDepth=0
	locked := filepath.Join(tmp, "deep", "locked")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	paths, firstErr := collectPaths(t, []string{tmp}, []Option{OptionDepth(0)})
	if want := []string{"a.txt", "deep"}; !slices.Equal(paths, want) {
		t.Fatalf("got %v, want %v", paths, want)
	}
	if firstErr != nil {
		t.Fatalf("超过 maxDepth 的子树不应被读取，也就不应产出错误: %v", firstErr)
	}
}
