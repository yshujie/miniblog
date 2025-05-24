package article

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// GetList 获取所有文章
func (c *ArticleController) GetList(ctx *gin.Context) {
	log.C(ctx).Infow("Get all articles function called")

	// 获取请求参数
	var req v1.ArticleListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	// 验证请求参数
	if err := validator.New().Struct(req); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	// 获取文章列表
	articles, err := c.biz.ArticleBiz().GetList(ctx, &req)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, articles)
}

// GetOneByCode 获取文章详情
func (c *ArticleController) GetOne(ctx *gin.Context) {
	log.C(ctx).Infow("Get one article function called")

	// 获取文章ID
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	// 获取文章详情
	article, err := c.biz.ArticleBiz().GetOne(ctx, id)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, article)
}
