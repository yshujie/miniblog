package mysql

import (
	"github.com/jinzhu/gorm"
	m "github.com/yshujie/blog-serve/internal/model"
)

type ArticleStore struct {
	db *gorm.DB
}

// new article Store
func NewArticleStore(db *gorm.DB) *ArticleStore {
	return &ArticleStore{db: db}
}

// create article
func (r *ArticleStore) Create(article *m.Article) error {
	return r.db.Create(article).Error
}

// update article
func (r *ArticleStore) Update(article *m.Article) error {
	return r.db.Save(article).Error
}

// delete article
func (r *ArticleStore) Delete(article *m.Article) error {
	return r.db.Delete(article).Error
}

// get article by id
func (r *ArticleStore) GetById(id int) (*m.Article, error) {
	var article m.Article
	return &article, r.db.Where("id = ?", id).First(&article).Error
}

// get article by title
func (r *ArticleStore) GetByTitle(title string) (*m.Article, error) {
	var article m.Article
	return &article, r.db.Where("title = ?", title).First(&article).Error
}

// get all articles
func (r *ArticleStore) GetAll() ([]m.Article, error) {
	var articles []m.Article
	return articles, r.db.Find(&articles).Error
}
