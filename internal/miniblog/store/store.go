package store

import (
	"context"

	"gorm.io/gorm"
)

var (
	// 全局变量，方便其他包直接调用已经初始化好的 S 实例
	S *datastore
)

// IStore 数据库操作接口
type IStore interface {
	DB() *gorm.DB
	Users() UserStore
	Modules() ModuleStore
	Sections() SectionStore
	Subsections() SubsectionStore
	Articles() ArticleStore
}

// datastore 数据库操作
type datastore struct {
	db *gorm.DB
}

var _ IStore = (*datastore)(nil)

// NewStore 创建一个 Store 实例
func NewStore(db *gorm.DB) *datastore {
	return &datastore{db: db}
}

// InTransaction binds every repository in the callback to the same transaction.
// In-memory implementations of IStore have no SQL connection.
func InTransaction(ctx context.Context, ds IStore, fn func(IStore) error) error {
	if ds.DB() == nil {
		return fn(ds)
	}
	return ds.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(NewStore(tx)) })
}

// DB 返回一个实现了 UserStore 接口的实例
func (s *datastore) DB() *gorm.DB {
	return s.db
}

// User 返回一个实现了 UserStore 接口的实例
func (s *datastore) Users() UserStore {
	return newUsers(s.db)
}

// Module 返回一个实现了 ModuleStore 接口的实例
func (s *datastore) Modules() ModuleStore {
	return newModules(s.db)
}

// Section 返回一个实现了 SectionStore 接口的实例
func (s *datastore) Sections() SectionStore {
	return newSections(s.db)
}

func (s *datastore) Subsections() SubsectionStore {
	return newSubsections(s.db)
}

// Article 返回一个实现了 ArticleStore 接口的实例
func (s *datastore) Articles() ArticleStore {
	return newArticles(s.db)
}
