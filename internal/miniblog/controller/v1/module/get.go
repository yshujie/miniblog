package module

import (
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

// GetAll 获取所有模块
func (c *ModuleController) GetAll(ctx *gin.Context) {
	log.C(ctx).Infow("Get all modules function called")

	modules, err := c.biz.ModuleBiz().GetAll(ctx.Request.Context())
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, modules)
}

// GetOneByCode 获取模块详情
func (c *ModuleController) GetOne(ctx *gin.Context) {
	log.C(ctx).Infow("Get one module function called")

	module, err := c.biz.ModuleBiz().GetOne(ctx.Request.Context(), ctx.Param("code"))
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, module)
}
