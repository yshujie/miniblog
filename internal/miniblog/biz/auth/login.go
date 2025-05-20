package auth

import (
	"context"

	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
	"github.com/yshujie/miniblog/pkg/auth"
	"github.com/yshujie/miniblog/pkg/token"
)

// Login 登录
func (b *authBiz) Login(ctx context.Context, r *v1.LoginRequest) (*v1.LoginResponse, error) {
	// 检查用户是否存在
	if err := b.checkUserExist(ctx, r.Username); err != nil {
		return nil, err
	}

	// 检查密码是否正确
	if err := b.checkPassword(ctx, r.Username, r.Password); err != nil {
		return nil, err
	}

	// 密码正确，则签发 token
	token, err := token.Sign(r.Username)
	if err != nil {
		return nil, errno.ErrTokenSign
	}

	return &v1.LoginResponse{Token: token}, nil
}

// 检查用户是否存在
func (b *authBiz) checkUserExist(ctx context.Context, username string) error {
	// 获取用户信息
	_, err := b.ds.Users().Get(username)
	if err != nil {
		return errno.ErrUserNotFound
	}

	return nil
}

func (b *authBiz) checkPassword(ctx context.Context, username string, password string) error {
	// 获取用户信息
	user, err := b.ds.Users().Get(username)
	if err != nil {
		return errno.ErrUserNotFound
	}

	// 对比密码
	if err := auth.Compare(user.Password, password); err != nil {
		return errno.ErrPasswordIncorrect
	}

	return nil
}
