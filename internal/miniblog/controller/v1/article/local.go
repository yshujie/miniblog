package article

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	articlebiz "github.com/yshujie/miniblog/internal/miniblog/biz/article"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
)

func bindLocalCommand(ctx *gin.Context, input any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	if decoder.Decode(input) != nil || decoder.Decode(new(any)) != io.EOF {
		core.WriteResponse(ctx, errno.ErrBind, nil)
		return false
	}
	return true
}
func (c *ArticleController) PatchLocal(ctx *gin.Context) {
	id, ok := commandID(ctx)
	if !ok {
		return
	}
	var request struct {
		Author *string `json:"author"`
	}
	if !bindLocalCommand(ctx, &request) {
		return
	}
	if request.Author == nil {
		core.WriteResponse(ctx, errno.ErrInvalidParameter, nil)
		return
	}
	out, err := c.biz.ArticleBiz().PatchLocal(ctx.Request.Context(), id, articlebiz.LocalPatchInput{Author: *request.Author})
	core.WriteResponse(ctx, err, out)
}
func (c *ArticleController) PublicationHold(ctx *gin.Context) {
	id, ok := commandID(ctx)
	if !ok {
		return
	}
	var request struct {
		Held   *bool  `json:"held"`
		Reason string `json:"reason"`
	}
	if !bindLocalCommand(ctx, &request) {
		return
	}
	if request.Held == nil {
		core.WriteResponse(ctx, errno.ErrInvalidParameter, nil)
		return
	}
	out, err := c.biz.ArticleBiz().SetPublicationHold(ctx.Request.Context(), id, articlebiz.HoldInput{Held: *request.Held, Reason: request.Reason})
	core.WriteResponse(ctx, err, out)
}
