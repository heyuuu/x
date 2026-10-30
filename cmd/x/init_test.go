package main

import "testing"

func TestNameRE(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "di", want: true},
		{name: "di2", want: true},
		{name: "my_pkg", want: true},
		{name: "my_pkg_2", want: true},
		{name: "Foo", want: false},      // 大写
		{name: "1abc", want: false},     // 数字开头
		{name: "foo-bar", want: false},  // 连字符
		{name: "_foo", want: false},     // 下划线开头
		{name: "foo_", want: false},     // 下划线结尾
		{name: "foo__bar", want: false}, // 连续下划线
		{name: "", want: false},
	}
	for _, tt := range tests {
		if got := nameRE.MatchString(tt.name); got != tt.want {
			t.Errorf("nameRE.MatchString(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
