package notionsync

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	syncbiz "github.com/yshujie/miniblog/internal/miniblog/notionsync"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
)

// Controller converts HTTP requests only; source writes are confined to bootstrap CLI.
type Controller struct{ service syncbiz.API }

func New(service syncbiz.API) *Controller { return &Controller{service: service} }

func write(c *gin.Context, err error, payload any) {
	if err != nil {
		var appError *syncbiz.Error
		if errors.As(err, &appError) {
			core.WriteResponse(c, &errno.Errno{HTTP: appError.HTTPStatus, Code: appError.Code, Message: appError.Message}, payload)
			return
		}
		var known *errno.Errno
		if !errors.As(err, &known) {
			err = errno.InternalServerError
		} else {
			err = known
		}
	}
	core.WriteResponse(c, err, payload)
}
func decode(c *gin.Context, value any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 128<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil || decoder.Decode(new(any)) != io.EOF {
		core.WriteResponse(c, errno.ErrBind, nil)
		return false
	}
	return true
}
func listQuery(c *gin.Context) (syncbiz.ListQuery, bool) {
	q := syncbiz.ListQuery{Page: 1, Limit: 20}
	if c.ShouldBindQuery(&q) != nil || q.Page < 1 || q.Page > 1000000 || q.Limit < 1 || q.Limit > 100 {
		core.WriteResponse(c, errno.ErrInvalidParameter, nil)
		return q, false
	}
	return q, true
}
func (c *Controller) Status(ctx *gin.Context) {
	out, err := c.service.Status(ctx.Request.Context())
	write(ctx, err, out)
}
func (c *Controller) Sources(ctx *gin.Context) {
	q, ok := listQuery(ctx)
	if !ok {
		return
	}
	out, err := c.service.Sources(ctx.Request.Context(), q)
	write(ctx, err, out)
}
func (c *Controller) Pages(ctx *gin.Context) {
	q := syncbiz.PageQuery{ListQuery: syncbiz.ListQuery{Page: 1, Limit: 20}}
	if ctx.ShouldBindQuery(&q) != nil || q.Page < 1 || q.Page > 1000000 || q.Limit < 1 || q.Limit > 100 {
		core.WriteResponse(ctx, errno.ErrInvalidParameter, nil)
		return
	}
	out, err := c.service.Pages(ctx.Request.Context(), q)
	write(ctx, err, out)
}
func (c *Controller) Runs(ctx *gin.Context) {
	q, ok := listQuery(ctx)
	if !ok {
		return
	}
	out, err := c.service.Runs(ctx.Request.Context(), q)
	write(ctx, err, out)
}
func (c *Controller) Run(ctx *gin.Context) {
	out, err := c.service.Run(ctx.Request.Context(), ctx.Param("run_id"))
	write(ctx, err, out)
}
func (c *Controller) Items(ctx *gin.Context) {
	q, ok := listQuery(ctx)
	if !ok {
		return
	}
	out, err := c.service.Items(ctx.Request.Context(), ctx.Param("run_id"), q)
	write(ctx, err, out)
}
func (c *Controller) UpdateControl(ctx *gin.Context) {
	var input syncbiz.ControlInput
	if !decode(ctx, &input) {
		return
	}
	out, err := c.service.UpdateControl(ctx.Request.Context(), input)
	write(ctx, err, out)
}
func (c *Controller) UpdateSource(ctx *gin.Context) {
	var input syncbiz.SourceInput
	if !decode(ctx, &input) {
		return
	}
	out, err := c.service.UpdateSource(ctx.Request.Context(), ctx.Param("source_id"), input)
	write(ctx, err, out)
}
func (c *Controller) BindCatalog(ctx *gin.Context) {
	var input syncbiz.BindingInput
	if !decode(ctx, &input) {
		return
	}
	input.BindingID = ctx.Param("binding_id")
	out, err := c.service.BindCatalog(ctx.Request.Context(), input)
	write(ctx, err, out)
}
func (c *Controller) Trigger(ctx *gin.Context) {
	var input syncbiz.TriggerInput
	if !decode(ctx, &input) {
		return
	}
	if input.Mode != "dry_run" && input.Mode != "sync" {
		core.WriteResponse(ctx, errno.ErrInvalidParameter, nil)
		return
	}
	out, err := c.service.Trigger(ctx.Request.Context(), input)
	if err != nil {
		write(ctx, err, out)
		return
	}
	ctx.JSON(http.StatusAccepted, core.Response{Code: "ok", Message: "", Payload: out})
}
