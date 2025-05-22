package article

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// Publish 发布文章
func (c *ArticleController) Publish(ctx *gin.Context) {
	log.C(ctx).Infow("Publish article function called")

	request := &v1.ArticleIdRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		log.C(ctx).Errorw("failed to bind request", "error", err)
		core.WriteResponse(ctx, err, nil)
		return
	}

	if err := c.biz.ArticleBiz().Publish(ctx, request); err != nil {
		log.C(ctx).Errorw("publish article failed", "error", err, fmt.Sprintf("%T", err))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, nil)
}

// Unpublish 下架文章
func (c *ArticleController) Unpublish(ctx *gin.Context) {
	log.C(ctx).Infow("Unpublish article function called")

	request := &v1.ArticleIdRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		log.C(ctx).Errorw("failed to bind request", "error", err)
		core.WriteResponse(ctx, err, nil)
		return
	}

	if err := c.biz.ArticleBiz().Unpublish(ctx, request); err != nil {
		log.C(ctx).Errorw("unpublish article failed", "error", err, fmt.Sprintf("%T", err))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, nil)
}
