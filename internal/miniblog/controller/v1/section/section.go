package module

import (
	"github.com/yshujie/miniblog/internal/miniblog/biz"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

// SectionController 模块控制器
type SectionController struct {
	biz biz.IBiz
}

// New 简单工厂函数，创建 ModuleController 实例
func New(ds store.IStore) *SectionController {
	log.Infow("... new section controller")
	return &SectionController{
		biz: biz.NewBiz(ds),
	}
}
