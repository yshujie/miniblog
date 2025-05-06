package store

import (
	"context"

	"github.com/yshujie/miniblog/internal/pkg/model"
	"gorm.io/gorm"
)

type UserStore interface {
	Create(ctx context.Context, user *model.User) error
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
func (u *users) Create(ctx context.Context, user *model.User) error {
	return u.db.Create(user).Error
}
