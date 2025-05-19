package module

import (
	"github.com/yshujie/miniblog/internal/miniblog/biz"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

// ModuleController 模块控制器
type ModuleController struct {
	biz biz.IBiz
}

// New 简单工厂函数，创建 ModuleController 实例
func New(ds store.IStore) *ModuleController {
	log.Infow("... new module controller")
	return &ModuleController{
		biz: biz.NewBiz(ds),
	}
}
