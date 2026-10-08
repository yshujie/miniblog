package blog

import (
	"context"
	"github.com/yshujie/miniblog/internal/miniblog/biz/reading"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// Blog is the v1 compatibility adapter. Public visibility and queries belong to reading.
type IBlogBiz interface {
	GetModuleList(context.Context) (*v1.GetModuleListResponse, error)
	GetModuleDetail(context.Context, *v1.GetModuleDetailRequest) (*v1.GetModuleDetailResponse, error)
	GetArticleDetail(context.Context, *v1.GetArticleDetailRequest) (*v1.GetArticleDetailResponse, error)
}
type blogBiz struct{ reading *reading.Service }

func New(ds store.IStore) *blogBiz { return &blogBiz{reading: reading.New(ds)} }
func (b *blogBiz) GetModuleList(ctx context.Context) (*v1.GetModuleListResponse, error) {
	return b.reading.Modules(ctx)
}
func (b *blogBiz) GetModuleDetail(ctx context.Context, r *v1.GetModuleDetailRequest) (*v1.GetModuleDetailResponse, error) {
	return b.reading.Module(ctx, r.ModuleCode)
}
func (b *blogBiz) GetArticleDetail(ctx context.Context, r *v1.GetArticleDetailRequest) (*v1.GetArticleDetailResponse, error) {
	return b.reading.Article(ctx, r.ArticleID)
}
