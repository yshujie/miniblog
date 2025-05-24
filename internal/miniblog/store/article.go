package store

import (
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
)

type ArticleStore interface {
	Create(article *model.Article) error
	GetOne(id int) (*model.Article, error)
	GetListBySectionCode(sectionCode string, page int, limit int) ([]*model.Article, error)
	Update(article *model.Article) error
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
func (a *articles) Create(article *model.Article) error {
	err := a.db.Create(article).Error
	if err != nil {
		return err
	}
	return nil
}

// Get 获取文章
func (a *articles) GetOne(id int) (*model.Article, error) {
	var article model.Article
	if err := a.db.Where("id = ?", id).First(&article).Error; err != nil {
		return nil, err
	}
	return &article, nil
}

// GetListBySectionCode 获取文章列表
func (a *articles) GetListBySectionCode(sectionCode string, page int, limit int) ([]*model.Article, error) {
	var articles []*model.Article
	return articles, a.db.Where("section_code = ?", sectionCode).Offset((page - 1) * limit).Limit(limit).Find(&articles).Error
}

// Update 更新文章
func (a *articles) Update(article *model.Article) error {
	return a.db.Save(article).Error
}
