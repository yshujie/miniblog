package service

// 列表项格式化选项
type ListFormatOptions struct {
	WithContent   bool   `json:"with_content"`   // 是否包含内容
	ContentLength int    `json:"content_length"` // 内容截取长度
	WithTags      bool   `json:"with_tags"`      // 是否包含标签
	TimeFormat    string `json:"time_format"`    // 时间格式化方式
}

// 列表项
type ArticleListItem struct {
	ID        uint     `json:"id"`
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	Content   string   `json:"content,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// 文章格式化接口
type ArticleFormatting interface {
	// 格式化文章列表
	FormatList([]*model.Article, *ListFormatOptions) []*ArticleListItem
	// 格式化单个文章
	FormatItem(*model.Article, *ListFormatOptions) *ArticleListItem
}

// 实现
type articleFormatting struct {
	sanitizer *util.Sanitizer
}

func (f *articleFormatting) FormatList(articles []*model.Article, opts *ListFormatOptions) []*ArticleListItem {
	result := make([]*ArticleListItem, len(articles))
	for i, article := range articles {
		result[i] = f.FormatItem(article, opts)
	}
	return result
}

func (f *articleFormatting) FormatItem(article *model.Article, opts *ListFormatOptions) *ArticleListItem {
	item := &ArticleListItem{
		ID:        article.ID,
		Title:     article.Title,
		CreatedAt: article.CreatedAt.Format(opts.TimeFormat),
		UpdatedAt: article.UpdatedAt.Format(opts.TimeFormat),
	}

	// 格式化摘要
	item.Summary = f.formatSummary(article.Content, opts.ContentLength)

	// 根据选项决定是否包含其他信息
	if opts.WithContent {
		item.Content = f.sanitizer.Clean(article.Content)
	}
	if opts.WithTags {
		item.Tags = article.Tags
	}

	return item
}
