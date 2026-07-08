package subsection

import (
	"github.com/yshujie/miniblog/internal/miniblog/biz"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

type SubsectionController struct {
	biz biz.IBiz
}

func New(ds store.IStore) *SubsectionController {
	log.Infow("... new subsection controller")
	return &SubsectionController{
		biz: biz.NewBiz(ds),
	}
}
