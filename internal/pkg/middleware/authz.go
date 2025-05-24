package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/known"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

type Auther interface {
	Authorize(sub, obj, act string) (bool, error)
}

// Authz 是 Gin 中间件，用来进行请求授权.
func Authz(a Auther) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文中获取用户名
		sub := c.GetString(known.XUsernameKey)
		// 获取请求的URL路径
		obj := c.Request.URL.Path
		// 获取请求的方法
		act := c.Request.Method

		log.Debugw("Build authorize context", "sub", sub, "obj", obj, "act", act)
		if allowed, _ := a.Authorize(sub, obj, act); !allowed {
			// 权限不足，暂不做处理
			// core.WriteResponse(c, errno.ErrUnauthorized, nil)
			log.Debugw("Build authorize context", "sub", sub, "obj", obj, "act", act)
			c.Abort()
			return
		}
	}
}
