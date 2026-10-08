package article

import (
	"errors"
	"fmt"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
	"gorm.io/gorm"
)

// Update 更新文章
func (c *ArticleController) Update(ctx *gin.Context) {
	log.C(ctx).Infow("Update article function called")

	request := &v1.UpdateArticleRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		log.C(ctx).Errorw("failed to bind request", "error", err)
		core.WriteResponse(ctx, errno.ErrBind, nil)
		return
	}
	id, ok := commandID(ctx)
	if !ok {
		return
	}
	pathID := fmt.Sprint(id)
	if request.ID != "" && request.ID != pathID {
		core.WriteResponse(ctx, errno.ErrInvalidParameter, nil)
		return
	}
	request.ID = pathID
	// Metadata edits keep working, but introducing/changing a source identity
	// follows the same migration gate as registration.
	current, err := store.NewStore(c.ds.DB().WithContext(ctx.Request.Context())).Articles().GetOne(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errno.ErrArticleNotFound
		}
		core.WriteResponse(ctx, err, nil)
		return
	}
	binding, bindingErr := store.PageBindingByArticle(store.NewStore(c.ds.DB().WithContext(ctx.Request.Context())), id)
	if bindingErr != nil {
		core.WriteResponse(ctx, errno.InternalServerError, nil)
		return
	}
	managed := binding != nil && binding.ManagementState == "managed"
	if request.ExternalLink != "" {
		identity, parseErr := source.Parse(request.ExternalLink)
		if parseErr != nil {
			core.WriteResponse(ctx, errno.ErrInvalidParameter, nil)
			return
		}
		owner, lookupErr := store.FindSourceOwner(store.NewStore(c.ds.DB().WithContext(ctx.Request.Context())), identity.SourceKey)
		if lookupErr != nil {
			core.WriteResponse(ctx, errno.InternalServerError, nil)
			return
		}
		if !managed && (current.SourceKey == nil || *current.SourceKey != identity.SourceKey) && (owner == nil || owner.ID != id) {
			if !c.registrationAllowed(ctx) {
				return
			}
		}
	}

	if _, err := govalidator.ValidateStruct(request); err != nil {
		log.C(ctx).Errorw("invalid request parameters", "error", err)
		core.WriteResponse(ctx, &errno.Errno{HTTP: 400, Code: errno.ErrInvalidParameter.Code, Message: err.Error()}, nil)
		return
	}

	// 更新文章
	response, err := c.biz.ArticleBiz().Update(ctx.Request.Context(), request)
	if err != nil {
		log.C(ctx).Errorw("update article failed", "error", err, fmt.Sprintf("%T", err))
		var typed *errno.Errno
		if errors.As(err, &typed) && typed.HTTP == 409 {
			latest, readErr := c.biz.ArticleBiz().GetOne(ctx.Request.Context(), id)
			if readErr == nil {
				core.WriteResponse(ctx, err, latest)
				return
			}
		}
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, response)
}
