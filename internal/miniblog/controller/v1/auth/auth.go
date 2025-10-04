package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/miniblog/biz"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// AuthController 认证控制器接口
type AuthController interface {
	Register(c *gin.Context)
	Login(c *gin.Context)
	Logout(c *gin.Context)
}

// authController 认证控制器实现
type authController struct {
	b biz.IBiz
}

// New 创建 AuthController 实例
func New(ds store.IStore) *authController {
	return &authController{biz.NewBiz(ds)}
}

// Register 注册
func (c *authController) Register(ctx *gin.Context) {

}

// Logout 登出
func (c *authController) Logout(ctx *gin.Context) {
	var req v1.LogoutRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		core.WriteResponse(ctx, errno.ErrBind, nil)
		return
	}

	// 调用业务层登出
	err := c.b.AuthBiz().Logout(ctx, &req)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	// 返回登出响应
	core.WriteResponse(ctx, nil, &v1.LogoutResponse{
		Message: "退出登录成功",
	})
}
