package service

import (
	m "github.com/yshujie/blog-serve/internal/model"
)

// article service interface
type ArticleSrv interface {
	GetArticleList(page int, pageSize int) ([]m.Article, error)
}

type ArticleListRequest struct {
	Filter   *ArticleFilter     `json:"filter"`
	Sorting  *Sorting           `json:"sorting"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Format   *ListFormatOptions `json:"format"`
}

type ArticleListResponse struct {
	Items      []*ArticleListItem `json:"items"`
	Pagination *Pagination        `json:"pagination"`
}

type ArticleService struct {
	querying   ArticleQuerying
	formatting ArticleFormatting
}

func (s *ArticleService) GetArticleList(req *ArticleListRequest) (*ArticleListResponse, error) {
	// 1. 准备分页参数
	pagination := &Pagination{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	// 2. 获取总数
	total, err := s.querying.Count(req.Filter)
	if err != nil {
		return nil, err
	}
	pagination.Total = total

	// 3. 查询列表
	articles, err := s.querying.QueryList(req.Filter, pagination, req.Sorting)
	if err != nil {
		return nil, err
	}

	// 4. 格式化列表
	items := s.formatting.FormatList(articles, req.Format)

	return &ArticleListResponse{
		Items:      items,
		Pagination: pagination,
	}, nil
}
