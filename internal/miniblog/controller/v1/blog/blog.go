package blog

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/yshujie/miniblog/internal/miniblog/biz"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// BlogController 博客控制器
type BlogController struct {
	biz biz.IBiz
}

// New 简单工厂函数，创建 BlogController 实例
func New(ds store.IStore) *BlogController {
	log.Infow("... new blog controller")
	return &BlogController{
		biz: biz.NewBiz(ds),
	}
}

// GetModuleDetail 获取模块详情
func (c *BlogController) GetModuleDetail(ctx *gin.Context) {
	log.C(ctx).Infow("Get module detail function called")

	// 获取请求数据
	req := &v1.GetModuleDetailRequest{}
	if err := ctx.ShouldBindQuery(req); err != nil {
		core.WriteResponse(ctx, errno.ErrBind, nil)
		return
	}

	// 验证请求数据
	if err := validator.New().Struct(req); err != nil {
		core.WriteResponse(ctx, errno.ErrInvalidParameter, nil)
		return
	}

	log.C(ctx).Infow("Get module detail function called", "req", req)

	// 调用 Biz 层处理业务
	moduleDetailResp, err := c.biz.BlogBiz().GetModuleDetail(req)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, moduleDetailResp)
}

// GetArticleDetail 获取文章详情
func (c *BlogController) GetArticleDetail(ctx *gin.Context) {
	log.C(ctx).Infow("Get article detail function called")

	// 获取请求数据
	req := &v1.GetArticleDetailRequest{}
	if err := ctx.ShouldBindQuery(req); err != nil {
		core.WriteResponse(ctx, errno.ErrBind, nil)
		return
	}

	// 验证请求数据
	if err := validator.New().Struct(req); err != nil {
		core.WriteResponse(ctx, errno.ErrInvalidParameter, nil)
		return
	}

	// 调用 Biz 层处理业务
	articleDetailResp, err := c.biz.BlogBiz().GetArticleDetail(req)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, articleDetailResp)
}
