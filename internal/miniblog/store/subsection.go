package store

import (
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
)

type SubsectionStore interface {
	Create(subsection *model.Subsection) error
	GetByCode(code string) (*model.Subsection, error)
	GetSubsections(sectionCode string) ([]*model.Subsection, error)
	GetNormalSubsections(sectionCode string) ([]*model.Subsection, error)
	Update(subsection *model.Subsection) error
	DeleteByCode(code string) error
}

type subsections struct {
	db *gorm.DB
}

var _ SubsectionStore = &subsections{}

func newSubsections(db *gorm.DB) *subsections {
	return &subsections{db}
}

func (s *subsections) Create(subsection *model.Subsection) error {
	return s.db.Create(subsection).Error
}

func (s *subsections) GetByCode(code string) (*model.Subsection, error) {
	var subsection model.Subsection
	if err := s.db.Where("code = ?", code).First(&subsection).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &subsection, nil
}

func (s *subsections) GetSubsections(sectionCode string) ([]*model.Subsection, error) {
	var items []*model.Subsection
	return items, s.db.Where("section_code = ?", sectionCode).Order("sort asc").Find(&items).Error
}

func (s *subsections) GetNormalSubsections(sectionCode string) ([]*model.Subsection, error) {
	var items []*model.Subsection
	return items, s.db.Where("section_code = ?", sectionCode).
		Where("status = ?", model.SubsectionStatusNormal).
		Order("sort asc").
		Find(&items).Error
}

func (s *subsections) Update(subsection *model.Subsection) error {
	return s.db.Save(subsection).Error
}

func (s *subsections) DeleteByCode(code string) error {
	return s.db.Where("code = ?", code).Delete(&model.Subsection{}).Error
}
