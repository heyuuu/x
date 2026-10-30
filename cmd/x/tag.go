package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func newTagCmd() *cobra.Command {
	var minor, force bool
	cmd := &cobra.Command{
		Use:   "tag <name>",
		Short: "给指定包打下一版本的 tag（默认递增 patch，--minor 递增 minor）",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return tagPackages([]string{args[0]}, minor, force)
		},
	}
	cmd.Flags().BoolVar(&minor, "minor", false, "递增 minor 并将 patch 清零")
	cmd.Flags().BoolVar(&force, "force", false, "跳过包内变更检查，强制打新 tag")
	return cmd
}

func newTagAllCmd() *cobra.Command {
	var minor, force bool
	cmd := &cobra.Command{
		Use:   "tag-all",
		Short: "给所有包打下一版本的 tag（默认递增 patch，--minor 递增 minor）",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return tagPackages(nil, minor, force)
		},
	}
	cmd.Flags().BoolVar(&minor, "minor", false, "递增 minor 并将 patch 清零")
	cmd.Flags().BoolVar(&force, "force", false, "跳过包内变更检查，强制打新 tag")
	return cmd
}

// tagPackages 给 targets 中的每个包打下一版本的 tag；targets 为空时作用于所有包。
// 每个 target 必须是已知的包；包目录下的工作区不干净时会拒绝打 tag，
// 以保证 tag 始终对应已提交的状态。包已有版本时，除非 force 为 true，
// 只有最新 tag 之后包内文件有变更才会打新 tag。
func tagPackages(targets []string, minor, force bool) error {
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
		if err := tagPackage(root, name, minor, force); err != nil {
			return err
		}
	}
	return nil
}

func tagPackage(root, name string, minor, force bool) error {
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
	latest, latestTag, hasLatest := latestVersion(tags)
	next := nextVersion(latest, hasLatest, minor)
	if !force && hasLatest {
		changed, err := gitOut(root, "log", "--oneline", latestTag+"..HEAD", "--", name)
		if err != nil {
			return err
		}
		if changed == "" {
			fmt.Printf("%s: 没有变化，无需新 tag\n", name)
			return nil
		}
	}

	tag := fmt.Sprintf("%s/v%s", name, next)
	if _, err := gitOut(root, "tag", tag); err != nil {
		return fmt.Errorf("%s：打 tag 失败：%w", name, err)
	}
	fmt.Printf("%s: tagged %s\n", name, tag)
	return nil
}

// latestVersion 从 git tag 输出中找出版本号最大的 tag：
// 返回其版本号和完整 tag 行（形如 "name/v0.1.0"）。
// 格式不符合的 tag 行会被忽略；没有任何合法版本时 ok 为 false。
func latestVersion(tags string) (version Version, tag string, ok bool) {
	for _, line := range strings.Split(tags, "\n") {
		line = strings.TrimSpace(line)
		if i := strings.Index(line, "/v"); i >= 0 {
			if v, valid := parseVersion(line[i+2:]); valid {
				if !ok || v.Compare(version) > 0 {
					version, tag, ok = v, line, true
				}
			}
		}
	}
	return version, tag, ok
}

// nextVersion 根据最新版本计算下一版本：
// 没有任何版本时（ok 为 false）使用初始版本，否则递增最新版本。
func nextVersion(latest Version, ok bool, minor bool) Version {
	if !ok {
		return initialVersion
	}
	return latest.Next(minor)
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
