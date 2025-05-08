package user

import (
	"context"
	"regexp"

	"github.com/jinzhu/copier"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/log"
	"github.com/yshujie/miniblog/internal/pkg/model"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
	"github.com/yshujie/miniblog/pkg/auth"
	"github.com/yshujie/miniblog/pkg/token"
)

// UserBiz 用户业务接口
type UserBiz interface {
	Create(ctx context.Context, r *v1.CreateUserRequest) error
	Login(ctx context.Context, r *v1.LoginRequest) (*v1.LoginResponse, error)
	ChangePassword(ctx context.Context, username string, r *v1.ChangePasswordRequest) error
	Get(ctx context.Context, username string) (*v1.GetUserResponse, error)
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

	log.C(ctx).Infow("start to create user in biz layer", "username", r.Username)

	if err := b.ds.Users().Create(ctx, &userM); err != nil {
		if match, _ := regexp.MatchString("Duplicate entry '.*' for key 'username'", err.Error()); match {
			log.C(ctx).Warnw("user already exists", "username", r.Username, "error", err)
			return errno.ErrUserAlreadyExists
		}

		log.C(ctx).Errorw("create user failed in biz layer", "error", err, "username", r.Username)
		return err
	}

	log.C(ctx).Infow("create user success in biz layer", "username", r.Username)
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

// ChangePassword 修改密码
func (b *userBiz) ChangePassword(ctx context.Context, username string, r *v1.ChangePasswordRequest) error {

	// 获取用户
	userM, err := b.ds.Users().Get(ctx, username)
	if err != nil {
		return err
	}

	// 对比密码
	if err := auth.Compare(userM.Password, r.OldPassword); err != nil {
		return errno.ErrPasswordIncorrect
	}

	// 更新密码
	userM.Password, _ = auth.Encrypt(r.NewPassword)
	if err := b.ds.Users().Update(ctx, userM); err != nil {
		return err
	}

	return nil
}

// Get 获取用户
func (b *userBiz) Get(ctx context.Context, username string) (*v1.GetUserResponse, error) {
	user, err := b.ds.Users().Get(ctx, username)
	if err != nil {
		return nil, err
	}

	var resp v1.GetUserResponse
	_ = copier.Copy(&resp, user)

	resp.CreatedAt = user.CreatedAt.Format("2006-01-02 15:04:05")
	resp.UpdatedAt = user.UpdatedAt.Format("2006-01-02 15:04:05")

	return &resp, nil
}
