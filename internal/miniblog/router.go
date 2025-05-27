package miniblog

import (
	"github.com/gin-gonic/gin"
	articleCtrl "github.com/yshujie/miniblog/internal/miniblog/controller/v1/article"
	authCtrl "github.com/yshujie/miniblog/internal/miniblog/controller/v1/auth"
	blogCtrl "github.com/yshujie/miniblog/internal/miniblog/controller/v1/blog"
	moduleCtrl "github.com/yshujie/miniblog/internal/miniblog/controller/v1/module"
	sectionCtrl "github.com/yshujie/miniblog/internal/miniblog/controller/v1/section"
	userCtrl "github.com/yshujie/miniblog/internal/miniblog/controller/v1/user"
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

	// 创建 controllers
	ac := authCtrl.New(store.S)
	bc := blogCtrl.New(store.S)
	uc := userCtrl.New(store.S, authz)
	mc := moduleCtrl.New(store.S)
	sc := sectionCtrl.New(store.S)
	arCtrl := articleCtrl.New(store.S)

	// 创建 v1 路由组
	v1 := g.Group("/v1")
	{
		// 创建 auth 路由分组
		authv1 := v1.Group("/auth")
		{
			authv1.POST("/login", ac.Login)
			authv1.POST("/logout", ac.Logout)
			authv1.POST("/register", ac.Register)
		}

		// 创建 blog 路由分组
		blogv1 := v1.Group("/blog")
		{
			blogv1.GET("/modules", bc.GetModuleList)
			blogv1.GET("/moduleDetail", bc.GetModuleDetail)
			blogv1.GET("/articleDetail", bc.GetArticleDetail)
		}

		adminv1 := v1.Group("/admin")
		{
			// 修改用户密码
			adminv1.GET("/users/:name/change-password", uc.ChangePassword)

			// 使用 Authn 和 Authz 中间件
			adminv1.Use(mw.Authn(), mw.Authz(authz))

			// users 路由分组
			userv1 := adminv1.Group("/users")
			{
				userv1.POST("", uc.Create)          // 创建用户
				userv1.GET(":name", uc.Get)         // 获取用户信息
				userv1.GET("/myinfo", uc.GetMyInfo) // 获取当前用户信息
			}

			// modules 路由分组
			modulesv1 := adminv1.Group("/modules")
			{
				modulesv1.GET("", mc.GetAll)      // 获取所有模块
				modulesv1.POST("", mc.Create)     // 创建模块
				modulesv1.GET(":code", mc.GetOne) // 获取模块信息
			}

			// sections 路由分组
			sectionsv1 := adminv1.Group("/sections")
			{
				sectionsv1.POST("", sc.Create)                  // 创建章节
				sectionsv1.GET(":module_code", sc.GetList)      // 获取章节列表
				sectionsv1.GET(":module_code/:code", sc.GetOne) // 获取章节信息
			}

			// articles 路由分组
			articlesv1 := adminv1.Group("/articles")
			{
				articlesv1.POST("", arCtrl.Create)                 // 创建文章
				articlesv1.GET("", arCtrl.GetList)                 // 获取文章列表
				articlesv1.GET("/:id", arCtrl.GetOne)              // 获取文章信息
				articlesv1.PUT("/:id", arCtrl.Update)              // 更新文章
				articlesv1.PUT("/:id/publish", arCtrl.Publish)     // 发布文章
				articlesv1.PUT("/:id/unpublish", arCtrl.Unpublish) // 下架文章
			}
		}
	}

	return nil
}
