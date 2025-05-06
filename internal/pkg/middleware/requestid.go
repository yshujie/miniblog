package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 获取请求头中的 X-Request-Id
		requestID := ctx.Request.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 将 RequestID 保存到 gin.Context 中
		ctx.Set("X-Request-Id", requestID)

		// 将 RequestID 保存到 HTTP 响应头中
		ctx.Writer.Header().Set("X-Request-Id", requestID)

		ctx.Next()
	}

}
