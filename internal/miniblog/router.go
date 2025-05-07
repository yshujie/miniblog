package miniblog

import (
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/miniblog/controller/v1/user"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/log"
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

	// 创建 v1 路由组
	v1 := g.Group("/v1")
	{
		// 创建 users 路由分组
		userv1 := v1.Group("/users")
		{
			userv1.POST("", user.New(store.S).Create)
		}

	}

	return nil
}
