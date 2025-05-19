package article

import (
	"context"
	"strings"

	"github.com/yshujie/miniblog/internal/miniblog/store"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// ArticleBiz 文章业务接口
type IArticleBiz interface {
	GetList(ctx context.Context, sectionCode string) ([]*v1.GetArticleListResponse, error)
	GetOne(ctx context.Context, id int) (*v1.GetArticleResponse, error)
}

// articleBiz 文章业务实现
type articleBiz struct {
	ds store.IStore
}

// 确保 articleBiz 实现了 IArticleBiz 接口
var _ IArticleBiz = (*articleBiz)(nil)

// New 简单工程函数，创建 articleBiz 实例
func New(ds store.IStore) *articleBiz {
	return &articleBiz{ds}
}

// GetList 获取所有文章
func (b *articleBiz) GetList(ctx context.Context, sectionCode string) ([]*v1.GetArticleListResponse, error) {
	articles, err := b.ds.Articles().GetListBySectionCode(ctx, sectionCode)
	if err != nil {
		return nil, err
	}

	response := make([]*v1.GetArticleListResponse, 0)
	for _, article := range articles {
		response = append(response, &v1.GetArticleListResponse{
			Articles: []*v1.ArticleInfo{
				{
					ID:          article.ID,
					Title:       article.Title,
					Content:     article.Content,
					SectionCode: article.SectionCode,
					Author:      article.Author,
					Tags:        strings.Split(article.Tags, ","),
					CreateTime:  article.CreateTime,
					UpdateTime:  article.UpdateTime,
				},
			},
		})
	}

	return response, nil
}

// GetOne 获取文章详情
func (b *articleBiz) GetOne(ctx context.Context, id int) (*v1.GetArticleResponse, error) {
	article, err := b.ds.Articles().Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return &v1.GetArticleResponse{
		Article: &v1.ArticleInfo{
			ID:          article.ID,
			Title:       article.Title,
			Content:     article.Content,
			SectionCode: article.SectionCode,
			Author:      article.Author,
			Tags:        strings.Split(article.Tags, ","),
			CreateTime:  article.CreateTime,
			UpdateTime:  article.UpdateTime,
		},
	}, nil
}
