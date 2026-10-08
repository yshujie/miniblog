package article

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

// Publish 发布文章
func (c *ArticleController) Publish(ctx *gin.Context) {
	log.C(ctx).Infow("Publish article function called")

	// 获取文章ID
	articleId, ok := commandID(ctx)
	if !ok {
		return
	}

	if err := c.biz.ArticleBiz().Publish(ctx.Request.Context(), articleId); err != nil {
		log.C(ctx).Errorw("publish article failed", "error", err, fmt.Sprintf("%T", err))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, nil)
}

// Unpublish 下架文章
func (c *ArticleController) Unpublish(ctx *gin.Context) {
	log.C(ctx).Infow("Unpublish article function called")

	// 获取文章ID
	articleId, ok := commandID(ctx)
	if !ok {
		return
	}

	if err := c.biz.ArticleBiz().Unpublish(ctx.Request.Context(), articleId); err != nil {
		log.C(ctx).Errorw("unpublish article failed", "error", err, fmt.Sprintf("%T", err))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, nil)
}
