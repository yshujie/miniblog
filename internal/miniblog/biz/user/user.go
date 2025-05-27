package user

import (
	"context"
	"regexp"
	"time"

	"github.com/jinzhu/copier"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/known"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
	"github.com/yshujie/miniblog/pkg/auth"
)

// UserBiz 用户业务接口
type IUserBiz interface {
	Create(ctx context.Context, r *v1.CreateUserRequest) error
	ChangePassword(ctx context.Context, username string, r *v1.ChangePasswordRequest) error
	Get(ctx context.Context, username string) (*v1.GetUserResponse, error)
	GetMyInfo(ctx context.Context) (*v1.GetUserResponse, error)
}

// userBiz 用户业务实现
type userBiz struct {
	ds store.IStore
}

// 确保 userBiz 实现了 UserBiz 接口
var _ IUserBiz = (*userBiz)(nil)

// New 简单工程函数，创建 userBiz 实例
func New(ds store.IStore) *userBiz {
	return &userBiz{ds}
}

// Create 创建用户
func (b *userBiz) Create(ctx context.Context, r *v1.CreateUserRequest) error {
	var userM model.UserM
	_ = copier.Copy(&userM, r)

	log.C(ctx).Infow("start to create user in biz layer", "username", r.Username)

	// 创建用户
	if err := b.ds.Users().Create(&userM); err != nil {
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

// ChangePassword 修改密码
func (b *userBiz) ChangePassword(ctx context.Context, username string, r *v1.ChangePasswordRequest) error {

	// 获取用户
	userM, err := b.ds.Users().Get(username)
	if err != nil {
		return err
	}

	// 对比密码
	if err := auth.Compare(userM.Password, r.OldPassword); err != nil {
		return errno.ErrPasswordIncorrect
	}

	// 更新密码
	userM.Password, _ = auth.Encrypt(r.NewPassword)
	if err := b.ds.Users().Update(userM); err != nil {
		return err
	}

	return nil
}

// Get 获取用户
func (b *userBiz) Get(ctx context.Context, username string) (*v1.GetUserResponse, error) {
	user, err := b.ds.Users().Get(username)
	if err != nil {
		return nil, err
	}

	var resp v1.GetUserResponse
	resp.User = v1.UserInfo{
		Username:     user.Username,
		Nickname:     user.Nickname,
		Avatar:       user.Avatar,
		Introduction: user.Introduction,
		Email:        user.Email,
		Phone:        user.Phone,
		Roles:        getAdminRoles(),
		CreatedAt:    user.CreatedAt.Format(time.DateTime),
		UpdatedAt:    user.UpdatedAt.Format(time.DateTime),
	}
	return &resp, nil
}

// GetMyInfo 获取当前用户信息
func (b *userBiz) GetMyInfo(ctx context.Context) (*v1.GetUserResponse, error) {
	// 从上下文中获取当前用户的用户名
	username := ctx.Value(known.XUsernameKey)
	if username == nil {
		return nil, errno.ErrUserNotFound
	}

	// 根据用户名获取用户信息
	user, err := b.ds.Users().Get(username.(string))
	if err != nil {
		return nil, err
	}

	// 将用户信息转换为响应对象
	var resp v1.GetUserResponse
	resp.User = v1.UserInfo{
		Username:     user.Username,
		Nickname:     user.Nickname,
		Avatar:       user.Avatar,
		Introduction: user.Introduction,
		Email:        user.Email,
		Phone:        user.Phone,
		Roles:        getAdminRoles(),
		CreatedAt:    user.CreatedAt.Format(time.DateTime),
		UpdatedAt:    user.UpdatedAt.Format(time.DateTime),
	}
	return &resp, nil
}

// getAdminRoles 获取管理员角色
func getAdminRoles() []string {
	return []string{"admin"}
}
