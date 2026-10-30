package main

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		s      string
		want   Version
		wantOK bool
	}{
		{s: "0.1.0", want: Version{0, 1, 0}, wantOK: true},
		{s: "10.20.30", want: Version{10, 20, 30}, wantOK: true},
		{s: "2.0.0", want: Version{2, 0, 0}, wantOK: true},
		{s: "1.0", wantOK: false},
		{s: "1.0.0.0", wantOK: false},
		{s: "1.0.x", wantOK: false},
		{s: "v1.0.0", wantOK: false}, // 不接受 v 前缀
		{s: "", wantOK: false},
		{s: "1.0.-1", wantOK: false}, // 负数
	}
	for _, tt := range tests {
		got, ok := parseVersion(tt.s)
		if ok != tt.wantOK {
			t.Errorf("parseVersion(%q) ok = %v, want %v", tt.s, ok, tt.wantOK)
			continue
		}
		if ok && got != tt.want {
			t.Errorf("parseVersion(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestVersionString(t *testing.T) {
	if got := (Version{0, 10, 2}).String(); got != "0.10.2" {
		t.Errorf("Version{0,10,2}.String() = %q, want %q", got, "0.10.2")
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		a, b Version
		want int // 1 表示 a>b，-1 表示 a<b，0 表示相等
	}{
		{a: Version{0, 1, 0}, b: Version{0, 1, 0}, want: 0},
		{a: Version{0, 1, 1}, b: Version{0, 1, 0}, want: 1},
		{a: Version{0, 1, 0}, b: Version{0, 2, 0}, want: -1},
		{a: Version{0, 10, 0}, b: Version{0, 9, 9}, want: 1}, // 两位数不会字典序排错
		{a: Version{2, 0, 0}, b: Version{1, 99, 99}, want: 1},
	}
	for _, tt := range tests {
		if got := sign(tt.a.Compare(tt.b)); got != tt.want {
			t.Errorf("%v.Compare(%v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestVersionNext(t *testing.T) {
	tests := []struct {
		name  string
		v     Version
		minor bool
		want  Version
	}{
		{name: "patch 加一", v: Version{0, 2, 5}, want: Version{0, 2, 6}},
		{name: "minor 递增并将 patch 清零", v: Version{0, 2, 5}, minor: true, want: Version{0, 3, 0}},
		{name: "minor 递增不改变 major", v: Version{2, 3, 7}, minor: true, want: Version{2, 4, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.Next(tt.minor); got != tt.want {
				t.Errorf("%v.Next(minor=%v) = %v, want %v", tt.v, tt.minor, got, tt.want)
			}
		})
	}
}
