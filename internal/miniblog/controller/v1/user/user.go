package user

import (
	"github.com/yshujie/miniblog/internal/miniblog/biz"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/log"
	"github.com/yshujie/miniblog/pkg/auth"
)

// UserController 用户控制器
type UserController struct {
	a *auth.Authz
	b biz.IBiz
}

// New 简单工厂函数，创建 UserController 实例
func New(ds store.IStore, a *auth.Authz) *UserController {
	log.Infow("... new user controller")
	return &UserController{
		a: a,
		b: biz.NewBiz(ds),
	}
}
