package middleware

import (
	"encoding/json"
	"fmt"
	"io"
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

		// 请求路由
		reqUri := c.Request.RequestURI

		// 状态码
		statusCode := c.Writer.Status()

		// 请求IP
		clientIP := c.ClientIP()

		// 服务器IP
		serverIP := c.Request.Host

		// 请求ID
		requestID := c.GetString(known.XRequestIDKey)

		// 获取 Referer
		referer := c.Request.Referer()

		// 获取 User-Agent
		userAgent := c.Request.UserAgent()

		// 获取 POST 数据
		var postData string
		if c.Request.Method == "POST" {
			body, err := io.ReadAll(c.Request.Body)
			if err == nil {
				postData = string(body)
			}
		}

		// 获取 Cookies
		cookies, _ := json.Marshal(c.Request.Cookies())

		// 构建日志消息
		msg := fmt.Sprintf("%s [%s][%d] [%.3f ms] [NTC] [-- [server_ip = %s] [client_ip = %s] [theUrl = %s] [referer = %s] [_USER_AGENT = %s] [posts = %s] [cookies = %s] --]",
			startTime.Format("15:04:05.000"),
			requestID,
			statusCode,
			float64(latencyTime.Microseconds())/1000,
			serverIP,
			clientIP,
			reqUri,
			referer,
			userAgent,
			postData,
			string(cookies),
		)

		// 输出日志
		log.C(c).Infow(msg)
	}
}
