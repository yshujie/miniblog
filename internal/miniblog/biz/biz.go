package biz

import (
	"github.com/yshujie/miniblog/internal/miniblog/biz/auth"
	"github.com/yshujie/miniblog/internal/miniblog/biz/module"
	"github.com/yshujie/miniblog/internal/miniblog/biz/user"
	"github.com/yshujie/miniblog/internal/miniblog/store"
)

// IBiz 业务接口，定义了 Biz 层需要实现的方法
type IBiz interface {
	UserBiz() user.IUserBiz
	AuthBiz() auth.IAuthBiz
	ModuleBiz() module.IModuleBiz
}

// biz 业务实现
type biz struct {
	ds store.IStore
}

// 确保 biz 实现了 IBiz 接口
var _ IBiz = (*biz)(nil)

// NewBiz 简单工厂函数，创建 biz 实例
func NewBiz(ds store.IStore) *biz {
	return &biz{ds: ds}
}

// Users 返回用户业务实例
func (b *biz) UserBiz() user.IUserBiz {
	return user.New(b.ds)
}

// AuthBiz 返回认证业务实例
func (b *biz) AuthBiz() auth.IAuthBiz {
	return auth.NewAuthBiz(b.ds)
}

// ModuleBiz 返回模块业务实例
func (b *biz) ModuleBiz() module.IModuleBiz {
	return module.New(b.ds)
}
