package module

import (
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

// GetList 获取所有模块
func (c *SectionController) GetList(ctx *gin.Context) {
	log.C(ctx).Infow("Get all sections function called")

	sections, err := c.biz.SectionBiz().GetList(ctx, ctx.Param("module_code"))
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, sections)
}

// GetOne 获取模块详情
func (c *SectionController) GetOne(ctx *gin.Context) {
	log.C(ctx).Infow("Get one section function called")

	section, err := c.biz.SectionBiz().GetOne(ctx, ctx.Param("code"))
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, section)
}
