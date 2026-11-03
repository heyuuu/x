package gitignore

import (
	"slices"
	"testing"
)

// TestParsePattern 以白盒方式 pin 住单行解析产物的结构：
// 否定前缀、dirOnly、锚定（isGlob）与分段结果，语法依据 gitignore(5)（RULES.md）。
func TestParsePattern(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		inclusion bool
		dirOnly   bool
		isGlob    bool
		segments  []string
	}{
		{name: "plain name", line: "*.log", segments: []string{"*.log"}},
		{name: "trailing slash marks dir only", line: "build/", dirOnly: true, segments: []string{"build"}},
		{name: "slash anywhere anchors pattern", line: "a/b", isGlob: true, segments: []string{"a", "b"}},
		{name: "leading slash anchors and drops empty segment", line: "/foo", isGlob: true, segments: []string{"foo"}},
		{name: "inner empty segments dropped", line: "a//b", isGlob: true, segments: []string{"a", "b"}},
		{name: "double star segment", line: "a/**/b", isGlob: true, segments: []string{"a", "**", "b"}},
		{name: "consecutive stars without slash stay plain", line: "***", segments: []string{"***"}},
		{name: "negation prefix stripped into flag", line: "!keep.log", inclusion: true, segments: []string{"keep.log"}},
		{name: "negation only leaves empty pattern", line: "!", inclusion: true, segments: []string{""}},
		{name: "slash only leaves empty pattern", line: "/", dirOnly: true, segments: []string{""}},
		{name: "double slash only leaves no segments", line: "//", dirOnly: true, isGlob: true},
		{name: "unescaped trailing spaces trimmed", line: "foo  ", segments: []string{"foo"}},
		{name: "escaped trailing space kept", line: `keep\ `, segments: []string{`keep\ `}},
		{name: "escaped hash kept", line: `\#foo`, segments: []string{`\#foo`}},
		{name: "trailing backslash kept", line: `foo\`, segments: []string{`foo\`}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePattern(tt.line, "")
			if got.inclusion != tt.inclusion {
				t.Errorf("inclusion = %v, want %v", got.inclusion, tt.inclusion)
			}
			if got.dirOnly != tt.dirOnly {
				t.Errorf("dirOnly = %v, want %v", got.dirOnly, tt.dirOnly)
			}
			if got.isGlob != tt.isGlob {
				t.Errorf("isGlob = %v, want %v", got.isGlob, tt.isGlob)
			}
			if !slices.Equal(got.segments, tt.segments) {
				t.Errorf("segments = %q, want %q", got.segments, tt.segments)
			}
		})
	}

	// domain 在解析入口一并规范化为规范路径；越根的 domain 原样保留（永不命中）。
	t.Run("domain is normalized", func(t *testing.T) {
		if got := parsePattern("foo", "a/").domain; got != "a" {
			t.Errorf("domain = %q, want %q", got, "a")
		}
		if got := parsePattern("foo", "../x").domain; got != "../x" {
			t.Errorf("越根 domain 应原样保留，got %q", got)
		}
	})
}

// TestParsePatterns pin 住整份文档的行级处理：
// 注释与空行跳过、CRLF 兼容、\# 行与转义行尾空格保留。
func TestParsePatterns(t *testing.T) {
	doc := "# comment\r\n\n*.log\r\nbuild/\r\nkeep\\ \r\n\\#root\r\n"
	patterns := parsePatterns(doc, "")

	if len(patterns) != 4 {
		t.Fatalf("got %d patterns, want 4", len(patterns))
	}
	want := [][]string{
		{"*.log"},
		{"build"},
		{`keep\ `},
		{`\#root`},
	}
	for i, wantSegments := range want {
		if !slices.Equal(patterns[i].segments, wantSegments) {
			t.Errorf("patterns[%d].segments = %q, want %q", i, patterns[i].segments, wantSegments)
		}
	}
}
