package main

import (
	"os"

	"github.com/yshujie/miniblog/internal/miniblog"
)

func main() {
	cmd := miniblog.NewMiniBlogCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
