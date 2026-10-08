package article

import (
	"context"
	"errors"
	"github.com/go-sql-driver/mysql"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type IArticleBiz interface {
	PatchLocal(context.Context, uint64, LocalPatchInput) (*v1.ArticleInfoResponse, error)
	SetPublicationHold(context.Context, uint64, HoldInput) (*v1.ArticleInfoResponse, error)
	Create(context.Context, *v1.CreateArticleRequest) (*v1.ArticleInfoResponse, error)
	Update(context.Context, *v1.UpdateArticleRequest) (*v1.ArticleInfoResponse, error)
	Publish(context.Context, uint64) error
	Unpublish(context.Context, uint64) error
	GetList(context.Context, *v1.ArticleListRequest) (*v1.GetArticleListResponse, error)
	GetOne(context.Context, uint64) (*v1.GetArticleResponse, error)
	Preview(context.Context, *v1.PreviewSourceRequest) (*v1.PreviewSourceResponse, error)
	Register(context.Context, *v1.RegisterArticleRequest) (*v1.RegisterArticleResponse, error)
	Move(context.Context, uint64, *v1.MoveArticleRequest) (*v1.ArticleCommandResponse, error)
	Archive(context.Context, uint64) (*v1.ArticleCommandResponse, error)
	Restore(context.Context, uint64) (*v1.ArticleCommandResponse, error)
	Reorder(context.Context, *v1.ReorderArticlesRequest) error
}
type articleBiz struct {
	ds            store.IStore
	notion        source.NotionClient
	defaultAuthor string
}

func New(ds store.IStore) *articleBiz { return NewWithNotionClient(ds, source.FromEnvironment()) }
func NewWithNotionClient(ds store.IStore, client source.NotionClient) *articleBiz {
	return &articleBiz{ds: ds, notion: client}
}

// NewForSync configures the create-only local author default.
func NewForSync(ds store.IStore, defaultAuthor string) *articleBiz {
	return &articleBiz{ds: ds, defaultAuthor: defaultAuthor}
}
func invalid(message string) error {
	return &errno.Errno{HTTP: 400, Code: errno.ErrInvalidParameter.Code, Message: message}
}
func conflict(message string) error {
	return &errno.Errno{HTTP: 409, Code: "ArticleConflict", Message: message}
}
func ParseID(value string) (uint64, error) {
	id, e := strconv.ParseUint(value, 10, 63)
	if e != nil || id == 0 {
		return 0, invalid("文章ID无效")
	}
	return id, nil
}
func (b *articleBiz) contextStore(ctx context.Context) store.IStore {
	if b.ds.DB() == nil {
		return b.ds
	}
	return store.NewStore(b.ds.DB().WithContext(ctx))
}
func validateFields(title, author string, tags []string) error {
	if strings.TrimSpace(title) == "" || utf8.RuneCountInString(title) > 255 {
		return invalid("标题须为1-255字")
	}
	if utf8.RuneCountInString(author) > 128 {
		return invalid("作者或标签过长")
	}
	if _, err := model.EncodeArticleTags(tags); err != nil {
		return invalid(err.Error())
	}
	return nil
}
func bindIdentity(a *model.Article, identity source.Identity) {
	a.Provider = &identity.Provider
	a.CanonicalURL = &identity.CanonicalURL
	a.SourceKey = &identity.SourceKey
}
func findSource(ds store.IStore, key string) (*model.Article, error) {
	a, err := store.FindSourceOwner(ds, key)
	if errors.Is(err, store.ErrSourceManagedPending) {
		return nil, sourceManaged()
	}
	if errors.Is(err, store.ErrSourceIdentityConflict) {
		return nil, conflict("来源身份存在冲突，请先审核")
	}
	return a, err
}

func sourceUniqueError(err error) bool {
	var my *mysql.MySQLError
	if errors.As(err, &my) {
		return my.Number == 1062 && strings.Contains(my.Message, "uq_article_source_key")
	}
	return errors.Is(err, gorm.ErrDuplicatedKey) || (err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: article.source_key"))
}
func nextPos(ds store.IStore, section, sub string) (int, error) {
	q := ds.DB().Model(&model.Article{}).Where("section_code = ?", section)
	if sub == "" {
		q = q.Where("subsection_code = '' OR subsection_code IS NULL")
	} else {
		q = q.Where("subsection_code = ?", sub)
	}
	var max *int
	if e := q.Select("MAX(pos)").Scan(&max).Error; e != nil {
		return 0, e
	}
	if max == nil {
		return 1, nil
	}
	return *max + 1, nil
}
func canonicalPlacement(a *model.Article, p *catalog.Placement) {
	a.SectionCode = p.Section.Code
	a.SubsectionCode = ""
	if p.Subsection != nil {
		a.SubsectionCode = p.Subsection.Code
	}
}
func save(ds store.IStore, a *model.Article) error {
	if ds.DB() == nil {
		return ds.Articles().Update(a)
	}
	a.UpdatedAt = time.Now()
	err := ds.DB().Model(&model.Article{}).Where("id = ?", a.ID).Updates(map[string]interface{}{
		"title": a.Title, "content": a.Content, "external_link": a.ExternalLink, "section_code": a.SectionCode, "subsection_code": a.SubsectionCode,
		"author": a.Author, "tags": a.Tags, "tags_json": a.TagsJSON, "pos": a.Pos, "status": a.Status, "provider": a.Provider, "canonical_url": a.CanonicalURL, "source_key": a.SourceKey, "updated_at": a.UpdatedAt}).Error
	if sourceUniqueError(err) {
		return conflict("该外部文档已被收录")
	}
	return err
}
func (b *articleBiz) withLocked(ctx context.Context, id uint64, targetSection, targetSub string, fn func(store.IStore, *model.Article, *catalog.Placement) error) error {
	return store.InTransaction(ctx, b.ds, func(ds store.IStore) error {
		if e := store.LockSourceControl(ds, false); e != nil {
			return e
		}
		return withLockedOnStore(ds, id, targetSection, targetSub, fn)
	})
}

// withLockedOnStore borrows the caller's transaction and never starts or completes it.
func withLockedOnStore(ds store.IStore, id uint64, targetSection, targetSub string, fn func(store.IStore, *model.Article, *catalog.Placement) error) error {
	a, e := ds.Articles().GetOne(id)
	if e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return errno.ErrArticleNotFound
		}
		return e
	}
	old, e := catalog.Resolve(ds, a.SectionCode, a.SubsectionCode)
	if e != nil {
		return e
	}
	target := old
	if targetSection != "" {
		target, e = catalog.Resolve(ds, targetSection, targetSub)
		if e != nil {
			return e
		}
	}
	if e = catalog.LockModules(ds, old.Module.Code, target.Module.Code); e != nil {
		return e
	}
	if ds.DB() != nil {
		if e = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&a).Error; e != nil {
			return e
		}
	}
	current, e := catalog.Resolve(ds, a.SectionCode, a.SubsectionCode)
	if e != nil {
		return e
	}
	if current.Module.ID != old.Module.ID {
		return conflict("文章位置已变更，请刷新后重试")
	}
	target = current
	if targetSection != "" {
		target, e = catalog.Resolve(ds, targetSection, targetSub)
		if e != nil {
			return e
		}
	}
	return fn(ds, a, target)
}

func (b *articleBiz) Preview(ctx context.Context, r *v1.PreviewSourceRequest) (*v1.PreviewSourceResponse, error) {
	identity, e := source.Parse(r.ExternalLink)
	if e != nil {
		return nil, invalid(e.Error())
	}
	out := &v1.PreviewSourceResponse{Provider: identity.Provider, CanonicalURL: identity.CanonicalURL, MetadataStatus: "manual_required"}
	if b.ds.DB() != nil {
		existing, e := findSource(b.contextStore(ctx), identity.SourceKey)
		if e != nil {
			return nil, e
		}
		if existing != nil {
			out.ExistingArticle, e = b.commandDTO(ctx, existing)
			if e != nil {
				return nil, e
			}
		}
	}
	if identity.Provider != "notion" {
		out.Reason = "manual_title"
		return out, nil
	}
	if identity.PageID == "" {
		out.Reason = "notion_page_id_missing"
		return out, nil
	}
	if b.notion == nil {
		out.Reason = "notion_not_configured"
		return out, nil
	}
	title, e := b.notion.GetTitle(ctx, identity.PageID)
	if e != nil {
		out.Reason = "notion_unavailable"
		var metadata *source.MetadataError
		if errors.As(e, &metadata) {
			out.Reason = metadata.Reason
		}
		return out, nil
	}
	if strings.TrimSpace(title) == "" {
		out.Reason = "notion_title_missing"
		return out, nil
	}
	out.Title = title
	out.MetadataStatus = "resolved"
	return out, nil
}
func (b *articleBiz) RegisterSource(ctx context.Context, r RegisterInput) (*v1.RegisterArticleResponse, error) {
	identity, e := source.Parse(r.ExternalLink)
	if e != nil {
		return nil, invalid(e.Error())
	}
	if b.ds.DB() == nil {
		return nil, invalid("登记需要数据库")
	}
	// Existing records remain unchanged, including archived records.
	existing, e := findSource(b.contextStore(ctx), identity.SourceKey)
	if e != nil {
		return nil, e
	}
	if existing != nil {
		dto, e := b.commandDTO(ctx, existing)
		return &v1.RegisterArticleResponse{Outcome: "already_registered", Article: dto}, e
	}
	if e = validateFields(r.Title, r.Author, r.Tags); e != nil {
		return nil, e
	}
	var a *model.Article
	outcome := "created"
	e = store.InTransaction(ctx, b.ds, func(ds store.IStore) error {
		if e := store.LockSourceControl(ds, true); e != nil {
			return e
		}
		if e := store.CheckSourceWrites(ds); e != nil {
			return e
		}
		if e := checkPendingSourceOwner(ds, identity); e != nil {
			return e
		}
		p, e := catalog.Resolve(ds, r.SectionCode, r.SubsectionCode)
		if e != nil {
			return e
		}
		if e = catalog.LockModules(ds, p.Module.Code); e != nil {
			return e
		}
		p, e = catalog.Resolve(ds, r.SectionCode, r.SubsectionCode)
		if e != nil {
			return e
		}
		if r.Publish {
			if e = catalog.RequireVisible(p); e != nil {
				return e
			}
		}
		a, e = findSource(ds, identity.SourceKey)
		if e != nil {
			return e
		}
		if a != nil {
			outcome = "already_registered"
			return nil
		}
		a = &model.Article{Title: strings.TrimSpace(r.Title), ExternalLink: strings.TrimSpace(r.ExternalLink), Author: r.Author, Status: model.ArticleStatusDraft}
		if e = model.SetArticleTags(a, r.Tags); e != nil {
			return e
		}
		canonicalPlacement(a, p)
		bindIdentity(a, identity)
		a.Pos, e = nextPos(ds, a.SectionCode, a.SubsectionCode)
		if e != nil {
			return e
		}
		if r.Publish {
			a.Publish()
		}
		return ds.Articles().Create(a)
	})
	if e != nil {
		// An insert racing in a different module is resolved by the unique source key.
		if !sourceUniqueError(e) {
			return nil, e
		}
		existing, lookup := findSource(b.contextStore(ctx), identity.SourceKey)
		if lookup != nil || existing == nil {
			return nil, e
		}
		a = existing
		outcome = "already_registered"
	}
	dto, e := b.commandDTO(ctx, a)
	if e != nil {
		return nil, e
	}
	return &v1.RegisterArticleResponse{Outcome: outcome, Article: dto}, nil
}
func (b *articleBiz) Create(ctx context.Context, r *v1.CreateArticleRequest) (*v1.ArticleInfoResponse, error) {
	p, e := catalog.Resolve(b.contextStore(ctx), r.SectionCode, r.SubsectionCode)
	if e != nil {
		return nil, e
	}
	if r.ModuleCode != "" {
		m, e := b.contextStore(ctx).Modules().GetByCode(r.ModuleCode)
		if e != nil {
			return nil, e
		}
		if m == nil || m.ID != p.Module.ID {
			return nil, invalid("章节不属于当前模块")
		}
	}
	response, e := b.Register(ctx, &v1.RegisterArticleRequest{ExternalLink: r.ExternalLink, Title: r.Title, SectionCode: r.SectionCode, SubsectionCode: r.SubsectionCode, Author: r.Author, Tags: r.Tags})
	if e != nil {
		return nil, e
	}
	return &v1.ArticleInfoResponse{Article: response.Article.ArticleInfo}, nil
}
func (b *articleBiz) Edit(ctx context.Context, r UpdateInput) (*v1.ArticleInfoResponse, error) {
	id := r.ID
	if id == 0 || id > math.MaxInt64 {
		return nil, invalid("文章ID无效")
	}
	if e := validateFields(r.Title, r.Author, r.Tags); e != nil {
		return nil, e
	}
	var identity *source.Identity
	if strings.TrimSpace(r.ExternalLink) != "" {
		parsed, err := source.Parse(r.ExternalLink)
		if err != nil {
			return nil, invalid(err.Error())
		}
		identity = &parsed
	}
	var result *model.Article
	var e error
	e = b.withLocked(ctx, id, r.SectionCode, r.SubsectionCode, func(ds store.IStore, a *model.Article, p *catalog.Placement) error {
		binding, bindErr := store.PageBindingByArticle(ds, a.ID)
		if bindErr != nil {
			return bindErr
		}
		if binding != nil && binding.ManagementState == model.NotionManagementManaged {
			same, err := editIsNoop(ds, a, p, r)
			if err != nil {
				return err
			}
			if same {
				result = a
				return nil
			}
			return sourceManaged()
		}
		if binding != nil && r.ExternalLink != a.ExternalLink {
			return conflict("历史外链不可通过编辑更换，请使用最新阅读链接")
		}
		if e := checkManualWritable(ds, a); e != nil {
			return e
		}
		if a.Status == model.ArticleStatusDeleted {
			return conflict("归档文章须先恢复")
		}
		if r.ModuleCode != "" {
			m, e := ds.Modules().GetByCode(r.ModuleCode)
			if e != nil {
				return e
			}
			if m == nil || m.ID != p.Module.ID {
				return invalid("章节不属于当前模块")
			}
		}
		if identity != nil {
			if err := checkPendingSourceOwner(ds, *identity); err != nil {
				return err
			}
			duplicate, err := findSource(ds, identity.SourceKey)
			if err != nil {
				return err
			}
			if duplicate != nil && duplicate.ID != id {
				return conflict("该外部文档已被收录")
			}
		} else if strings.TrimSpace(a.ExternalLink) != "" {
			return invalid("已登记的外部来源不能清空")
		}
		if a.Status == model.ArticleStatusPublished {
			if e = catalog.RequireVisible(p); e != nil {
				return e
			}
		}
		changed := a.SectionCode != p.Section.Code || (p.Subsection == nil && a.SubsectionCode != "") || (p.Subsection != nil && a.SubsectionCode != p.Subsection.Code)
		a.Title = r.Title
		a.Author = r.Author
		if e = model.SetArticleTags(a, r.Tags); e != nil {
			return e
		}
		a.ExternalLink = r.ExternalLink
		if r.Content != nil {
			a.Content = *r.Content
		}
		canonicalPlacement(a, p)
		if identity != nil {
			if e = bindParsedIdentity(ds, a, *identity); e != nil {
				return e
			}
		}
		if changed {
			a.Pos, e = nextPos(ds, a.SectionCode, a.SubsectionCode)
			if e != nil {
				return e
			}
		}
		result = a
		return save(ds, a)
	})
	if e != nil {
		return nil, e
	}
	info, e := b.transform(ctx, result, false)
	return &v1.ArticleInfoResponse{Article: info}, e
}
func (b *articleBiz) changeStatus(ctx context.Context, id uint64, status int) error {
	if id == 0 || id > math.MaxInt64 {
		return invalid("文章ID无效")
	}
	return b.withLocked(ctx, id, "", "", func(ds store.IStore, a *model.Article, p *catalog.Placement) error {
		if e := checkManualWritable(ds, a); e != nil {
			return e
		}
		if status == model.ArticleStatusDraft && a.Status != model.ArticleStatusDeleted {
			return conflict("只能恢复归档文章")
		}
		if err := validatePublication(a, p, status); err != nil {
			return err
		}
		a.Status = status
		return save(ds, a)
	})
}
func (b *articleBiz) Publish(ctx context.Context, id uint64) error {
	return b.changeStatus(ctx, id, model.ArticleStatusPublished)
}
func (b *articleBiz) Unpublish(ctx context.Context, id uint64) error {
	return b.changeStatus(ctx, id, model.ArticleStatusUnpublished)
}
func (b *articleBiz) Archive(ctx context.Context, id uint64) (*v1.ArticleCommandResponse, error) {
	if e := b.changeStatus(ctx, id, model.ArticleStatusDeleted); e != nil {
		return nil, e
	}
	return b.commandResponse(ctx, id)
}
func (b *articleBiz) Restore(ctx context.Context, id uint64) (*v1.ArticleCommandResponse, error) {
	if e := b.changeStatus(ctx, id, model.ArticleStatusDraft); e != nil {
		return nil, e
	}
	return b.commandResponse(ctx, id)
}
func (b *articleBiz) MoveArticle(ctx context.Context, id uint64, r MoveInput) (*v1.ArticleCommandResponse, error) {
	if id == 0 || id > math.MaxInt64 {
		return nil, invalid("文章ID无效")
	}
	if r.SectionCode == "" {
		return nil, invalid("必须指定目标章节")
	}
	e := b.withLocked(ctx, id, r.SectionCode, r.SubsectionCode, func(ds store.IStore, a *model.Article, p *catalog.Placement) error {
		if e := checkManualWritable(ds, a); e != nil {
			return e
		}
		if a.Status == model.ArticleStatusDeleted {
			return conflict("归档文章须先恢复")
		}
		if a.Status == model.ArticleStatusPublished {
			if e := catalog.RequireVisible(p); e != nil {
				return e
			}
		}
		if a.SectionCode == p.Section.Code && ((p.Subsection == nil && a.SubsectionCode == "") || (p.Subsection != nil && a.SubsectionCode == p.Subsection.Code)) {
			return nil
		}
		canonicalPlacement(a, p)
		var e error
		a.Pos, e = nextPos(ds, a.SectionCode, a.SubsectionCode)
		if e != nil {
			return e
		}
		return save(ds, a)
	})
	if e != nil {
		return nil, e
	}
	return b.commandResponse(ctx, id)
}
func (b *articleBiz) ReorderArticles(ctx context.Context, r ReorderInput) error {
	return store.InTransaction(ctx, b.ds, func(ds store.IStore) error {
		p, e := catalog.Resolve(ds, r.SectionCode, r.SubsectionCode)
		if e != nil {
			return e
		}
		if e = catalog.LockModules(ds, p.Module.Code); e != nil {
			return e
		}
		p, e = catalog.Resolve(ds, r.SectionCode, r.SubsectionCode)
		if e != nil {
			return e
		}
		sub := ""
		if p.Subsection != nil {
			sub = p.Subsection.Code
		}
		q := store.FilterArticles(ds.DB(), store.ArticleFilter{SectionCode: p.Section.Code, SubsectionCode: sub, DirectOnly: sub == ""})
		var actual []*model.Article
		if e = q.Where("article.status <> ? OR article.status IS NULL", model.ArticleStatusDeleted).Find(&actual).Error; e != nil {
			return e
		}
		requested := map[uint64]int{}
		for i, id := range r.ArticleIDs {
			if id == 0 || id > math.MaxInt64 {
				return invalid("文章ID无效")
			}
			if _, ok := requested[id]; ok {
				return invalid("排序ID不能重复")
			}
			requested[id] = i + 1
		}
		if len(actual) != len(requested) {
			return conflict("请提交当前位置全部未归档文章")
		}
		for _, a := range actual {
			pos, ok := requested[a.ID]
			if !ok {
				return conflict("排序集合已变更，请刷新")
			}
			a.Pos = pos
			if e = save(ds, a); e != nil {
				return e
			}
		}
		return nil
	})
}
func parseStatus(raw string) (int, error) {
	switch strings.ToLower(raw) {
	case "":
		return 0, nil
	case "draft", "1":
		return 1, nil
	case "published", "2":
		return 2, nil
	case "unpublished", "3":
		return 3, nil
	case "deleted", "archived", "4":
		return 4, nil
	}
	return 0, invalid("文章状态无效")
}
func (b *articleBiz) GetList(ctx context.Context, r *v1.ArticleListRequest) (*v1.GetArticleListResponse, error) {
	if r.Page < 0 || r.Limit < 0 || r.Limit > 100 {
		return nil, invalid("分页参数无效")
	}
	page, limit := r.Page, r.Limit
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	if page-1 > math.MaxInt/limit {
		return nil, invalid("分页参数无效")
	}
	status, e := parseStatus(r.Status)
	if e != nil {
		return nil, e
	}
	rows, total, e := store.ListArticles(ctx, b.ds.DB(), store.ArticleFilter{ModuleCode: r.ModuleCode, SectionCode: r.SectionCode, SubsectionCode: r.SubsectionCode, DirectOnly: r.DirectOnly, Title: r.Title, Status: status, Page: page, Limit: limit})
	if e != nil {
		return nil, e
	}
	associations, e := store.LoadArticleAssociations(ctx, b.ds.DB(), rows)
	if e != nil {
		return nil, e
	}
	out := &v1.GetArticleListResponse{Articles: make([]*v1.ArticleInfo, 0, len(rows)), Total: total}
	for _, a := range rows {
		info, e := articleInfo(a, associations, true)
		if e != nil {
			return nil, e
		}
		out.Articles = append(out.Articles, info)
	}
	return out, nil
}
func (b *articleBiz) GetOne(ctx context.Context, id uint64) (*v1.GetArticleResponse, error) {
	a, e := b.contextStore(ctx).Articles().GetOne(id)
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, errno.ErrArticleNotFound
	}
	if e != nil {
		return nil, e
	}
	info, e := b.transform(ctx, a, false)
	return &v1.GetArticleResponse{Article: info}, e
}
func (b *articleBiz) transform(ctx context.Context, a *model.Article, simple bool) (*v1.ArticleInfo, error) {
	associations, e := store.LoadArticleAssociations(ctx, b.ds.DB(), []*model.Article{a})
	if e != nil {
		return nil, e
	}
	return articleInfo(a, associations, simple)
}
func articleInfo(a *model.Article, associations *store.ArticleAssociations, simple bool) (*v1.ArticleInfo, error) {
	s := associations.Sections[a.SectionCode]
	if s == nil {
		return nil, errno.ErrSectionNotFound
	}
	m := associations.Modules[s.ModuleCode]
	if m == nil {
		return nil, errno.ErrModuleNotFound
	}
	tags, err := model.ArticleTags(a)
	if err != nil {
		return nil, err
	}
	out := &v1.ArticleInfo{ID: a.ID, IDText: strconv.FormatUint(a.ID, 10), Title: a.Title, Module: v1.ModuleInfo{ID: int(m.ID), Code: m.Code, Title: m.Title, Status: m.Status, Sort: m.Sort}, Section: v1.SectionInfo{Code: s.Code, Title: s.Title, ModuleCode: m.Code, Sort: s.Sort, Status: s.Status}, Author: a.Author, Tags: tags, Pos: a.Pos, Status: a.GetStatusString(), ExternalLink: a.ExternalLink, Provider: a.Provider, CanonicalURL: a.CanonicalURL}
	if a.SubsectionCode != "" {
		sub := associations.Subsections[a.SubsectionCode]
		if sub == nil {
			return nil, errno.ErrSubsectionNotFound
		}
		out.Subsection = &v1.SubsectionInfo{Code: sub.Code, Title: sub.Title, SectionCode: s.Code, Sort: sub.Sort, Status: sub.Status}
	}
	decorateManagement(out, a, associations)
	if !simple {
		out.Content = a.Content
		out.CreatedAt = a.CreatedAt.Format("2006-01-02 15:04:05")
		out.UpdatedAt = a.UpdatedAt.Format("2006-01-02 15:04:05")
	}
	return out, nil
}
func (b *articleBiz) commandDTO(ctx context.Context, a *model.Article) (*v1.ArticleDTO, error) {
	info, e := b.transform(ctx, a, false)
	if e != nil {
		return nil, e
	}
	return &v1.ArticleDTO{ArticleInfo: info, ID: info.IDText}, nil
}
func (b *articleBiz) commandResponse(ctx context.Context, id uint64) (*v1.ArticleCommandResponse, error) {
	a, e := b.contextStore(ctx).Articles().GetOne(id)
	if e != nil {
		return nil, e
	}
	dto, e := b.commandDTO(ctx, a)
	return &v1.ArticleCommandResponse{Article: dto}, e
}
