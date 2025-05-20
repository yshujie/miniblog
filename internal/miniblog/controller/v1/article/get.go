package article

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

// GetList 获取所有文章
func (c *ArticleController) GetList(ctx *gin.Context) {
	log.C(ctx).Infow("Get all articles function called")

	// 获取文章列表
	articles, err := c.biz.ArticleBiz().GetList(ctx, ctx.Param("section_code"))
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
