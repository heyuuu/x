package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"
)

// nameRE 限制包名只能是小写标识符，
// 使其既可作为目录名，也可作为 Go module 路径的末段。
var nameRE = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <name>",
		Short: "创建一个新包，并生成独立的 go.mod",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return initPackage(args[0])
		},
	}
}

func initPackage(name string) error {
	if !nameRE.MatchString(name) {
		return fmt.Errorf("无效的包名 %q：仅允许小写字母和数字，下划线只能出现在分段之间", name)
	}

	root, err := rootDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(root, name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("目录 %s 已存在", dir)
	}

	if err := os.Mkdir(dir, 0o755); err != nil {
		return err
	}

	c := exec.Command("go", "mod", "init", repoModule+"/"+name)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod init 执行失败：%v\n%s", err, out)
	}
	fmt.Printf("created package %s (%s)\n", name, repoModule+"/"+name)
	return nil
}
