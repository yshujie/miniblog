package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// NoCache 禁止缓存中间件
func NoCache(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate, value")
	ctx.Header("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
	ctx.Header("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	ctx.Next()
}

// Secure 安全中间件
func Secure(ctx *gin.Context) {
	ctx.Header("X-Frame-Options", "ALLOWALL")
	ctx.Header("Content-Security-Policy", "frame-ancestors *")
	ctx.Header("X-Content-Type-Options", "nosniff")
	ctx.Header("X-XSS-Protection", "1; mode=block")
	if ctx.Request.TLS != nil {
		ctx.Header("Strict-Transport-Security", "max-age=31536000")
	}
}
