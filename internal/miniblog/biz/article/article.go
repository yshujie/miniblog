package article

import (
	"context"
	"strconv"
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
	Create(ctx context.Context, r *v1.CreateArticleRequest) (*v1.ArticleInfoResponse, error)
	Update(ctx context.Context, r *v1.UpdateArticleRequest) (*v1.ArticleInfoResponse, error)
	Publish(ctx context.Context, articleId uint64) error
	Unpublish(ctx context.Context, articleId uint64) error
	GetList(ctx context.Context, r *v1.ArticleListRequest) (*v1.GetArticleListResponse, error)
	GetOne(ctx context.Context, id uint64) (*v1.GetArticleResponse, error)
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
func (b *articleBiz) Create(ctx context.Context, r *v1.CreateArticleRequest) (*v1.ArticleInfoResponse, error) {
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

	// 获取当前 section 下的文章列表，计算下一个 pos 值
	filter := map[string]interface{}{
		"section_code": r.SectionCode,
	}
	existingArticles, err := b.ds.Articles().GetList(filter, 1, 1000) // 获取所有文章
	if err != nil {
		return nil, err
	}

	// 计算下一个 pos 值
	nextPos := 1
	if len(existingArticles) > 0 {
		maxPos := 0
		for _, existingArticle := range existingArticles {
			if existingArticle.Pos > maxPos {
				maxPos = existingArticle.Pos
			}
		}
		nextPos = maxPos + 1
	}

	// 创建文章
	article := &model.Article{
		Title:        r.Title,
		Content:      content,
		ExternalLink: r.ExternalLink,
		SectionCode:  r.SectionCode,
		Author:       r.Author,
		Tags:         strings.Join(r.Tags, ","),
		Pos:          nextPos,
	}
	// 存草稿
	article.SaveDraft()
	if err := b.ds.Articles().Create(article); err != nil {
		return nil, err
	}

	articleInfo, err := b.transformArticleInfo(article, false)
	if err != nil {
		return nil, err
	}

	return &v1.ArticleInfoResponse{
		Article: articleInfo,
	}, nil
}

// Update 更新文章
func (b *articleBiz) Update(ctx context.Context, r *v1.UpdateArticleRequest) (*v1.ArticleInfoResponse, error) {
	// 将字符串转换为 int64
	id, err := strconv.ParseInt(r.ID, 10, 64)
	if err != nil {
		return nil, err
	}
	article, err := b.ds.Articles().GetOne(uint64(id))
	if err != nil {
		return nil, err
	}

	// 更新文章
	article.Title = r.Title
	article.Author = r.Author
	article.Tags = strings.Join(r.Tags, ",")
	article.ExternalLink = r.ExternalLink
	article.SectionCode = r.SectionCode
	article.Content = r.Content
	// 注意：更新时不修改 pos 字段，保持原有排序

	// 存草稿
	article.SaveDraft()
	if err := b.ds.Articles().Update(article); err != nil {
		return nil, err
	}

	articleInfo, err := b.transformArticleInfo(article, false)
	if err != nil {
		return nil, err
	}

	return &v1.ArticleInfoResponse{
		Article: articleInfo,
	}, nil
}

// Publish 发布文章
func (b *articleBiz) Publish(ctx context.Context, articleId uint64) error {
	article, err := b.ds.Articles().GetOne(articleId)
	if err != nil {
		return errno.ErrArticleNotFound
	}

	// 发布文章
	article.Publish()
	if err := b.ds.Articles().Update(article); err != nil {
		return errno.ErrUpdateArticleFailed
	}

	return nil
}

// Unpublish 下架文章
func (b *articleBiz) Unpublish(ctx context.Context, articleId uint64) error {
	article, err := b.ds.Articles().GetOne(articleId)
	if err != nil {
		return errno.ErrArticleNotFound
	}

	// 下架文章
	article.Unpublish()
	if err := b.ds.Articles().Update(article); err != nil {
		return errno.ErrUpdateArticleFailed
	}

	return nil
}

// loadArticleContent 加载文章内容
func loadArticleContent(externalLink string, ctx context.Context) (string, error) {
	// 读取文档内容
	content, err := feishu.GetClient(
		viper.GetString("feishu.docreader.appid"),
		viper.GetString("feishu.docreader.appsecret"),
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
func (b *articleBiz) GetList(ctx context.Context, r *v1.ArticleListRequest) (*v1.GetArticleListResponse, error) {
	filter := map[string]interface{}{}

	if r.SectionCode != "" {
		filter["section_code"] = r.SectionCode
	}

	log.Infow("GetList", "filter", filter, "page", r.Page, "limit", r.Limit)
	articles, err := b.ds.Articles().GetList(filter, r.Page, r.Limit)
	if err != nil {
		return nil, err
	}

	response := &v1.GetArticleListResponse{
		Articles: make([]*v1.ArticleInfo, len(articles)),
	}

	for i, article := range articles {
		articleInfo, err := b.transformArticleInfo(article, true)
		if err != nil {
			return nil, err
		}
		response.Articles[i] = articleInfo
	}

	response.Total = len(articles)

	return response, nil
}

// GetOne 获取文章详情
func (b *articleBiz) GetOne(ctx context.Context, id uint64) (*v1.GetArticleResponse, error) {
	article, err := b.ds.Articles().GetOne(id)
	if err != nil {
		return nil, err
	}

	articleInfo, err := b.transformArticleInfo(article, false)
	if err != nil {
		return nil, err
	}

	return &v1.GetArticleResponse{
		Article: articleInfo,
	}, nil
}

// transformArticleInfo 转换文章信息
func (b *articleBiz) transformArticleInfo(article *model.Article, isSimple bool) (*v1.ArticleInfo, error) {
	module, section, err := b.queryArticleModuleAndSection(article)
	if err != nil {
		return nil, err
	}

	articleInfo := &v1.ArticleInfo{
		ID:    article.ID,
		Title: article.Title,
		Module: v1.ModuleInfo{
			Code:  module.Code,
			Title: module.Title,
		},
		Section: v1.SectionInfo{
			Code:  section.Code,
			Title: section.Title,
		},
		Author:       article.Author,
		Tags:         strings.Split(article.Tags, ","),
		Pos:          article.Pos,
		Status:       article.GetStatusString(),
		ExternalLink: article.ExternalLink,
	}

	if isSimple {
		return articleInfo, nil
	}

	articleInfo.Content = article.Content
	articleInfo.CreatedAt = article.CreatedAt.Format("2006-01-02 15:04:05")
	articleInfo.UpdatedAt = article.UpdatedAt.Format("2006-01-02 15:04:05")

	return articleInfo, nil
}

// queryArticleModuleAndSection 查询文章所属模块和章节
func (b *articleBiz) queryArticleModuleAndSection(article *model.Article) (*model.Module, *model.Section, error) {
	section, err := b.ds.Sections().GetByCode(article.SectionCode)
	if err != nil {
		return nil, nil, err
	}

	module, err := b.ds.Modules().GetByCode(section.ModuleCode)
	if err != nil {
		return nil, nil, err
	}

	return module, section, nil
}
