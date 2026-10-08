package section

import (
	"context"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

type ISectionBiz interface {
	Create(context.Context, *v1.CreateSectionRequest) (*v1.CreateSectionResponse, error)
	Update(context.Context, string, *v1.UpdateSectionRequest) (*v1.UpdateSectionResponse, error)
	Publish(context.Context, string) (*v1.SectionStatusResponse, error)
	Unpublish(context.Context, string) (*v1.SectionStatusResponse, error)
	GetList(context.Context, string) (*v1.GetSectionListResponse, error)
	GetOne(context.Context, string) (*v1.GetSectionResponse, error)
	Delete(context.Context, string) error
}
type sectionBiz struct{ ds store.IStore }

func New(ds store.IStore) *sectionBiz { return &sectionBiz{ds: ds} }
func (b *sectionBiz) contextStore(ctx context.Context) store.IStore {
	if b.ds.DB() == nil {
		return b.ds
	}
	return store.NewStore(b.ds.DB().WithContext(ctx))
}
func (b *sectionBiz) Create(ctx context.Context, r *v1.CreateSectionRequest) (*v1.CreateSectionResponse, error) {
	m, e := catalog.New(b.ds).CreateSection(ctx, catalog.SectionInput{Code: r.Code, Title: r.Title, ModuleCode: r.ModuleCode, Sort: r.Sort})
	if e != nil {
		return nil, e
	}
	return &v1.CreateSectionResponse{Section: toSectionInfo(m)}, nil
}
func (b *sectionBiz) Update(ctx context.Context, code string, r *v1.UpdateSectionRequest) (*v1.UpdateSectionResponse, error) {
	m, e := catalog.New(b.ds).UpdateSection(ctx, code, catalog.UpdateInput{Title: r.Title, Sort: r.Sort})
	if e != nil {
		return nil, e
	}
	return &v1.UpdateSectionResponse{Section: toSectionInfo(m)}, nil
}
func (b *sectionBiz) Publish(ctx context.Context, code string) (*v1.SectionStatusResponse, error) {
	m, e := catalog.New(b.ds).SectionStatus(ctx, code, model.SectionStatusNormal)
	if e != nil {
		return nil, e
	}
	return &v1.SectionStatusResponse{Section: toSectionInfo(m)}, nil
}
func (b *sectionBiz) Unpublish(ctx context.Context, code string) (*v1.SectionStatusResponse, error) {
	m, e := catalog.New(b.ds).SectionStatus(ctx, code, model.SectionStatusDeleted)
	if e != nil {
		return nil, e
	}
	return &v1.SectionStatusResponse{Section: toSectionInfo(m)}, nil
}
func (b *sectionBiz) Delete(ctx context.Context, code string) error {
	return catalog.New(b.ds).DeleteSection(ctx, code)
}
func (b *sectionBiz) GetList(ctx context.Context, parent string) (*v1.GetSectionListResponse, error) {
	rows, e := b.contextStore(ctx).Sections().GetSections(parent)
	if e != nil {
		return nil, e
	}
	out := &v1.GetSectionListResponse{Sections: make([]*v1.SectionInfo, 0, len(rows))}
	for _, m := range rows {
		out.Sections = append(out.Sections, toSectionInfo(m))
	}
	return out, nil
}
func (b *sectionBiz) GetOne(ctx context.Context, code string) (*v1.GetSectionResponse, error) {
	m, e := b.contextStore(ctx).Sections().GetByCode(code)
	if e != nil {
		return nil, e
	}
	if m == nil {
		return nil, errno.ErrSectionNotFound
	}
	return &v1.GetSectionResponse{Section: toSectionInfo(m)}, nil
}
func toSectionInfo(m *model.Section) *v1.SectionInfo {
	if m == nil {
		return nil
	}
	return &v1.SectionInfo{Code: m.Code, Title: m.Title, ModuleCode: m.ModuleCode, Sort: m.Sort, Status: m.Status}
}
