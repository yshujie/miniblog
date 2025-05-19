package store

import (
	"context"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
)

type ArticleStore interface {
	Create(ctx context.Context, article *model.Article) error
	Get(ctx context.Context, id int) (*model.Article, error)
	Update(ctx context.Context, article *model.Article) error
	Delete(ctx context.Context, id int) error
	GetListBySectionCode(ctx context.Context, sectionCode string, page, pageSize int) ([]*model.Article, error)
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
func (a *articles) Create(ctx context.Context, article *model.Article) error {
	return a.db.Create(article).Error
}

// Get 获取文章
func (a *articles) Get(ctx context.Context, id int) (*model.Article, error) {
	var article model.Article
	if err := a.db.Where("id = ?", id).First(&article).Error; err != nil {
		return nil, err
	}
	return &article, nil
}

// Update 更新文章
func (a *articles) Update(ctx context.Context, article *model.Article) error {
	return a.db.Model(&model.Article{}).Where("id = ?", article.ID).Updates(article).Error
}

// Delete 删除文章
func (a *articles) Delete(ctx context.Context, id int) error {
	return a.db.Where("id = ?", id).Delete(&model.Article{}).Error
}

// GetListBySectionCode 获取文章列表
func (a *articles) GetListBySectionCode(ctx context.Context, sectionCode string, page, pageSize int) ([]*model.Article, error) {
	var articles []*model.Article
	return articles, a.db.Where("section_code = ?", sectionCode).Offset((page - 1) * pageSize).Limit(pageSize).Find(&articles).Error
}
