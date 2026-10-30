package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <name> <command...>",
		Short: "在指定包目录下执行任意命令",
		Args:  cobra.MinimumNArgs(2),
		// 命令需要原样透传给子进程，禁用 flag 解析避免拦截命令自身的参数。
		// 查看帮助用 `x help run`。
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return runIn(args[0], args[1:])
		},
	}
	return cmd
}

func newRunAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run-all <command...>",
		Short: "在所有包目录下执行任意命令",
		Args:  cobra.MinimumNArgs(1),
		// 同 run，禁用 flag 解析以支持命令自身的参数。
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return runInAll(args)
		},
	}
	return cmd
}

func runIn(name string, command []string) error {
	root, err := rootDir()
	if err != nil {
		return err
	}
	names, err := packages(root)
	if err != nil {
		return err
	}
	for _, n := range names {
		if n == name {
			return execute(root+"/"+name, name, command)
		}
	}
	return fmt.Errorf("%s：不是包（未找到 go.mod）", name)
}

func runInAll(command []string) error {
	root, err := rootDir()
	if err != nil {
		return err
	}
	names, err := packages(root)
	if err != nil {
		return err
	}

	var failed []string
	for _, name := range names {
		if err := execute(root+"/"+name, name, command); err != nil {
			failed = append(failed, name)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("命令 %s 在以下包中执行失败：%s", strings.Join(command, " "), strings.Join(failed, ", "))
	}
	return nil
}

// execute 在 dir 下执行 command，name 仅用于输出标识。
func execute(dir, name string, command []string) error {
	fmt.Printf("=== %s: %s\n", name, strings.Join(command, " "))
	c := exec.Command(command[0], command[1:]...)
	c.Dir = dir
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}
