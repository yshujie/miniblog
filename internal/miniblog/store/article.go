package store

import (
	"context"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
)

type ArticleStore interface {
	Create(ctx context.Context, article *model.Article) (*model.Article, error)
	Get(ctx context.Context, id int) (*model.Article, error)
	GetListBySectionCode(ctx context.Context, sectionCode string) ([]*model.Article, error)
}

// ArticleStore 接口的实现
type articles struct {
	db *gorm.DB
}

var _ ArticleStore = &articles{}

// newArticles 创建文章存储实例
func newArticles(db *gorm.DB) *articles {
	return &articles{db}
}

// Create 创建文章
func (a *articles) Create(ctx context.Context, article *model.Article) (*model.Article, error) {
	err := a.db.Create(article).Error
	if err != nil {
		return nil, err
	}
	return article, nil
}

// Get 获取文章
func (a *articles) Get(ctx context.Context, id int) (*model.Article, error) {
	var article model.Article
	if err := a.db.Where("id = ?", id).First(&article).Error; err != nil {
		return nil, err
	}
	return &article, nil
}

// GetListBySectionCode 获取文章列表
func (a *articles) GetListBySectionCode(ctx context.Context, sectionCode string) ([]*model.Article, error) {
	var articles []*model.Article
	return articles, a.db.Where("section_code = ?", sectionCode).Find(&articles).Error
}
