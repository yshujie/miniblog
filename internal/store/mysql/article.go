package mysql

import (
	"github.com/jinzhu/gorm"
	m "github.com/yshujie/blog-serve/internal/model"
)

// article store interface
type ArticleStore interface {
	Create(article *m.Article) error
	Update(article *m.Article) error
	Delete(article *m.Article) error
	GetById(id int) (*m.Article, error)
	GetByTitle(title string) (*m.Article, error)
	GetArticleList(limit, offset int) ([])
}

type articleStore struct {
	db *gorm.DB
}

// new article Store
func NewArticleStore(db *gorm.DB) *articleStore {
	return &articleStore{db: db}
}

// create article
func (r *articleStore) Create(article *m.Article) error {
	return r.db.Create(article).Error
}

// update article
func (r *articleStore) Update(article *m.Article) error {
	return r.db.Save(article).Error
}

// delete article
func (r *articleStore) Delete(article *m.Article) error {
	return r.db.Delete(article).Error
}

// get article by id
func (r *articleStore) GetById(id int) (*m.Article, error) {
	var article m.Article
	return &article, r.db.Where("id = ?", id).First(&article).Error
}

// get article by title
func (r *articleStore) GetByTitle(title string) (*m.Article, error) {
	var article m.Article
	return &article, r.db.Where("title = ?", title).First(&article).Error
}

// get article list
func (r *articleStore) GetArticleList(limit, offset int) (error) {
	var articles []m.Article

}
