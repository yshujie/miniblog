package user

import (
	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/yshujie/miniblog/internal/miniblog/biz"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// UserController 用户控制器
type UserController struct {
	b biz.IBiz
}

// New 简单工厂函数，创建 UserController 实例
func New(ds store.IStore) *UserController {
	return &UserController{
		b: biz.NewBiz(ds),
	}
}

// Create 创建用户
func (uc *UserController) Create(c *gin.Context) {
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
	if err := uc.b.Users().Create(c, &r); err != nil {
		core.WriteResponse(c, err, nil)
		return
	}

	// 返回成功响应
	core.WriteResponse(c, nil, nil)
}
