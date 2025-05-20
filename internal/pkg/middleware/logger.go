package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/known"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 处理请求
		c.Next()

		// 结束时间
		endTime := time.Now()

		// 执行时间
		latencyTime := endTime.Sub(startTime)

		// 请求方式
		reqMethod := c.Request.Method

		// 请求路由
		reqUri := c.Request.RequestURI

		// 状态码
		statusCode := c.Writer.Status()

		// 请求IP
		clientIP := c.ClientIP()

		// 请求ID
		requestID := c.GetString(known.XRequestIDKey)

		// 用户名
		username := c.GetString(known.XUsernameKey)

		// 日志格式
		log.C(c).Infow("HTTP Request",
			"request_id", requestID,
			"status", statusCode,
			"latency", latencyTime,
			"client_ip", clientIP,
			"method", reqMethod,
			"uri", reqUri,
			"username", username,
		)
	}
}
