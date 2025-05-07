package user

import (
	"context"
	"regexp"

	"github.com/jinzhu/copier"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/model"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
	"github.com/yshujie/miniblog/pkg/auth"
	"github.com/yshujie/miniblog/pkg/token"
)

// UserBiz 用户业务接口
type UserBiz interface {
	Create(ctx context.Context, r *v1.CreateUserRequest) error
}

// userBiz 用户业务实现
type userBiz struct {
	ds store.IStore
}

// 确保 userBiz 实现了 UserBiz 接口
var _ UserBiz = (*userBiz)(nil)

func New(ds store.IStore) *userBiz {
	return &userBiz{ds}
}

func (b *userBiz) Create(ctx context.Context, r *v1.CreateUserRequest) error {
	var userM model.UserM
	_ = copier.Copy(&userM, r)

	if err := b.ds.Users().Create(ctx, &userM); err != nil {
		if match, _ := regexp.MatchString("Duplicate entry '.*' for key 'username'", err.Error()); match {
			return errno.ErrUserAlreadyExists
		}

		return err
	}

	return nil
}

// Login 登录
func (b *userBiz) Login(ctx context.Context, r *v1.LoginRequest) (*v1.LoginResponse, error) {
	user, err := b.ds.Users().Get(ctx, r.Username)
	if err != nil {
		return nil, errno.ErrUserNotFound
	}

	// 对比密码
	if err := auth.Compare(user.Password, r.Password); err != nil {
		return nil, errno.ErrPasswordIncorrect
	}

	// 密码正确，则签发 token
	token, err := token.Sign(r.Username)
	if err != nil {
		return nil, errno.ErrTokenSign
	}

	return &v1.LoginResponse{
		Token: token,
	}, nil
}
