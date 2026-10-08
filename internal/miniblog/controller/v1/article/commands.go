package article

import (
	"github.com/gin-gonic/gin"
	articlebiz "github.com/yshujie/miniblog/internal/miniblog/biz/article"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
	"os"
	"strconv"
)

func (c *ArticleController) registrationAllowed(ctx *gin.Context) bool {
	enabled, _ := strconv.ParseBool(os.Getenv("MINIBLOG_CONTENT_REGISTER_ENABLED"))
	ready := false
	if enabled {
		var err error
		ready, err = store.RegistrationReady(ctx.Request.Context(), c.ds.DB())
		if err != nil {
			ready = false
		}
	}
	if !ready {
		core.WriteResponse(ctx, &errno.Errno{HTTP: 503, Code: "ContentRegistrationUnavailable", Message: "文章登记暂不可用，请完成来源迁移并开启登记"}, nil)
	}
	return ready
}
func (c *ArticleController) Preview(ctx *gin.Context) {
	var r v1.PreviewSourceRequest
	if ctx.ShouldBindJSON(&r) != nil {
		core.WriteResponse(ctx, errno.ErrBind, nil)
		return
	}
	out, err := c.biz.ArticleBiz().Preview(ctx.Request.Context(), &r)
	core.WriteResponse(ctx, err, out)
}
func (c *ArticleController) Register(ctx *gin.Context) {
	if !c.registrationAllowed(ctx) {
		return
	}
	var r v1.RegisterArticleRequest
	if ctx.ShouldBindJSON(&r) != nil {
		core.WriteResponse(ctx, errno.ErrBind, nil)
		return
	}
	out, err := c.biz.ArticleBiz().Register(ctx.Request.Context(), &r)
	core.WriteResponse(ctx, err, out)
}
func commandID(ctx *gin.Context) (uint64, bool) {
	id, err := articlebiz.ParseID(ctx.Param("id"))
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return 0, false
	}
	return id, true
}
func (c *ArticleController) Move(ctx *gin.Context) {
	id, ok := commandID(ctx)
	if !ok {
		return
	}
	var r v1.MoveArticleRequest
	if ctx.ShouldBindJSON(&r) != nil {
		core.WriteResponse(ctx, errno.ErrBind, nil)
		return
	}
	out, err := c.biz.ArticleBiz().Move(ctx.Request.Context(), id, &r)
	core.WriteResponse(ctx, err, out)
}
func (c *ArticleController) Archive(ctx *gin.Context) {
	id, ok := commandID(ctx)
	if !ok {
		return
	}
	out, err := c.biz.ArticleBiz().Archive(ctx.Request.Context(), id)
	core.WriteResponse(ctx, err, out)
}
func (c *ArticleController) Restore(ctx *gin.Context) {
	id, ok := commandID(ctx)
	if !ok {
		return
	}
	out, err := c.biz.ArticleBiz().Restore(ctx.Request.Context(), id)
	core.WriteResponse(ctx, err, out)
}
func (c *ArticleController) Reorder(ctx *gin.Context) {
	var r v1.ReorderArticlesRequest
	if ctx.ShouldBindJSON(&r) != nil {
		core.WriteResponse(ctx, errno.ErrBind, nil)
		return
	}
	err := c.biz.ArticleBiz().Reorder(ctx.Request.Context(), &r)
	core.WriteResponse(ctx, err, nil)
}
