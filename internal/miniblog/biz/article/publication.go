package article

import (
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"strings"
)

// Validate state commands in one place; status zero denotes a newly imported record.
func validatePublication(a *model.Article, p *catalog.Placement, status int) error {
	switch status {
	case model.ArticleStatusPublished:
		if a.Status == model.ArticleStatusDeleted {
			return conflict("归档文章须先恢复")
		}
		if strings.TrimSpace(a.ExternalLink) != "" {
			if _, err := source.Parse(a.ExternalLink); err != nil {
				return invalid(err.Error())
			}
		}
		if err := validateFields(a.Title, a.Author, strings.Split(a.Tags, ",")); err != nil {
			return err
		}
		return catalog.RequireVisible(p)
	case model.ArticleStatusUnpublished:
		if a.Status != 0 && a.Status != model.ArticleStatusPublished && a.Status != model.ArticleStatusUnpublished {
			return conflict("只能下架已发布文章")
		}
	case model.ArticleStatusDraft:
		if a.Status != 0 && a.Status != model.ArticleStatusDraft && a.Status != model.ArticleStatusDeleted {
			return conflict("请通过恢复命令恢复归档文章")
		}
	case model.ArticleStatusDeleted:
	default:
		return invalid("文章状态无效")
	}
	return nil
}
