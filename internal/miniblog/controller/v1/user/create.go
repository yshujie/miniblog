package user

import (
	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

const defaultMethods = "(GET)|(POST)|(PUT)|(DELETE)"

// Create 创建用户
func (ctrl *UserController) Create(c *gin.Context) {
	log.C(c).Infow("Create user function called")

	// 声明 CreateUserRequest
	var r v1.CreateUserRequest

	// 将请求体中的参数解析到 CreateUserRequest 实例中
	if err := c.ShouldBindJSON(&r); err != nil {
		core.WriteResponse(c, errno.ErrBind, nil)
		return
	}

	// 验证请求参数
	if _, err := govalidator.ValidateStruct(r); err != nil {
		core.WriteResponse(c, errno.ErrInvalidParameter.SetMessage("%s", err.Error()), nil)
		return
	}

	// 调用 Biz 层，创建用户
	if err := ctrl.b.Users().Create(c, &r); err != nil {
		log.C(c).Errorw("create user failed", "error", err, "username", r.Username)
		core.WriteResponse(c, err, nil)
		return
	}

	log.C(c).Infow("create user success", "username", r.Username)
	log.C(c).Infow("start to add named policy", "username", r.Username)

	// 创建用户后，新增用户授权策略
	if _, err := ctrl.a.AddNamedPolicy("p", r.Username, "/v1/users/"+r.Username, defaultMethods); err != nil {
		log.C(c).Errorw("add named policy failed", "error", err, "username", r.Username)
		core.WriteResponse(c, err, nil)
		return
	}

	log.C(c).Infow("add named policy success", "username", r.Username)

	// 返回成功响应
	core.WriteResponse(c, nil, r)
}
