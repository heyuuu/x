package main

import "testing"

func TestNextVersion(t *testing.T) {
	tests := []struct {
		name     string
		versions []Version
		minor    bool
		want     Version
	}{
		{name: "无历史版本，使用初始版本", versions: nil, want: initialVersion},
		{name: "无历史版本时 --minor 也使用初始版本", versions: nil, minor: true, want: initialVersion},
		{name: "单个版本，patch 加一", versions: []Version{{0, 1, 0}}, want: Version{0, 1, 1}},
		{name: "多个版本，取最新后 patch 加一", versions: []Version{{0, 1, 0}, {0, 2, 0}, {0, 1, 5}}, want: Version{0, 2, 1}},
		{name: "两位数 patch 不会排错", versions: []Version{{0, 9, 9}, {0, 10, 2}}, want: Version{0, 10, 3}},
		{name: "major 递增后的版本仍正确", versions: []Version{{1, 9, 9}, {2, 0, 0}}, want: Version{2, 0, 1}},
		{name: "minor 递增并将 patch 清零", versions: []Version{{0, 2, 5}}, minor: true, want: Version{0, 3, 0}},
		{name: "minor 递增取最新版本", versions: []Version{{0, 9, 9}, {0, 10, 2}}, minor: true, want: Version{0, 11, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextVersion(tt.versions, tt.minor); got != tt.want {
				t.Errorf("nextVersion(%v, minor=%v) = %v, want %v", tt.versions, tt.minor, got, tt.want)
			}
		})
	}
}

func TestParseVersions(t *testing.T) {
	tests := []struct {
		name string
		tags string
		want []Version
	}{
		{name: "空输出", tags: "", want: nil},
		{name: "解析标准 tag 行", tags: "di/v0.1.0\ndi/v0.2.3\n", want: []Version{{0, 1, 0}, {0, 2, 3}}},
		{name: "忽略旧格式 major tag", tags: "di/v2\ndi/v0.1.0\n", want: []Version{{0, 1, 0}}},
		{name: "忽略两段式版本", tags: "di/v0.1\ndi/v1.0.0\n", want: []Version{{1, 0, 0}}},
		{name: "忽略空行", tags: "\n\ndi/v1.0.0\n", want: []Version{{1, 0, 0}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseVersions(tt.tags)
			if len(got) != len(tt.want) {
				t.Fatalf("parseVersions(%q) = %v, want %v", tt.tags, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("parseVersions(%q) = %v, want %v", tt.tags, got, tt.want)
				}
			}
		})
	}
}

func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	default:
		return 0
	}
}
