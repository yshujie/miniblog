package article

import (
	"context"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// Business inputs do not depend on JSON binding or API version.
type RegisterInput struct {
	ExternalLink, Title, SectionCode, SubsectionCode, Author string
	Tags                                                     []string
	Publish                                                  bool
}
type UpdateInput struct {
	ID                                                                   uint64
	Title, Author, ModuleCode, SectionCode, SubsectionCode, ExternalLink string
	Tags                                                                 []string
	Content                                                              *string
}
type MoveInput struct{ SectionCode, SubsectionCode string }
type ReorderInput struct {
	SectionCode, SubsectionCode string
	ArticleIDs                  []uint64
}

// Existing v1 callers use these adapters; all commands execute the same use cases.
func (b *articleBiz) Register(ctx context.Context, r *v1.RegisterArticleRequest) (*v1.RegisterArticleResponse, error) {
	return b.RegisterSource(ctx, RegisterInput{ExternalLink: r.ExternalLink, Title: r.Title, SectionCode: r.SectionCode, SubsectionCode: r.SubsectionCode, Author: r.Author, Tags: r.Tags, Publish: r.Publish})
}
func (b *articleBiz) Update(ctx context.Context, r *v1.UpdateArticleRequest) (*v1.ArticleInfoResponse, error) {
	id, err := ParseID(r.ID)
	if err != nil {
		return nil, err
	}
	return b.Edit(ctx, UpdateInput{ID: id, Title: r.Title, Author: r.Author, ModuleCode: r.ModuleCode, SectionCode: r.SectionCode, SubsectionCode: r.SubsectionCode, ExternalLink: r.ExternalLink, Tags: r.Tags, Content: r.Content})
}
func (b *articleBiz) Move(ctx context.Context, id uint64, r *v1.MoveArticleRequest) (*v1.ArticleCommandResponse, error) {
	return b.MoveArticle(ctx, id, MoveInput{SectionCode: r.SectionCode, SubsectionCode: r.SubsectionCode})
}
func (b *articleBiz) Reorder(ctx context.Context, r *v1.ReorderArticlesRequest) error {
	ids := make([]uint64, 0, len(r.ArticleIDs))
	for _, raw := range r.ArticleIDs {
		id, err := ParseID(raw)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}
	return b.ReorderArticles(ctx, ReorderInput{SectionCode: r.SectionCode, SubsectionCode: r.SubsectionCode, ArticleIDs: ids})
}
