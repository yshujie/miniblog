package miniblog

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewMiniBlogCommand 创建博客的 *cobra.Command 对象
// 之后可通过 Command 对象的 Execute 方法启动应用程序
func NewMiniBlogCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "miniblog",                                                                // 指定命令名字
		Short:        "A good Go pratical project",                                              // 命令的简述
		Long:         "A good Go pratical project, used to create user with basic information.", // 命令的详细描述
		SilenceUsage: true,                                                                      // 静默命令执行错误
		// cmd.Execute() 方法执行时，会调用 RunE 方法，执行 run() 方法
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
		// 命令运行时，不需要指定命令行参数
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if len(arg) > 0 {
					return fmt.Errorf("%q does not take any arguments, got %q", cmd.CommandPath(), arg)
				}
			}
			return nil
		},
	}
}

// run 函数，实际的业务代码入口
func run() error {
	fmt.Println("Hello, World!")

	return nil
}
