package gitignore

import "testing"

// TestNormalizePath pin 住路径规范化："/" 分隔、根为空串、前导 "/" 视为相对根、
// 根内的 ".." 文本消解、越过根的路径（ok=false）不进入判定。
func TestNormalizePath(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"", "", true},
		{".", "", true},
		{"./", "", true},
		{"/", "", true},
		{"//", "", true},
		{"a", "a", true},
		{"a/", "a", true},
		{"/a", "a", true},
		{"./a", "a", true},
		{"a//b", "a/b", true},
		{"a/./b", "a/b", true},
		{"a/./b/", "a/b", true},
		{"a/../b", "b", true},
		{"a/..", "", true},
		{"..", "", false},
		{"../a", "", false},
		{"../../a/b", "", false},
		{"a/../../x", "", false},
	}
	for _, tt := range tests {
		got, ok := normalizePath(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("normalizePath(%q) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}
