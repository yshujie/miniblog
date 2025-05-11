package auth

import (
	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

func (ctrl *authController) Login(ctx *gin.Context) {
	log.C(ctx).Infow("Login function called")

	// 从请求中获取登录请求参数
	var req v1.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		core.WriteResponse(ctx, errno.ErrBind, nil)
		return
	}

	// 验证请求参数
	if _, err := govalidator.ValidateStruct(req); err != nil {
		core.WriteResponse(ctx, errno.ErrInvalidParameter.SetMessage("%s", err.Error()), nil)
		return
	}

	// 调用业务层登录
	resp, err := ctrl.b.AuthBiz().Login(ctx, &req)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	// 返回登录响应
	core.WriteResponse(ctx, nil, resp)
}
