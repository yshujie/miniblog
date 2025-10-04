package module

import (
	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// Update 更新模块
func (c *ModuleController) Update(ctx *gin.Context) {
	log.C(ctx).Infow("Update module function called")

	req := &v1.UpdateModuleRequest{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		log.C(ctx).Errorw("failed to bind request", "error", err)
		core.WriteResponse(ctx, err, nil)
		return
	}

	if _, err := govalidator.ValidateStruct(req); err != nil {
		log.C(ctx).Errorw("invalid request parameters", "error", err)
		core.WriteResponse(ctx, errno.ErrInvalidParameter.SetMessage("%s", err.Error()), nil)
		return
	}

	resp, err := c.biz.ModuleBiz().Update(ctx, ctx.Param("code"), req)
	if err != nil {
		log.C(ctx).Errorw("update module failed", "error", err, "code", ctx.Param("code"))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, resp)
}
