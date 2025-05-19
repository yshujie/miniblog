package module

import (
	"github.com/yshujie/miniblog/internal/miniblog/biz"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/log"
)

// ArticleController 文章控制器
type ArticleController struct {
	biz biz.IBiz
}

// New 简单工厂函数，创建 ArticleController 实例
func New(ds store.IStore) *ArticleController {
	log.Infow("... new article controller")
	return &ArticleController{
		biz: biz.NewBiz(ds),
	}
}
