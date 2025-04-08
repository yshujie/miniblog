package public

import (
	"github.com/gin-gonic/gin"
)

// GetArticleList 获取文章列表
func GetArticleList(c *gin.Context) {

	// 返回 hello world
	c.JSON(200, gin.H{
		"message": "hello world",
	})
}
