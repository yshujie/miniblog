package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/marmotedu/component-base/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/known"
	"github.com/yshujie/miniblog/pkg/token"
)

func Authn() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 从请求头中解析 token
		username, err := token.ParseRequest(ctx)
		if err != nil {
			core.WriteResponse(ctx, errno.ErrInvalidToken, nil)
			ctx.Abort()
			return
		}
		ctx.Set(known.XUsernameKey, username)

		ctx.Next()
	}
}
