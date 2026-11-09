package xfilepath

import "testing"

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		prefix string
		want   bool
	}{
		{"子目录", "/a/b", "/a", true},
		{"自身也算", "/a", "/a", true},
		// 与 strings.HasPrefix 的区别所在：按路径段边界比较
		{"兄弟目录不算段前缀", "/a/bb", "/a/b", false},
		{"相似名是子目录", "/a/bb", "/a", true},
		{"目录名以 .. 开头不误判为上层", "/a/..foo", "/a", true},
		{"prefix 为根目录", "/foo", "/", true},
		{"根自身", "/", "/", true},
		{"相对路径", "a/b", "a", true},
		{"相对路径逃逸", "a/b", "a/c", false},
		{"绝对/相对混合", "a/b", "/a", false},
		{"prefix 为空串", "/a", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasPrefix(tt.path, tt.prefix); got != tt.want {
				t.Errorf("HasPrefix(%q, %q) = %v, want %v", tt.path, tt.prefix, got, tt.want)
			}
		})
	}
}
