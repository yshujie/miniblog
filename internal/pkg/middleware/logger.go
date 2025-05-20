package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/known"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

// bodyLogWriter 是一个自定义的响应写入器，用于捕获响应体
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 读取请求体
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			// 重新设置请求体，因为读取后需要重置
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 创建自定义响应写入器
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

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

		// 获取 Cookies
		cookies, _ := json.Marshal(c.Request.Cookies())

		// 构建请求日志消息
		requestMsg := fmt.Sprintf("%s [%s][%d] [%.3f ms] [NTC] [-- [server_ip = %s] [client_ip = %s] [theUrl = %s] [referer = %s] [_USER_AGENT = %s] [request_body = %s] [cookies = %s] --]",
			startTime.Format("15:04:05.000"),
			requestID,
			statusCode,
			float64(latencyTime.Microseconds())/1000,
			serverIP,
			clientIP,
			reqUri,
			referer,
			userAgent,
			string(requestBody),
			string(cookies),
		)

		// 构建响应日志消息
		responseMsg := fmt.Sprintf("%s [%s][%d] [%.3f ms] [NTC] [-- [response_body = %s] --]",
			endTime.Format("15:04:05.000"),
			requestID,
			statusCode,
			float64(latencyTime.Microseconds())/1000,
			blw.body.String(),
		)

		// 输出请求日志
		log.C(c).Infow("HTTP Request", "request", requestMsg)

		// 输出响应日志
		log.C(c).Infow("HTTP Response", "response", responseMsg)
	}
}
