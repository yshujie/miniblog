package article

import (
	"context"
	"strings"

	"github.com/spf13/viper"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/feishu"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// ArticleBiz 文章业务接口
type IArticleBiz interface {
	Create(ctx context.Context, r *v1.CreateArticleRequest) (*v1.CreateArticleResponse, error)
	GetList(ctx context.Context, sectionCode string) (*v1.GetArticleListResponse, error)
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
	existingSection, err := b.ds.Sections().GetByCode(r.SectionCode)
	if err != nil {
		return nil, err
	}
	if existingSection == nil {
		return nil, errno.ErrSectionNotFound
	}

	// 读取 article 内容
	content, err := loadArticleContent(r.ExternalLink, ctx)
	if err != nil {
		return nil, err
	}

	// 创建文章
	article := &model.Article{
		Title:       r.Title,
		Content:     content,
		SectionCode: r.SectionCode,
		Author:      r.Author,
		Tags:        strings.Join(r.Tags, ","),
	}
	if err := b.ds.Articles().Create(article); err != nil {
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

// loadArticleContent 加载文章内容
func loadArticleContent(externalLink string, ctx context.Context) (string, error) {
	// 读取文档内容
	content, err := feishu.GetClient(
		viper.GetString("feishu.doc_reader.app_id"),
		viper.GetString("feishu.doc_reader.app_secret"),
		ctx,
	).
		DocReader.ReadContent(externalLink, "docx", "markdown")
	if err != nil {
		log.Warnw("failed to read doc content", "error", err)
		return "", errno.ErrReadDocFailed
	}

	log.Infow("read doc content", "content", content)
	return content, nil
}

// GetList 获取所有文章
func (b *articleBiz) GetList(ctx context.Context, sectionCode string) (*v1.GetArticleListResponse, error) {
	articles, err := b.ds.Articles().GetListBySectionCode(sectionCode)
	if err != nil {
		return nil, err
	}

	response := &v1.GetArticleListResponse{
		Articles: make([]*v1.ArticleInfo, 0),
	}
	for _, article := range articles {
		response.Articles = append(response.Articles, &v1.ArticleInfo{
			ID:    article.ID,
			Title: article.Title,
		})
	}

	return response, nil
}

// GetOne 获取文章详情
func (b *articleBiz) GetOne(ctx context.Context, id int) (*v1.GetArticleResponse, error) {
	article, err := b.ds.Articles().GetOne(id)
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
