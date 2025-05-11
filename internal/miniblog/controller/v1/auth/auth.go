package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/miniblog/biz"
	"github.com/yshujie/miniblog/internal/miniblog/store"
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

}
