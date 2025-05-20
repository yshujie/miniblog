package store

import (
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
)

type UserStore interface {
	Create(user *model.UserM) error
	Get(username string) (*model.UserM, error)
	Update(user *model.UserM) error
}

// UserStore 接口的实现
type users struct {
	db *gorm.DB
}

var _ UserStore = &users{}

// NewUsers 创建用户存储实例
func newUsers(db *gorm.DB) *users {
	return &users{db}
}

// Create 创建用户
func (u *users) Create(user *model.UserM) error {
	err := u.db.Create(&user).Error
	if err != nil {
		return err
	}

	return nil
}

// GetUser 获取用户
func (u *users) Get(username string) (*model.UserM, error) {
	var user model.UserM
	if err := u.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// Update 更新用户
func (u *users) Update(user *model.UserM) error {
	return u.db.Save(user).Error
}
