package section

import (
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

// Publish 上架 section
func (c *SectionController) Publish(ctx *gin.Context) {
	log.C(ctx).Infow("Publish section function called")

	resp, err := c.biz.SectionBiz().Publish(ctx, ctx.Param("code"))
	if err != nil {
		log.C(ctx).Errorw("publish section failed", "error", err, "code", ctx.Param("code"))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, resp)
}

// Unpublish 下架 section
func (c *SectionController) Unpublish(ctx *gin.Context) {
	log.C(ctx).Infow("Unpublish section function called")

	resp, err := c.biz.SectionBiz().Unpublish(ctx, ctx.Param("code"))
	if err != nil {
		log.C(ctx).Errorw("unpublish section failed", "error", err, "code", ctx.Param("code"))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, resp)
}

// Delete 物理删除 section
func (c *SectionController) Delete(ctx *gin.Context) {
	log.C(ctx).Infow("Delete section function called")

	if err := c.biz.SectionBiz().Delete(ctx, ctx.Param("code")); err != nil {
		log.C(ctx).Errorw("delete section failed", "error", err, "code", ctx.Param("code"))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, nil)
}
