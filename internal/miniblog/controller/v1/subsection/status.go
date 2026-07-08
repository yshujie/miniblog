package subsection

import (
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

func (c *SubsectionController) Publish(ctx *gin.Context) {
	log.C(ctx).Infow("Publish subsection function called")

	resp, err := c.biz.SubsectionBiz().Publish(ctx, ctx.Param("code"))
	if err != nil {
		log.C(ctx).Errorw("publish subsection failed", "error", err, "code", ctx.Param("code"))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, resp)
}

func (c *SubsectionController) Unpublish(ctx *gin.Context) {
	log.C(ctx).Infow("Unpublish subsection function called")

	resp, err := c.biz.SubsectionBiz().Unpublish(ctx, ctx.Param("code"))
	if err != nil {
		log.C(ctx).Errorw("unpublish subsection failed", "error", err, "code", ctx.Param("code"))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, resp)
}

func (c *SubsectionController) Delete(ctx *gin.Context) {
	log.C(ctx).Infow("Delete subsection function called")

	if err := c.biz.SubsectionBiz().Delete(ctx, ctx.Param("code")); err != nil {
		log.C(ctx).Errorw("delete subsection failed", "error", err, "code", ctx.Param("code"))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, nil)
}
