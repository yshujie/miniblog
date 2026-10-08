package catalog

import (
	"github.com/gin-gonic/gin"
	catalogbiz "github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

type Controller struct{ service *catalogbiz.Service }

func New(ds store.IStore) *Controller { return &Controller{service: catalogbiz.New(ds)} }
func (c *Controller) Reorder(ctx *gin.Context) {
	var r v1.ReorderCatalogRequest
	if ctx.ShouldBindJSON(&r) != nil {
		core.WriteResponse(ctx, errno.ErrBind, nil)
		return
	}
	core.WriteResponse(ctx, c.service.Reorder(ctx.Request.Context(), catalogbiz.ReorderInput{Kind: r.Kind, ParentCode: r.ParentCode, Codes: r.Codes}), nil)
}
