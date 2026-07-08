package subsection

import (
	"fmt"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

func (c *SubsectionController) Create(ctx *gin.Context) {
	log.C(ctx).Infow("Create subsection function called")

	request := &v1.CreateSubsectionRequest{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		log.C(ctx).Errorw("failed to bind request", "error", err)
		core.WriteResponse(ctx, err, nil)
		return
	}

	if _, err := govalidator.ValidateStruct(request); err != nil {
		log.C(ctx).Errorw("invalid request parameters", "error", err)
		core.WriteResponse(ctx, errno.ErrInvalidParameter.SetMessage("%s", err.Error()), nil)
		return
	}

	response, err := c.biz.SubsectionBiz().Create(ctx, request)
	if err != nil {
		log.C(ctx).Errorw("create subsection failed", "error", err, "code", request.Code, "error_type", fmt.Sprintf("%T", err))
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, response)
}
