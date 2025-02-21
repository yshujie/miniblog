package adminserver

import (
	"github.com/yshujie/blog-serve/pkg/app"
)

func NewApp() *app.App {
	return app.NewApp("admin-server", "admin-server", app.WithRunFunc(run))
}

func run(basename string) error {
	return nil
}
