package article

import (
	"context"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"testing"
)

func TestArticleBizPublish(t *testing.T) {
	db := contentDB(t)
	b := NewWithNotionClient(store.NewStore(db), nil)
	a := register(t, b, "https://example.com/publish", "s1", "", false)
	id, _ := ParseID(a.Article.ID)
	if err := b.Publish(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	var got model.Article
	db.First(&got, id)
	if got.Status != model.ArticleStatusPublished {
		t.Fatal(got.Status)
	}
}
func TestArticleBizUnpublish(t *testing.T) {
	db := contentDB(t)
	b := NewWithNotionClient(store.NewStore(db), nil)
	a := register(t, b, "https://example.com/unpublish", "s1", "", true)
	id, _ := ParseID(a.Article.ID)
	if err := b.Unpublish(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	var got model.Article
	db.First(&got, id)
	if got.Status != model.ArticleStatusUnpublished {
		t.Fatal(got.Status)
	}
}
func TestArticleBizPublishNotFound(t *testing.T) {
	b := NewWithNotionClient(store.NewStore(contentDB(t)), nil)
	errorHTTP(t, b.Publish(context.Background(), 99), 404)
}
