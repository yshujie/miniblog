package store

import (
	"sync"

	"gorm.io/gorm"
)

var (
	once sync.Once
	// 全局变量，方便其他包直接调用已经初始化好的 S 实例
	S *datastore
)

// IStore 数据库操作接口
type IStore interface {
	DB() *gorm.DB
	Users() UserStore
}

// datastore 数据库操作
type datastore struct {
	db *gorm.DB
}

var _ IStore = (*datastore)(nil)

// NewStore 创建一个 Store 实例
func NewStore(db *gorm.DB) *datastore {
	once.Do(func() {
		S = &datastore{db}
	})
	return S
}

// DB 返回一个实现了 UserStore 接口的实例
func (s *datastore) DB() *gorm.DB {
	return s.db
}

// User 返回一个实现了 UserStore 接口的实例
func (s *datastore) Users() UserStore {
	return newUsers(s.db)
}
