package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/yshujie/miniblog/internal/pkg/known"
)

func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 获取请求头中的 X-Request-Id
		requestID := ctx.Request.Header.Get(known.XRequestIDKey)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 将 RequestID 保存到 gin.Context 中
		ctx.Set(known.XRequestIDKey, requestID)

		// 将 RequestID 保存到 HTTP 响应头中
		ctx.Writer.Header().Set(known.XRequestIDKey, requestID)

		ctx.Next()
	}

}
