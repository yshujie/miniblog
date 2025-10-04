package module

import (
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

// Publish 上架模块
func (c *ModuleController) Publish(ctx *gin.Context) {
	log.C(ctx).Infow("Publish module function called")

	resp, err := c.biz.ModuleBiz().Publish(ctx, ctx.Param("code"))
	if err != nil {
		log.C(ctx).Errorw("publish module failed", "error", err, "code", ctx.Param("code"))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, resp)
}

// Unpublish 下架模块
func (c *ModuleController) Unpublish(ctx *gin.Context) {
	log.C(ctx).Infow("Unpublish module function called")

	resp, err := c.biz.ModuleBiz().Unpublish(ctx, ctx.Param("code"))
	if err != nil {
		log.C(ctx).Errorw("unpublish module failed", "error", err, "code", ctx.Param("code"))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, resp)
}

// Delete 物理删除模块
func (c *ModuleController) Delete(ctx *gin.Context) {
	log.C(ctx).Infow("Delete module function called")

	if err := c.biz.ModuleBiz().Delete(ctx, ctx.Param("code")); err != nil {
		log.C(ctx).Errorw("delete module failed", "error", err, "code", ctx.Param("code"))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, nil)
}
