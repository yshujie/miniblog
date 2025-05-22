package article

import (
	"fmt"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// Update 更新文章
func (c *ArticleController) Update(ctx *gin.Context) {
	log.C(ctx).Infow("Update article function called")

	request := &v1.UpdateArticleRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		log.C(ctx).Errorw("failed to bind request", "error", err)
		core.WriteResponse(ctx, err, nil)
		return
	}

	if _, err := govalidator.ValidateStruct(request); err != nil {
		log.C(ctx).Errorw("invalid request parameters", "error", err)
		core.WriteResponse(ctx, errno.ErrInvalidParameter.SetMessage("%s", err.Error()), nil)
		return
	}

	// 更新文章
	if err := c.biz.ArticleBiz().Update(ctx, request); err != nil {
		log.C(ctx).Errorw("update article failed", "error", err, fmt.Sprintf("%T", err))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, nil)
}
