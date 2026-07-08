package subsection

import (
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

func (c *SubsectionController) GetList(ctx *gin.Context) {
	log.C(ctx).Infow("Get all subsections function called")

	subsections, err := c.biz.SubsectionBiz().GetList(ctx, ctx.Param("section_code"))
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, subsections)
}

func (c *SubsectionController) GetOne(ctx *gin.Context) {
	log.C(ctx).Infow("Get one subsection function called")

	subsection, err := c.biz.SubsectionBiz().GetOne(ctx, ctx.Param("code"))
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, subsection)
}
