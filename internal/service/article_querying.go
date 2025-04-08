package service

import "time"

// 查询条件
type ArticleFilter struct {
	Keywords   string    `json:"keywords"`
	CategoryID uint      `json:"category_id"`
	Tags       []string  `json:"tags"`
	Status     string    `json:"status"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	AuthorID   uint      `json:"author_id"`
}

// 分页参数
type Pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// 排序参数
type Sorting struct {
	Field     string `json:"field"`     // created_at, updated_at, title
	Direction string `json:"direction"` // asc, desc
}

// 文章查询接口
type ArticleQuerying interface {
	// 查询文章列表
	QueryList(filter *ArticleFilter, pagination *Pagination, sorting *Sorting) ([]*model.Article, error)
	// 获取总数
	Count(filter *ArticleFilter) (int64, error)
}

// 实现
type articleQuerying struct {
	repo  repository.ArticleRepository
	cache cache.ArticleCache
}

func (q *articleQuerying) QueryList(filter *ArticleFilter, pagination *Pagination, sorting *Sorting) ([]*model.Article, error) {
	// 1. 检查缓存
	cacheKey := q.generateCacheKey(filter, pagination, sorting)
	if cached := q.cache.Get(cacheKey); cached != nil {
		return cached, nil
	}

	// 2. 构建查询条件
	query := q.buildQuery(filter)

	// 3. 应用排序
	query = q.applySorting(query, sorting)

	// 4. 应用分页
	query = q.applyPagination(query, pagination)

	// 5. 执行查询
	articles, err := q.repo.FindByQuery(query)
	if err != nil {
		return nil, err
	}

	// 6. 设置缓存
	q.cache.Set(cacheKey, articles)

	return articles, nil
}
