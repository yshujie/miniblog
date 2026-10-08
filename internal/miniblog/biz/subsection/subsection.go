package subsection

import (
	"context"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

type ISubsectionBiz interface {
	Create(context.Context, *v1.CreateSubsectionRequest) (*v1.CreateSubsectionResponse, error)
	Update(context.Context, string, *v1.UpdateSubsectionRequest) (*v1.UpdateSubsectionResponse, error)
	Publish(context.Context, string) (*v1.SubsectionStatusResponse, error)
	Unpublish(context.Context, string) (*v1.SubsectionStatusResponse, error)
	GetList(context.Context, string) (*v1.GetSubsectionListResponse, error)
	GetOne(context.Context, string) (*v1.GetSubsectionResponse, error)
	Delete(context.Context, string) error
}
type subsectionBiz struct{ ds store.IStore }

func New(ds store.IStore) *subsectionBiz { return &subsectionBiz{ds: ds} }
func (b *subsectionBiz) contextStore(ctx context.Context) store.IStore {
	if b.ds.DB() == nil {
		return b.ds
	}
	return store.NewStore(b.ds.DB().WithContext(ctx))
}
func (b *subsectionBiz) Create(ctx context.Context, r *v1.CreateSubsectionRequest) (*v1.CreateSubsectionResponse, error) {
	m, e := catalog.New(b.ds).CreateSubsection(ctx, catalog.SubsectionInput{Code: r.Code, Title: r.Title, SectionCode: r.SectionCode, Sort: r.Sort})
	if e != nil {
		return nil, e
	}
	return &v1.CreateSubsectionResponse{Subsection: toSubsectionInfo(m)}, nil
}
func (b *subsectionBiz) Update(ctx context.Context, code string, r *v1.UpdateSubsectionRequest) (*v1.UpdateSubsectionResponse, error) {
	m, e := catalog.New(b.ds).UpdateSubsection(ctx, code, catalog.UpdateInput{Title: r.Title, Sort: r.Sort})
	if e != nil {
		return nil, e
	}
	return &v1.UpdateSubsectionResponse{Subsection: toSubsectionInfo(m)}, nil
}
func (b *subsectionBiz) Publish(ctx context.Context, code string) (*v1.SubsectionStatusResponse, error) {
	m, e := catalog.New(b.ds).SubsectionStatus(ctx, code, model.SubsectionStatusNormal)
	if e != nil {
		return nil, e
	}
	return &v1.SubsectionStatusResponse{Subsection: toSubsectionInfo(m)}, nil
}
func (b *subsectionBiz) Unpublish(ctx context.Context, code string) (*v1.SubsectionStatusResponse, error) {
	m, e := catalog.New(b.ds).SubsectionStatus(ctx, code, model.SubsectionStatusDeleted)
	if e != nil {
		return nil, e
	}
	return &v1.SubsectionStatusResponse{Subsection: toSubsectionInfo(m)}, nil
}
func (b *subsectionBiz) Delete(ctx context.Context, code string) error {
	return catalog.New(b.ds).DeleteSubsection(ctx, code)
}
func (b *subsectionBiz) GetList(ctx context.Context, parent string) (*v1.GetSubsectionListResponse, error) {
	rows, e := b.contextStore(ctx).Subsections().GetSubsections(parent)
	if e != nil {
		return nil, e
	}
	out := &v1.GetSubsectionListResponse{Subsections: make([]*v1.SubsectionInfo, 0, len(rows))}
	for _, m := range rows {
		out.Subsections = append(out.Subsections, toSubsectionInfo(m))
	}
	return out, nil
}
func (b *subsectionBiz) GetOne(ctx context.Context, code string) (*v1.GetSubsectionResponse, error) {
	m, e := b.contextStore(ctx).Subsections().GetByCode(code)
	if e != nil {
		return nil, e
	}
	if m == nil {
		return nil, errno.ErrSubsectionNotFound
	}
	return &v1.GetSubsectionResponse{Subsection: toSubsectionInfo(m)}, nil
}
func toSubsectionInfo(m *model.Subsection) *v1.SubsectionInfo {
	if m == nil {
		return nil
	}
	return &v1.SubsectionInfo{Code: m.Code, Title: m.Title, SectionCode: m.SectionCode, Sort: m.Sort, Status: m.Status}
}
