// Command x 用于管理仓库中的各个包。
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "x:", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "x",
		Short: "管理 x 仓库中的各个包",
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
