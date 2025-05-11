package auth

import (
	"context"

	"github.com/yshujie/miniblog/internal/miniblog/store"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// IAuthBiz 认证业务接口
type IAuthBiz interface {
	Register(ctx context.Context, r *v1.RegisterRequest) error
	Login(ctx context.Context, r *v1.LoginRequest) (*v1.LoginResponse, error)
	Logout(ctx context.Context, r *v1.LogoutRequest) error
}

// authBiz 认证业务实现
type authBiz struct {
	ds store.IStore
}

// 确保 authBiz 实现了 IAuthBiz 接口
var _ IAuthBiz = (*authBiz)(nil)

// NewAuthBiz 创建认证业务实例
func NewAuthBiz(ds store.IStore) *authBiz {
	return &authBiz{ds}
}

// Register 注册
func (b *authBiz) Register(ctx context.Context, r *v1.RegisterRequest) error {
	return nil
}

// Logout 登出
func (b *authBiz) Logout(ctx context.Context, r *v1.LogoutRequest) error {
	return nil
}
