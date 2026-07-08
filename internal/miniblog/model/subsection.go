package model

import (
	"time"

	"github.com/yshujie/miniblog/pkg/util/idutil"
	"gorm.io/gorm"
)

// Subsection 子章节（section 下的分类）
type Subsection struct {
	ID          uint64    `json:"id"`
	Code        string    `json:"code"`
	Title       string    `json:"title"`
	Sort        int       `json:"sort"`
	SectionCode string    `json:"section_code"`
	Status      int       `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

const (
	SubsectionStatusNormal = iota + 1
	SubsectionStatusDeleted
)

func (s *Subsection) TableName() string {
	return "subsection"
}

func (s *Subsection) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = idutil.GetIntID()
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	if s.Status == 0 {
		s.Status = SubsectionStatusNormal
	}
	return
}

func (s *Subsection) BeforeUpdate(tx *gorm.DB) (err error) {
	s.UpdatedAt = time.Now()
	return
}

func (s *Subsection) Publish() {
	s.Status = SubsectionStatusNormal
}

func (s *Subsection) Unpublish() {
	s.Status = SubsectionStatusDeleted
}
