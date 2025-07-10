package model

import (
	"time"

	"github.com/yshujie/miniblog/pkg/auth"
	"github.com/yshujie/miniblog/pkg/util/idutil"
	"gorm.io/gorm"
)

type UserM struct {
	ID           uint64    `gorm:"column:id;primary_key"`
	Username     string    `gorm:"column:username;not null"`
	Password     string    `gorm:"column:password;not null"`
	Nickname     string    `gorm:"column:nickname"`
	Avatar       string    `gorm:"column:avatar"`
	Email        string    `gorm:"column:email"`
	Phone        string    `gorm:"column:phone"`
	Introduction string    `gorm:"column:introduction"`
	Status       int       `gorm:"column:status"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名
func (u *UserM) TableName() string {
	return "user"
}

// BeforeCreate 在创建前设置信息
func (u *UserM) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = idutil.GetIntID()
	u.Password, err = auth.Encrypt(u.Password)
	if err != nil {
		return err
	}

	return
}

// BeforeUpdate 在更新前设置信息
func (u *UserM) BeforeUpdate(tx *gorm.DB) (err error) {
	u.UpdatedAt = time.Now()
	return
}
