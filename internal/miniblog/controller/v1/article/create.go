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

// Create 创建 article 记录
func (c *ArticleController) Create(ctx *gin.Context) {
	log.C(ctx).Infow("Create article function called")
	if !c.registrationAllowed(ctx) {
		return
	}

	request := &v1.CreateArticleRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		log.C(ctx).Errorw("failed to bind request", "error", err)
		core.WriteResponse(ctx, err, nil)
		return
	}

	if _, err := govalidator.ValidateStruct(request); err != nil {
		log.C(ctx).Errorw("invalid request parameters", "error", err)
		core.WriteResponse(ctx, &errno.Errno{HTTP: 400, Code: errno.ErrInvalidParameter.Code, Message: err.Error()}, nil)
		return
	}

	response, err := c.biz.ArticleBiz().Create(ctx.Request.Context(), request)
	if err != nil {
		log.C(ctx).Errorw("create article failed", "error", err, "title", request.Title, "section_code", request.SectionCode, "error_type", fmt.Sprintf("%T", err))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, response)
}
