package article

import (
	"context"
	"strings"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// ArticleBiz 文章业务接口
type IArticleBiz interface {
	Create(ctx context.Context, r *v1.CreateArticleRequest) (*v1.CreateArticleResponse, error)
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

// Create 创建文章
func (b *articleBiz) Create(ctx context.Context, r *v1.CreateArticleRequest) (*v1.CreateArticleResponse, error) {
	// 检查 section_code 是否已存在
	existingSection, err := b.ds.Sections().GetByCode(ctx, r.SectionCode)
	if err != nil {
		return nil, err
	}
	if existingSection == nil {
		return nil, errno.ErrSectionNotFound
	}

	// 创建文章
	article, err := b.ds.Articles().Create(ctx, &model.Article{
		Title:       r.Title,
		Content:     r.Content,
		SectionCode: r.SectionCode,
		Author:      r.Author,
		Tags:        strings.Join(r.Tags, ","),
	})
	if err != nil {
		return nil, err
	}

	return &v1.CreateArticleResponse{
		Article: &v1.ArticleInfo{
			ID:          article.ID,
			Title:       article.Title,
			Content:     article.Content,
			SectionCode: article.SectionCode,
			Author:      article.Author,
			Tags:        strings.Split(article.Tags, ","),
			CreatedAt:   article.CreatedAt,
			UpdatedAt:   article.UpdatedAt,
		},
	}, nil
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
					CreatedAt:   article.CreatedAt,
					UpdatedAt:   article.UpdatedAt,
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
			CreatedAt:   article.CreatedAt,
			UpdatedAt:   article.UpdatedAt,
		},
	}, nil
}
