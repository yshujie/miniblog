package article

import (
	"context"
	"errors"
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
)

type fakeArticleStore struct {
	articles map[uint64]*model.Article
}

func newFakeArticleStore() *fakeArticleStore {
	return &fakeArticleStore{articles: make(map[uint64]*model.Article)}
}

func (f *fakeArticleStore) Create(article *model.Article) error {
	if article.ID == 0 {
		article.ID = uint64(len(f.articles) + 1)
	}
	copy := *article
	f.articles[article.ID] = &copy
	return nil
}

func (f *fakeArticleStore) GetOne(id uint64) (*model.Article, error) {
	if article, ok := f.articles[id]; ok {
		copy := *article
		return &copy, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeArticleStore) GetList(filter interface{}, page int, limit int) ([]*model.Article, error) {
	return nil, nil
}

func (f *fakeArticleStore) Update(article *model.Article) error {
	if _, ok := f.articles[article.ID]; !ok {
		return errors.New("not found")
	}
	copy := *article
	f.articles[article.ID] = &copy
	return nil
}

type fakeArticleBizStore struct {
	articles *fakeArticleStore
}

func (f *fakeArticleBizStore) DB() *gorm.DB                 { return nil }
func (f *fakeArticleBizStore) Users() store.UserStore       { return nil }
func (f *fakeArticleBizStore) Modules() store.ModuleStore   { return nil }
func (f *fakeArticleBizStore) Sections() store.SectionStore { return nil }
func (f *fakeArticleBizStore) Articles() store.ArticleStore { return f.articles }

func TestArticleBizPublish(t *testing.T) {
	fakeStore := &fakeArticleBizStore{articles: newFakeArticleStore()}
	fakeStore.articles.articles[1] = &model.Article{ID: 1, Status: model.ArticleStatusDraft}
	biz := articleBiz{ds: fakeStore}

	if err := biz.Publish(context.Background(), 1); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if fakeStore.articles.articles[1].Status != model.ArticleStatusPublished {
		t.Fatalf("expected status published, got %d", fakeStore.articles.articles[1].Status)
	}
}

func TestArticleBizUnpublish(t *testing.T) {
	fakeStore := &fakeArticleBizStore{articles: newFakeArticleStore()}
	fakeStore.articles.articles[1] = &model.Article{ID: 1, Status: model.ArticleStatusPublished}
	biz := articleBiz{ds: fakeStore}

	if err := biz.Unpublish(context.Background(), 1); err != nil {
		t.Fatalf("Unpublish returned error: %v", err)
	}
	if fakeStore.articles.articles[1].Status != model.ArticleStatusUnpublished {
		t.Fatalf("expected status unpublished, got %d", fakeStore.articles.articles[1].Status)
	}
}

func TestArticleBizPublishNotFound(t *testing.T) {
	biz := articleBiz{ds: &fakeArticleBizStore{articles: newFakeArticleStore()}}

	if err := biz.Publish(context.Background(), 99); err == nil {
		t.Fatalf("expected error when publishing non-existent article")
	}
}
