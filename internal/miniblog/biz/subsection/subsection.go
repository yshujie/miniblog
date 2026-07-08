package subsection

import (
	"context"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

type ISubsectionBiz interface {
	Create(ctx context.Context, r *v1.CreateSubsectionRequest) (*v1.CreateSubsectionResponse, error)
	Update(ctx context.Context, code string, r *v1.UpdateSubsectionRequest) (*v1.UpdateSubsectionResponse, error)
	Publish(ctx context.Context, code string) (*v1.SubsectionStatusResponse, error)
	Unpublish(ctx context.Context, code string) (*v1.SubsectionStatusResponse, error)
	GetList(ctx context.Context, sectionCode string) (*v1.GetSubsectionListResponse, error)
	GetOne(ctx context.Context, code string) (*v1.GetSubsectionResponse, error)
	Delete(ctx context.Context, code string) error
}

type subsectionBiz struct {
	ds store.IStore
}

var _ ISubsectionBiz = (*subsectionBiz)(nil)

func New(ds store.IStore) *subsectionBiz {
	return &subsectionBiz{ds}
}

func (b *subsectionBiz) Create(ctx context.Context, r *v1.CreateSubsectionRequest) (*v1.CreateSubsectionResponse, error) {
	existing, err := b.ds.Subsections().GetByCode(r.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errno.ErrSubsectionAlreadyExists
	}

	section, err := b.ds.Sections().GetByCode(r.SectionCode)
	if err != nil {
		return nil, err
	}
	if section == nil {
		return nil, errno.ErrSectionNotFound
	}

	subsection := &model.Subsection{
		Code:        r.Code,
		Title:       r.Title,
		SectionCode: r.SectionCode,
	}
	if r.Sort != nil {
		subsection.Sort = *r.Sort
	}
	if err = b.ds.Subsections().Create(subsection); err != nil {
		return nil, err
	}

	return &v1.CreateSubsectionResponse{Subsection: toSubsectionInfo(subsection)}, nil
}

func (b *subsectionBiz) Update(ctx context.Context, code string, r *v1.UpdateSubsectionRequest) (*v1.UpdateSubsectionResponse, error) {
	subsection, err := b.ds.Subsections().GetByCode(code)
	if err != nil {
		return nil, err
	}
	if subsection == nil {
		return nil, errno.ErrSubsectionNotFound
	}

	subsection.Title = r.Title
	if r.Sort != nil {
		subsection.Sort = *r.Sort
	}
	if err = b.ds.Subsections().Update(subsection); err != nil {
		return nil, err
	}

	return &v1.UpdateSubsectionResponse{Subsection: toSubsectionInfo(subsection)}, nil
}

func (b *subsectionBiz) Publish(ctx context.Context, code string) (*v1.SubsectionStatusResponse, error) {
	subsection, err := b.ds.Subsections().GetByCode(code)
	if err != nil {
		return nil, err
	}
	if subsection == nil {
		return nil, errno.ErrSubsectionNotFound
	}

	subsection.Publish()
	if err = b.ds.Subsections().Update(subsection); err != nil {
		return nil, err
	}

	return &v1.SubsectionStatusResponse{Subsection: toSubsectionInfo(subsection)}, nil
}

func (b *subsectionBiz) Unpublish(ctx context.Context, code string) (*v1.SubsectionStatusResponse, error) {
	subsection, err := b.ds.Subsections().GetByCode(code)
	if err != nil {
		return nil, err
	}
	if subsection == nil {
		return nil, errno.ErrSubsectionNotFound
	}

	subsection.Unpublish()
	if err = b.ds.Subsections().Update(subsection); err != nil {
		return nil, err
	}

	return &v1.SubsectionStatusResponse{Subsection: toSubsectionInfo(subsection)}, nil
}

func (b *subsectionBiz) GetList(ctx context.Context, sectionCode string) (*v1.GetSubsectionListResponse, error) {
	items, err := b.ds.Subsections().GetSubsections(sectionCode)
	if err != nil {
		return nil, err
	}

	response := &v1.GetSubsectionListResponse{
		Subsections: make([]*v1.SubsectionInfo, 0, len(items)),
	}
	for _, item := range items {
		response.Subsections = append(response.Subsections, toSubsectionInfo(item))
	}
	return response, nil
}

func (b *subsectionBiz) GetOne(ctx context.Context, code string) (*v1.GetSubsectionResponse, error) {
	subsection, err := b.ds.Subsections().GetByCode(code)
	if err != nil {
		return nil, err
	}
	if subsection == nil {
		return nil, errno.ErrSubsectionNotFound
	}

	return &v1.GetSubsectionResponse{Subsection: toSubsectionInfo(subsection)}, nil
}

func (b *subsectionBiz) Delete(ctx context.Context, code string) error {
	subsection, err := b.ds.Subsections().GetByCode(code)
	if err != nil {
		return err
	}
	if subsection == nil {
		return errno.ErrSubsectionNotFound
	}

	filter := map[string]interface{}{"subsection_code": code}
	articles, err := b.ds.Articles().GetList(filter, 1, 1)
	if err != nil {
		return err
	}
	if len(articles) > 0 {
		return errno.ErrSubsectionHasArticles
	}

	return b.ds.Subsections().DeleteByCode(code)
}

func toSubsectionInfo(subsection *model.Subsection) *v1.SubsectionInfo {
	if subsection == nil {
		return nil
	}
	return &v1.SubsectionInfo{
		Code:        subsection.Code,
		Title:       subsection.Title,
		SectionCode: subsection.SectionCode,
		Sort:        subsection.Sort,
		Status:      subsection.Status,
	}
}
