package main

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"
)

func newVersionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "versions",
		Short: "按字典序列出所有包的最新版本（go.mod require 格式）",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return printVersions()
		},
	}
}

// printVersions 按包名字典序输出每个包的最新版本，格式为
// "module 路径 vX.Y.Z"；从未打过 tag 的包输出初始版本。
func printVersions() error {
	root, err := rootDir()
	if err != nil {
		return err
	}
	names, err := packages(root)
	if err != nil {
		return err
	}
	slices.Sort(names)
	for _, name := range names {
		tags, err := gitOut(root, "tag", "-l", name+"/v*")
		if err != nil {
			return err
		}
		v, _, ok := latestVersion(tags)
		if !ok {
			v = initialVersion
		}
		fmt.Printf("%s/%s v%s\n", repoModule, name, v)
	}
	return nil
}
