package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/yshujie/blog-serve/internal/model"
	"github.com/yshujie/blog-serve/internal/service"
	"github.com/yshujie/blog-serve/internal/store/mysql"
	"github.com/yshujie/blog-serve/internal/util"
	"net/http"
)

type ArticleHandler struct {
	articleService service.ArticleService
	logger         *log.Logger
}

func NewArticleHandler(articleService service.ArticleService, logger *log.Logger) *ArticleHandler {
	return &ArticleHandler{
		articleService: articleService,
		logger:         logger,
	}
}

// GetArticleList 获取文章列表
func (h *ArticleHandler) GetArticleList(c *gin.Context) {
	// 1. 解析请求参数
	req := &service.ArticleListRequest{
		Filter: &service.ArticleFilter{
			Keywords:   c.Query("keywords"),
			CategoryID: utils.StringToUint(c.Query("category_id")),
			Tags:       utils.ParseTags(c.Query("tags")),
			Status:     c.Query("status"),
			AuthorID:   utils.StringToUint(c.Query("author_id")),
		},
		Sorting: &service.Sorting{
			Field:     c.DefaultQuery("sort_field", "created_at"),
			Direction: c.DefaultQuery("sort_dir", "desc"),
		},
		Page:     utils.StringToInt(c.DefaultQuery("page", "1")),
		PageSize: utils.StringToInt(c.DefaultQuery("page_size", "10")),
		Format: &service.ListFormatOptions{
			WithContent:   c.DefaultQuery("with_content", "false") == "true",
			ContentLength: utils.StringToInt(c.DefaultQuery("content_length", "200")),
			WithTags:      c.DefaultQuery("with_tags", "true") == "true",
			TimeFormat:    c.DefaultQuery("time_format", "2006-01-02 15:04:05"),
		},
	}

	// 2. 参数验证
	if err := h.validateListRequest(req); err != nil {
		h.handleError(c, err)
		return
	}

	// 3. 调用服务层获取数据
	resp, err := h.articleService.GetArticleList(req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// 4. 返回响应
	h.success(c, resp)
}

// GetArticle 获取文章详情
func (h *ArticleHandler) GetArticle(c *gin.Context) {
	id := utils.StringToUint(c.Param("id"))
	if id == 0 {
		h.handleError(c, errors.New("invalid article id"))
		return
	}

	article, err := h.articleService.GetArticle(id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	h.success(c, article)
}

// CreateArticle 创建文章
func (h *ArticleHandler) CreateArticle(c *gin.Context) {
	var req service.CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, err)
		return
	}

	// 获取当前用户ID（假设已经通过中间件设置）
	userID, exists := c.Get("user_id")
	if !exists {
		h.handleError(c, errors.New("unauthorized"))
		return
	}
	req.AuthorID = userID.(uint)

	article, err := h.articleService.CreateArticle(&req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	h.success(c, article)
}

// UpdateArticle 更新文章
func (h *ArticleHandler) UpdateArticle(c *gin.Context) {
	id := utils.StringToUint(c.Param("id"))
	if id == 0 {
		h.handleError(c, errors.New("invalid article id"))
		return
	}

	var req service.UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, err)
		return
	}

	// 权限检查
	if err := h.checkArticlePermission(c, id); err != nil {
		h.handleError(c, err)
		return
	}

	article, err := h.articleService.UpdateArticle(id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	h.success(c, article)
}

// DeleteArticle 删除文章
func (h *ArticleHandler) DeleteArticle(c *gin.Context) {
	id := utils.StringToUint(c.Param("id"))
	if id == 0 {
		h.handleError(c, errors.New("invalid article id"))
		return
	}

	// 权限检查
	if err := h.checkArticlePermission(c, id); err != nil {
		h.handleError(c, err)
		return
	}

	if err := h.articleService.DeleteArticle(id); err != nil {
		h.handleError(c, err)
		return
	}

	h.success(c, gin.H{"message": "article deleted successfully"})
}

// UpdateArticleStatus 更新文章状态
func (h *ArticleHandler) UpdateArticleStatus(c *gin.Context) {
	id := utils.StringToUint(c.Param("id"))
	if id == 0 {
		h.handleError(c, errors.New("invalid article id"))
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=draft published archived"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, err)
		return
	}

	if err := h.articleService.UpdateArticleStatus(id, req.Status); err != nil {
		h.handleError(c, err)
		return
	}

	h.success(c, gin.H{"message": "status updated successfully"})
}

// 辅助方法

// validateListRequest 验证列表请求参数
func (h *ArticleHandler) validateListRequest(req *service.ArticleListRequest) error {
	if req.Page < 1 {
		return errors.New("invalid page number")
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		return errors.New("invalid page size")
	}
	return nil
}

// checkArticlePermission 检查文章操作权限
func (h *ArticleHandler) checkArticlePermission(c *gin.Context, articleID uint) error {
	userID, exists := c.Get("user_id")
	if !exists {
		return errors.New("unauthorized")
	}

	// 检查是否是文章作者或管理员
	isAdmin, _ := c.Get("is_admin")
	if isAdmin.(bool) {
		return nil
	}

	article, err := h.articleService.GetArticle(articleID)
	if err != nil {
		return err
	}

	if article.AuthorID != userID.(uint) {
		return errors.New("permission denied")
	}

	return nil
}

// handleError 统一错误处理
func (h *ArticleHandler) handleError(c *gin.Context, err error) {
	h.logger.Error("handler error", "error", err)

	var statusCode int
	var errorMessage string

	switch {
	case errors.Is(err, service.ErrNotFound):
		statusCode = http.StatusNotFound
		errorMessage = "Article not found"
	case errors.Is(err, service.ErrInvalidInput):
		statusCode = http.StatusBadRequest
		errorMessage = err.Error()
	case errors.Is(err, service.ErrUnauthorized):
		statusCode = http.StatusUnauthorized
		errorMessage = "Unauthorized"
	case errors.Is(err, service.ErrPermissionDenied):
		statusCode = http.StatusForbidden
		errorMessage = "Permission denied"
	default:
		statusCode = http.StatusInternalServerError
		errorMessage = "Internal server error"
	}

	c.JSON(statusCode, gin.H{
		"error": errorMessage,
	})
}

// success 统一成功响应
func (h *ArticleHandler) success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"data": data,
	})
}
