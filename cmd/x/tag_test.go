package main

import "testing"

func TestNextVersion(t *testing.T) {
	tests := []struct {
		name   string
		latest Version
		ok     bool
		minor  bool
		want   Version
	}{
		{name: "无历史版本，使用初始版本", ok: false, want: initialVersion},
		{name: "无历史版本时 --minor 也使用初始版本", ok: false, minor: true, want: initialVersion},
		{name: "单个版本，patch 加一", ok: true, latest: Version{0, 1, 0}, want: Version{0, 1, 1}},
		{name: "两位数 patch 递增正确", ok: true, latest: Version{0, 10, 2}, want: Version{0, 10, 3}},
		{name: "major 递增后的版本仍正确", ok: true, latest: Version{2, 0, 0}, want: Version{2, 0, 1}},
		{name: "minor 递增并将 patch 清零", ok: true, latest: Version{0, 2, 5}, minor: true, want: Version{0, 3, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextVersion(tt.latest, tt.ok, tt.minor); got != tt.want {
				t.Errorf("nextVersion(%v, ok=%v, minor=%v) = %v, want %v", tt.latest, tt.ok, tt.minor, got, tt.want)
			}
		})
	}
}

func TestLatestVersion(t *testing.T) {
	tests := []struct {
		name     string
		tags     string
		wantVer  Version
		wantTag  string
		wantBool bool
	}{
		{name: "空输出", tags: ""},
		{name: "无合法版本", tags: "di/v2\ndi/v0.1\n"},
		{name: "忽略空行", tags: "\n\ndi/v1.0.0\n", wantVer: Version{1, 0, 0}, wantTag: "di/v1.0.0", wantBool: true},
		{name: "忽略旧格式 major tag", tags: "di/v2\ndi/v0.1.0\n", wantVer: Version{0, 1, 0}, wantTag: "di/v0.1.0", wantBool: true},
		{name: "忽略两段式版本", tags: "di/v0.1\ndi/v1.0.0\n", wantVer: Version{1, 0, 0}, wantTag: "di/v1.0.0", wantBool: true},
		{name: "多个版本取最大", tags: "di/v0.10.0\ndi/v0.9.0\ndi/v0.2.3\n", wantVer: Version{0, 10, 0}, wantTag: "di/v0.10.0", wantBool: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver, tag, ok := latestVersion(tt.tags)
			if ver != tt.wantVer || tag != tt.wantTag || ok != tt.wantBool {
				t.Errorf("latestVersion(%q) = (%v, %q, %v), want (%v, %q, %v)",
					tt.tags, ver, tag, ok, tt.wantVer, tt.wantTag, tt.wantBool)
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
