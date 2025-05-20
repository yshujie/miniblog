package module

import (
	"fmt"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// Create 创建模块
func (c *ModuleController) Create(ctx *gin.Context) {
	log.C(ctx).Infow("Create module function called")

	// 打印请求头、请求体
	log.C(ctx).Infow("request headers", "headers", ctx.Request.Header)
	log.C(ctx).Infow("request body", "body", ctx.Request.Body)

	// 承接请求参数
	request := &v1.CreateModuleRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		log.C(ctx).Errorw("failed to bind request", "error", err)
		core.WriteResponse(ctx, err, nil)
		return
	}

	// 验证请求参数
	if _, err := govalidator.ValidateStruct(request); err != nil {
		log.C(ctx).Errorw("invalid request parameters", "error", err)
		core.WriteResponse(ctx, errno.ErrInvalidParameter.SetMessage("%s", err.Error()), nil)
		return
	}

	// 调用 Biz 层，创建模块
	response, err := c.biz.ModuleBiz().Create(ctx, request)
	if err != nil {
		log.C(ctx).Errorw("create module failed", "error", err, "code", request.Code, "title", request.Title, "error_type", fmt.Sprintf("%T", err))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, response)
}
