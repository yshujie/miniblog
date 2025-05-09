package miniblog

import (
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/miniblog/controller/v1/user"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/log"
	mw "github.com/yshujie/miniblog/internal/pkg/middleware"
	"github.com/yshujie/miniblog/pkg/auth"
)

// installRouters 安装 miniblog 的路由
func installRouters(g *gin.Engine) error {
	// 注册 404 Handler
	g.NoRoute(func(ctx *gin.Context) {
		core.WriteResponse(ctx, errno.ErrPageNotFound, nil)
	})

	// 注册 /health 路由
	g.GET("/health", func(ctx *gin.Context) {
		log.C(ctx).Infow("health check")

		core.WriteResponse(ctx, nil, map[string]string{"status": "ok"})
	})

	// 创建 authz
	authz, err := auth.NewAuthz(store.S.DB())
	if err != nil {
		return err
	}

	// 创建 user controller
	uc := user.New(store.S, authz)

	// 登录
	g.POST("/login", uc.Login)

	// 创建 v1 路由组
	v1 := g.Group("/v1")
	{
		// 创建 users 路由分组
		userv1 := v1.Group("/users")
		{
			userv1.POST("", uc.Create)
			userv1.PUT(":name/change-password", uc.ChangePassword)
			userv1.Use(mw.Authn(), mw.Authz(authz))
			userv1.GET(":name", uc.Get)
		}

	}

	return nil
}
