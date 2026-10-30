package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Version 表示三段式版本号 X.Y.Z，对应 tag 名 {name}/v{X.Y.Z} 中的版本部分。
type Version struct {
	Major, Minor, Patch int
}

// initialVersion 是没有历史 tag 时使用的初始版本。
var initialVersion = Version{Minor: 1}

// parseVersion 解析不带 v 前缀的三段式版本号（如 "0.1.0"），
// 格式不合法时第二个返回值为 false。
func parseVersion(s string) (Version, bool) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, false
	}
	var v Version
	nums := [3]*int{&v.Major, &v.Minor, &v.Patch}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return Version{}, false
		}
		*nums[i] = n
	}
	return v, true
}

// String 返回不带 v 前缀的版本号字符串。
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare 按数值逐段比较：大于返回正数，小于返回负数，相等返回 0。
func (v Version) Compare(o Version) int {
	for _, d := range [3]int{v.Major - o.Major, v.Minor - o.Minor, v.Patch - o.Patch} {
		if d != 0 {
			return d
		}
	}
	return 0
}

// Next 返回下一版本：minor 为 true 时递增 minor 并将 patch 清零，
// 否则仅递增 patch。
func (v Version) Next(minor bool) Version {
	if minor {
		return Version{Major: v.Major, Minor: v.Minor + 1, Patch: 0}
	}
	return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
}
