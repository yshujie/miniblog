package store

import (
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/pkg/log"
	"gorm.io/gorm"
)

type ArticleStore interface {
	Create(article *model.Article) error
	GetOne(id uint64) (*model.Article, error)
	GetList(filter interface{}, page int, limit int) ([]*model.Article, error)
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
func (a *articles) GetOne(id uint64) (*model.Article, error) {
	var article model.Article
	if err := a.db.Where("id = ?", id).First(&article).Error; err != nil {
		return nil, err
	}
	return &article, nil
}

// GetList 获取文章列表
func (a *articles) GetList(filter interface{}, page int, limit int) ([]*model.Article, error) {
	log.Infow("GetList", "filter", filter, "page", page, "limit", limit)
	var articles []*model.Article

	query := a.db.Where(filter)

	// 分页查询
	query = query.Offset((page - 1) * limit).Limit(limit)

	return articles, query.Find(&articles).Error
}

// Update 更新文章
func (a *articles) Update(article *model.Article) error {
	return a.db.Save(article).Error
}
