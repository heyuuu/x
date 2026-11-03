package gitignore

import "testing"

// TestMatchSegment pin 住段级 fnmatch 语义：*、?、字符类（! 与 ^ 取反、
// 首 ] 字面、a-z 范围、尾 - 字面、类内 \ 转义）、\ 转义、未闭合类与尾部 \
// 永不匹配。字符类语义与 git check-ignore 实测对齐（[!b] 取反、[]] 匹配 ]）。
func TestMatchSegment(t *testing.T) {
	tests := []struct {
		pattern string
		name    string
		want    bool
	}{
		{"*.log", "debug.log", true},
		{"*.log", "main.go", false},
		{"main.go", "main.go", true},
		{"main.go", "main.go.txt", false},
		{"*", "", true},
		{"a*c", "ac", true},
		{"a?c", "abc", true},
		{"a?c", "ac", false},
		{"a?c", "a中c", true},
		{"a**b", "axxb", true},
		{"[!b]at", "cat", true},
		{"[!b]at", "bat", false},
		{"[^b]at", "cat", true},
		{"[^b]at", "bat", false},
		{"[]]", "]", true},
		{"[]]", "x", false},
		{"[!]]", "x", true},
		{"[!]]", "]", false},
		{"[a-]", "a", true},
		{"[a-]", "-", true},
		{"[a-]", "z", false},
		{"[a-z0-9_]", "m", true},
		{"[a-z]", "M", false},
		{"[!a-z]", "M", true},
		{`[\]]`, "]", true},
		{`\#x`, "#x", true},
		{`\*x`, "*x", true},
		{`\*x`, "ax", false},
		{`x\`, "x", false},
		{"foo[", "foo[", false},
		{"[]", "", false},
	}
	for _, tt := range tests {
		if got := matchSegment(tt.pattern, tt.name); got != tt.want {
			t.Errorf("matchSegment(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
		}
	}
}
