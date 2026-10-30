package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// repoModule 是仓库的 module 路径前缀。
const repoModule = "github.com/heyuuu/x"

// rootDir 返回仓库根目录：从当前工作目录向上查找，
// 直到找到 .git 目录为止。
func rootDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if fi, err := os.Stat(filepath.Join(dir, ".git")); err == nil && fi.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("从 %s 向上未找到仓库根目录（.git）", dir)
		}
		dir = parent
	}
}

// packages 返回所有包含 go.mod 的一级子目录名，即仓库中的全部 Go 包。
// 工具所在的 cmd 目录不参与枚举。
func packages(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || e.Name() == "cmd" {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, e.Name(), "go.mod")); err == nil {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
