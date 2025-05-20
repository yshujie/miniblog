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

	// auth 路由
	g.POST("/register", ac.Register)
	g.POST("/login", ac.Login)
	g.POST("/logout", ac.Logout)

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

		// 创建 blog 路由分组
		blogv1 := v1.Group("/blog")
		{
			blogv1.GET("/moduleDetail", bc.GetModuleDetail)
		}

		// 创建 modules 路由分组
		modulesv1 := v1.Group("/modules")
		{
			modulesv1.POST("", mc.Create)
			modulesv1.GET("", mc.GetAll)
			modulesv1.GET(":code", mc.GetOne)
		}

		// 创建 sections 路由分组
		sectionsv1 := v1.Group("/sections")
		{
			sectionsv1.POST("", sc.Create)
			sectionsv1.GET(":module_code", sc.GetList)
			sectionsv1.GET(":module_code/:code", sc.GetOne)
		}

		// 创建 articles 路由分组
		articlesv1 := v1.Group("/articles")
		{
			articlesv1.POST("", arCtrl.Create)
			articlesv1.GET(":section_code", arCtrl.GetList)
			articlesv1.GET(":section_code/:id", arCtrl.GetOne)
		}
	}

	return nil
}
