package store

import (
	"context"

	"github.com/yshujie/miniblog/internal/pkg/log"
	"github.com/yshujie/miniblog/internal/pkg/model"
	"gorm.io/gorm"
)

type UserStore interface {
	Create(ctx context.Context, user *model.UserM) error
	Get(ctx context.Context, username string) (*model.UserM, error)
	Update(ctx context.Context, user *model.UserM) error
}

// UserStore 接口的实现
type users struct {
	db *gorm.DB
}

var _ UserStore = &users{nil}

func newUsers(db *gorm.DB) *users {
	return &users{db}
}

// Create 创建用户
func (u *users) Create(ctx context.Context, user *model.UserM) error {
	log.C(ctx).Infow("start to create user in store layer", "username", user.Username)

	err := u.db.Create(&user).Error
	if err != nil {
		log.C(ctx).Errorw("create user failed in store layer", "error", err, "username", user.Username)
		return err
	}

	log.C(ctx).Infow("create user success in store layer", "username", user.Username)
	return nil
}

// GetUser 获取用户
func (u *users) Get(ctx context.Context, username string) (*model.UserM, error) {
	var user model.UserM
	if err := u.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// Update 更新用户
func (u *users) Update(ctx context.Context, user *model.UserM) error {
	return u.db.Save(user).Error
}
