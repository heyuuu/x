// Command x 用于管理仓库中的各个包。
package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "x",
		Short: "管理 x 仓库中的各个包",
		// 错误不再附带 usage；需要时用 `x help <命令>` 查看
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// PersistentPreRunE 之前发生的输入类错误仍附 usage；运行期错误不再附 usage
			cmd.SilenceUsage = true
			return nil
		},
		// 不创建 completion 命令
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	cmd.AddCommand(
		newInitCmd(),
		newRunCmd(),
		newRunAllCmd(),
		newTagCmd(),
		newTagAllCmd(),
		newVersionsCmd(),
	)
	return cmd
}
