package visitorserver

import (
	"github.com/yshujie/blog-serve/pkg/app"
)

func NewApp() *app.App {
	return app.NewApp("visitor-server", "visitor-server", app.WithRunFunc(run))
}

func run(basename string) error {
	return nil
}
