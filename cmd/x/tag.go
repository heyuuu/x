package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func newTagCmd() *cobra.Command {
	var minor bool
	cmd := &cobra.Command{
		Use:   "tag <name>",
		Short: "给指定包打下一版本的 tag（默认递增 patch，--minor 递增 minor）",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return tagPackages([]string{args[0]}, minor)
		},
	}
	cmd.Flags().BoolVar(&minor, "minor", false, "递增 minor 并将 patch 清零")
	return cmd
}

func newTagAllCmd() *cobra.Command {
	var minor bool
	cmd := &cobra.Command{
		Use:   "tag-all",
		Short: "给所有包打下一版本的 tag（默认递增 patch，--minor 递增 minor）",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return tagPackages(nil, minor)
		},
	}
	cmd.Flags().BoolVar(&minor, "minor", false, "递增 minor 并将 patch 清零")
	return cmd
}

// tagPackages 给 targets 中的每个包打下一版本的 tag；targets 为空时作用于所有包。
// 每个 target 必须是已知的包；包目录下的工作区不干净时会拒绝打 tag，
// 以保证 tag 始终对应已提交的状态。
func tagPackages(targets []string, minor bool) error {
	root, err := rootDir()
	if err != nil {
		return err
	}
	names, err := packages(root)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		targets = names
	}

	known := make(map[string]bool, len(names))
	for _, n := range names {
		known[n] = true
	}
	for _, name := range targets {
		if !known[name] {
			return fmt.Errorf("%s：不是包（未找到 go.mod）", name)
		}
	}
	for _, name := range targets {
		if err := tagPackage(root, name, minor); err != nil {
			return err
		}
	}
	return nil
}

func tagPackage(root, name string, minor bool) error {
	dirty, err := gitOut(root, "status", "--porcelain", "--", name)
	if err != nil {
		return err
	}
	if dirty != "" {
		return fmt.Errorf("%s：工作区不干净，请先提交", name)
	}

	tags, err := gitOut(root, "tag", "-l", name+"/v*")
	if err != nil {
		return err
	}
	next := nextVersion(parseVersions(tags), minor)

	tag := fmt.Sprintf("%s/v%s", name, next)
	if _, err := gitOut(root, "tag", tag); err != nil {
		return fmt.Errorf("%s：打 tag 失败：%w", name, err)
	}
	fmt.Printf("%s: tagged %s\n", name, tag)
	return nil
}

// nextVersion 根据已有版本列表计算下一版本：
// 没有任何版本时使用初始版本，否则取最新版本递增。
func nextVersion(versions []Version, minor bool) Version {
	if len(versions) == 0 {
		return initialVersion
	}
	latest := versions[0]
	for _, v := range versions[1:] {
		if v.Compare(latest) > 0 {
			latest = v
		}
	}
	return latest.Next(minor)
}

// parseVersions 从 git tag 输出中解析出形如 "name/v0.1.0" 的版本号列表。
// 格式不符合的 tag 行会被忽略。
func parseVersions(tags string) []Version {
	var versions []Version
	for _, line := range strings.Split(tags, "\n") {
		line = strings.TrimSpace(line)
		if i := strings.Index(line, "/v"); i >= 0 {
			if v, ok := parseVersion(line[i+2:]); ok {
				versions = append(versions, v)
			}
		}
	}
	return versions
}

func gitOut(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s：%w", strings.Join(args, " "), err)
	}
	return string(out), nil
}
