package http

import (
	"github.com/gin-gonic/gin"
	"github.com/yshujie/blog-serve/api/v1/admin"
	"github.com/yshujie/blog-serve/api/v1/common"
	"github.com/yshujie/blog-serve/api/v1/public"
)

func Start(ip *string, port *int) {
	// 初始化 router
	router := gin.Default()

	gin.SetMode(gin.ReleaseMode)

	// // 设置跨域
	// router.Use(middleware.Cors())

	// // 加载 public 路由
	initPublicRouter(router)

	// // 加载 admin 路由
	// initAdminRouter(router)

	// // 加载 common 路由
	// initCommonRouter(router)

	// // 启动 router
	// router.Run(fmt.Sprintf("%s:%d", *ip, *port))
}

func initPublicRouter(router *gin.Engine) {
	publicRouter := router.Group("api/v1/public")
	{
		publicRouter.GET("article/list", public.GetArticleList)
	}
}

func initAdminRouter(router *gin.Engine) {
	adminRouter := router.Group("api/v1/admin")
	{
		adminRouter.POST("article/add", admin.AddArticle)
	}
}

// 加载 common 路由

func initCommonRouter(router *gin.Engine) {
	commonRouter := router.Group("api/v1/common")
	{
		commonRouter.POST("login", common.Login())
	}
}
